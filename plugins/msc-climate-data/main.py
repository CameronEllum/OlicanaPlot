"""MSC Climate Data Plugin - IPC Handler."""

from __future__ import annotations

import datetime
from dataclasses import dataclass, field

import ipc_helpers
import msc_api


@dataclass
class PluginState:
    """Mutable plugin state."""

    station: msc_api.Station | None = None
    start_date: str = ""
    end_date: str = ""
    observations: list[msc_api.DailyObservation] = field(default_factory=list)


state = PluginState()


def handle_initialize(args: str) -> None:
    """Multi-step discovery: 1. Map & Dates -> 2. Station Selection."""
    today = datetime.date.today()
    start_default = today - datetime.timedelta(days=395)
    end_default = today - datetime.timedelta(days=30)

    # Discovery state
    step = 1
    coords = {"lat": 51.0447, "lng": -114.0719}  # Default to Calgary
    start_date = start_default.isoformat()
    end_date = end_default.isoformat()
    discovered_stations: list[msc_api.Station] = []

    while True:
        if step == 1:
            schema = {
                "type": "object",
                "title": "MSC Climate Discovery",
                "required": ["location", "startDate", "endDate"],
                "properties": {
                    "location": {
                        "type": "object",
                        "title": "1. Pick a location on the map",
                        "properties": {"lat": {"type": "number"}, "lng": {"type": "number"}},
                    },
                    "startDate": {"type": "string", "title": "2. Start Date", "format": "date"},
                    "endDate": {"type": "string", "title": "3. End Date", "format": "date"},
                },
            }
            ui = {
                "ui:order": ["location", "startDate", "endDate"],
                "location": {"ui:widget": "map-picker"},
                "startDate": {"ui:widget": "date"},
                "endDate": {"ui:widget": "date"},
            }
            data = {"location": coords, "startDate": start_date, "endDate": end_date}
            ipc_helpers.send_show_form("MSC Climate Discovery - Step 1/2", schema, ui, data)

        elif step == 2:
            schema = {
                "type": "object",
                "title": f"Discovery Results: {len(discovered_stations)} stations verified",
                "required": ["station"],
                "properties": {
                    "station": {
                        "type": "string",
                        "title": "4. Select verified station",
                        "oneOf": [
                            {"const": s.climate_id, "title": f"{s.name} ({s.climate_id})"}
                            for s in discovered_stations
                        ],
                    }
                },
            }
            ui = {"station": {"ui:widget": "select"}}
            data = {"station": discovered_stations[0].climate_id if discovered_stations else ""}
            ipc_helpers.send_show_form("MSC Climate Discovery - Step 2/2", schema, ui, data)

        req = ipc_helpers.read_request()
        if not req:
            break

        if "result" in req:
            res = req["result"]
            if step == 1:
                coords = res.get("location", coords)
                start_date = res.get("startDate", start_date)
                end_date = res.get("endDate", end_date)

                ipc_helpers.log(
                    "info", f"Verifying stations near {coords['lat']}, {coords['lng']}..."
                )
                try:
                    discovered_stations = msc_api.search_stations_bbox(
                        coords["lat"], coords["lng"], start_date, end_date
                    )
                    if not discovered_stations:
                        msg = (
                            f"No stations found with daily data near "
                            f"{coords['lat']:.2f}, {coords['lng']:.2f} for that period. "
                            "Please try another location or date range."
                        )
                        ipc_helpers.log("warn", msg)
                        ipc_helpers.send_error(msg)
                        continue

                    step = 2
                except Exception as e:
                    ipc_helpers.send_error(f"Discovery search failed: {e}")
                    continue

            elif step == 2:
                selected_id = res["station"]
                stn = next((s for s in discovered_stations if s.climate_id == selected_id), None)
                if not stn:
                    ipc_helpers.send_error("Selection lost. Please try again.")
                    step = 1
                    continue

                state.station = stn
                state.start_date = start_date
                state.end_date = end_date

                ipc_helpers.log("info", f"Selected {stn.name}. Fetching full data set...")
                try:
                    state.observations = msc_api.fetch_daily_data(selected_id, start_date, end_date)
                    if not state.observations:
                        ipc_helpers.send_error(
                            "Station verification failed: no actual observations returned."
                        )
                        step = 1
                        continue

                    ipc_helpers.log("info", f"Loaded {len(state.observations)} daily observations")
                    ipc_helpers.send_response({"result": "initialized"})
                    break
                except Exception as e:
                    ipc_helpers.send_error(f"Final data fetch failed: {e}")
                    step = 1
                    continue

        elif "error" in req:
            ipc_helpers.log("info", "Discovery cancelled")
            ipc_helpers.send_error("cancelled")
            break


def handle_get_chart_config() -> None:
    if not state.station:
        ipc_helpers.send_error("Not initialized")
        return

    config = {
        "title": f"{state.station.name} Temperature History",
        "axis_labels": ["Date", "Temperature (°C)"],
    }
    ipc_helpers.send_response({"result": config})


def handle_get_series_config() -> None:
    prefix = f"{state.station.name} " if state.station else ""
    series = [
        {"id": "mean_temp", "name": f"{prefix}Mean Temp", "color": ipc_helpers.CHART_COLORS[0]},
        {"id": "min_temp", "name": f"{prefix}Min Temp", "color": ipc_helpers.CHART_COLORS[2]},
        {"id": "max_temp", "name": f"{prefix}Max Temp", "color": ipc_helpers.CHART_COLORS[1]},
    ]
    ipc_helpers.send_response({"result": series})


def handle_get_series_data(series_id: str, preferred_storage: str) -> None:
    if not state.observations:
        ipc_helpers.send_error("No data available")
        return

    # Convert observations to X (timestamps) and Y (values)
    values: list[float] = []

    for obs in state.observations:
        try:
            # Parse date YYYY-MM-DD
            dt = datetime.datetime.strptime(obs.date, "%Y-%m-%d")
            timestamp = dt.timestamp()

            val = None
            if series_id == "mean_temp":
                val = obs.mean_temp
            elif series_id == "min_temp":
                val = obs.min_temp
            elif series_id == "max_temp":
                val = obs.max_temp

            if val is not None:
                if preferred_storage == "interleaved":
                    values.append(float(timestamp))
                    values.append(float(val))
                else:
                    # In this simple implementation, we assume interleaved for now
                    # as that's what we mostly use.
                    # TODO: handle 'arrays' if requested
                    values.append(float(timestamp))
                    values.append(float(val))
        except Exception:
            continue

    ipc_helpers.send_binary_data(values, "interleaved")


def main() -> None:
    while True:
        req = ipc_helpers.read_request()
        if not req:
            break

        method = req.get("method")
        if method == "info":
            ipc_helpers.send_response({"name": "MSC Climate Data", "version": 1})
        elif method == "initialize":
            handle_initialize(req.get("args", ""))
        elif method == "get_chart_config":
            handle_get_chart_config()
        elif method == "get_series_config":
            handle_get_series_config()
        elif method == "get_series_data":
            handle_get_series_data(
                req.get("series_id", ""), req.get("preferred_storage", "interleaved")
            )
        else:
            ipc_helpers.send_error(f"Unknown method: {method}")


if __name__ == "__main__":
    main()
