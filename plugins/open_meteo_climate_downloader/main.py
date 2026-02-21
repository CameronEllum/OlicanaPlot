import datetime
import os
import sys
from dataclasses import dataclass, field
from typing import Any

import polars as pl

# Add SDK path to sys.path
sdk_path = os.path.abspath(
    os.path.join(os.path.dirname(__file__), "../../sdk/python")
)
if sdk_path not in sys.path:
    sys.path.append(sdk_path)

import protocol  # noqa: E402

import history  # noqa: E402
from cache import ClimateCache  # noqa: E402
from open_meteo import fetch_historical, geocode_city  # noqa: E402


@dataclass
class PluginState:
    """Mutable plugin state."""

    cities: list[str] = field(default_factory=list)
    start_date: str = ""
    end_date: str = ""
    mode: str = "Daily"
    data: dict[str, pl.DataFrame] = field(default_factory=dict)


state = PluginState()
cache = ClimateCache()


def get_history_options() -> tuple[list[str], dict[str, dict[str, Any]]]:
    """Retrieve history options formatted for UI dropdown."""
    past_searches = history.get_past_searches()
    options = ["New Search"]
    search_maps = {}
    for s in past_searches:
        c_str = ", ".join(s["cities"])
        label = f"{c_str} ({s['start_date']} to {s['end_date']})"
        if label not in options:
            options.append(label)
            search_maps[label] = s
    return options, search_maps


def build_schema(
    search_options: list[str],
) -> tuple[dict[str, Any], dict[str, Any]]:
    """Construct JSON Schema and UI Schema for the configuration form."""
    today = datetime.date.today()
    schema = {
        "type": "object",
        "title": "Open Meteo Setup",
        "required": ["search_type", "cities", "start_date", "end_date"],
        "properties": {
            "search_type": {
                "type": "string",
                "title": "Search History",
                "enum": search_options,
                "default": "New Search",
            },
            "cities": {
                "type": "array",
                "title": "Locations (Add to list)",
                "items": {"type": "string", "title": "City Name"},
                "minItems": 1,
            },
            "map_markers": {
                "type": "array",
                "title": "Map Preview",
                "items": {"type": "object"},
            },
            "start_date": {
                "type": "string",
                "title": "Start Date",
                "format": "date",
                "maximum": str(today),
            },
            "end_date": {
                "type": "string",
                "title": "End Date",
                "format": "date",
                "maximum": str(today),
            },
            "mode": {
                "type": "string",
                "title": "Data Mode",
                "enum": [
                    "Daily",
                    "Daily Means",
                    "Daily Means (smoothed)",
                    "Monthly",
                    "Monthly Means",
                ],
                "default": "Daily",
            },
        },
    }
    ui = {
        "ui:order": [
            "search_type",
            "start_date",
            "end_date",
            "cities",
            "map_markers",
            "mode",
        ],
        "ui:classNames": "grid grid-cols-2 gap-4",
        "search_type": {"ui:classNames": "col-span-2"},
        "cities": {"ui:classNames": "col-span-2"},
        "map_markers": {
            "ui:widget": "map-picker",
            "ui:classNames": "col-span-2",
        },
        "mode": {"ui:classNames": "col-span-2"},
        "start_date": {
            "ui:widget": "date",
            "ui:classNames": "col-span-1",
        },
        "end_date": {
            "ui:widget": "date",
            "ui:classNames": "col-span-1",
        },
    }

    if len(search_options) <= 1:
        schema["properties"]["search_type"]["ui:hidden"] = True
        ui["search_type"]["ui:hidden"] = True

    return schema, ui


def geocode_city_list(cities: list[str]) -> tuple[list[dict[str, Any]], bool]:
    """Geocode a string list of cities into a list of map markers."""
    markers = []
    all_success = True
    for city in cities:
        city = city.strip()
        if not city:
            continue
        loc = cache.get_city_location(city)
        if not loc:
            loc = geocode_city(city)
            if loc:
                cache.save_city(city, loc[0], loc[1])
        if loc:
            markers.append({"lat": loc[0], "lng": loc[1], "popup": city})
        else:
            all_success = False
    return markers, all_success


def load_city_data(city: str, start_date: str, end_date: str) -> bool:
    """Load climate data for a single city from cache or remote API."""
    if cache.has_date_range(city, start_date, end_date):
        protocol.log("info", f"Using cached data for {city}")
        state.data[city] = cache.get_daily_data(city, start_date, end_date)
        return True

    protocol.log("info", f"Geocoding {city}...")
    loc = cache.get_city_location(city)
    if not loc:
        loc = geocode_city(city)
        if loc:
            cache.save_city(city, loc[0], loc[1])

    if not loc:
        protocol.send_error(f"Could not calculate coordinates for {city}")
        return False

    lat, lng = loc
    protocol.log("info", f"Fetching open-meteo for {city}...")
    try:
        df_new = fetch_historical(lat, lng, start_date, end_date)
        if not df_new.is_empty():
            cache.save_daily(city, df_new)
            state.data[city] = cache.get_daily_data(city, start_date, end_date)
        else:
            protocol.log("warn", f"No data returned for {city}")
        return True
    except Exception as e:
        protocol.send_error(f"Failed fetching {city}: {e}")
        return False


def process_form_change(
    req: dict[str, Any],
    last_cities: list[str],
    last_search: str,
    search_maps: dict[str, dict[str, Any]],
) -> tuple[bool, list[str], str]:
    """Handle live form updates and coordinate geocoding for UI previews."""
    new_data = req.get("data", {})
    curr_cities = new_data.get("cities", [])
    curr_search = new_data.get("search_type", "New Search")
    needs_update = False

    if curr_search != last_search:
        last_search = curr_search
        if curr_search != "New Search" and curr_search in search_maps:
            s = search_maps[curr_search]
            new_data.update(
                {
                    "cities": s["cities"],
                    "start_date": s["start_date"],
                    "end_date": s["end_date"],
                }
            )
            curr_cities = s["cities"]
            needs_update = True

    if curr_cities != last_cities:
        last_cities = list(curr_cities)
        markers, _ = geocode_city_list(curr_cities)
        new_data["map_markers"] = markers
        needs_update = True

    if needs_update:
        protocol.send_form_update(data=new_data)
    else:
        protocol.send_response({})

    return needs_update, last_cities, last_search


def process_form_submit(
    res: dict[str, Any], def_start: str, def_end: str
) -> bool:
    """Validate submitted form parameters and trigger batch data loading."""
    cities = res.get("cities", [])
    start_date = res.get("start_date", def_start)
    end_date = res.get("end_date", def_end)
    mode = res.get("mode", "Daily")

    if start_date >= end_date:
        protocol.send_error("Start date must be before end date.")
        return False

    cities = [c.strip() for c in cities if c.strip()]
    if not cities:
        protocol.send_error("At least one valid city is required.")
        return False

    protocol.log("info", f"Processing: {', '.join(cities)}")
    for city in cities:
        if not load_city_data(city, start_date, end_date):
            return False

    history.log_search(cities, start_date, end_date)
    state.cities = cities
    state.start_date = start_date
    state.end_date = end_date
    state.mode = mode
    return True


def handle_initialize(args: str) -> None:
    """Implement the interactive map initialization form."""
    today = datetime.date.today()
    default_start = (today - datetime.timedelta(days=365 * 30)).strftime(
        "%Y-%m-%d"
    )
    default_end = today.strftime("%Y-%m-%d")

    data = {
        "search_type": "New Search",
        "cities": ["Calgary"],
        "start_date": default_start,
        "end_date": default_end,
        "map_markers": [],
        "mode": "Daily",
    }

    # Setup initial markers payload
    initial_markers, _ = geocode_city_list(data["cities"])
    data["map_markers"] = initial_markers

    search_opts, search_maps = get_history_options()
    schema, ui = build_schema(search_opts)

    protocol.send_show_form(
        "Open Meteo Configuration",
        schema,
        ui,
        data,
        handle_form_change=True,
    )

    last_cities = list(data["cities"])
    last_search_type = data["search_type"]

    while True:
        req = protocol.read_request()
        if not req:
            break

        method = req.get("method")
        if method == "form_change":
            _, last_cities, last_search_type = process_form_change(
                req, last_cities, last_search_type, search_maps
            )
            continue

        if "result" in req:
            if process_form_submit(req["result"], default_start, default_end):
                protocol.log(
                    "info", f"Loaded data for {len(state.data)} cities."
                )
                protocol.send_response({"result": "initialized"})
                break
            continue

        if "error" in req:
            protocol.log("info", "Configuration cancelled")
            protocol.send_error("cancelled")
            break


def handle_get_chart_config() -> None:
    """Provide configuration for rendering the chart UI elements."""
    if not state.cities:
        protocol.send_error("Not initialized")
        return

    title = "Climate Comparison"
    if len(state.cities) == 1:
        title = f"{state.cities[0]} Temperature History"
    elif len(state.cities) <= 3:
        title = f"{', '.join(state.cities)} Temperature History"

    config = {
        "title": title,
        "axes": [
            {
                "subplot": [0, 0],
                "x_axes": [{"title": "Date", "type": "date"}],
                "y_axes": [{"title": "Temperature (°C)"}],
            }
        ],
    }
    protocol.send_response({"result": config})


def handle_get_series_config() -> None:
    """List the dataseries exported into the main plot application."""
    if not state.cities:
        protocol.send_error("Not initialized")
        return

    series = []
    for city in state.cities:
        prefix = f"{city} " if len(state.cities) > 1 else ""
        series.extend(
            [
                {"id": f"{city}_tmean", "name": f"{prefix}Mean Temp"},
                {"id": f"{city}_tmin", "name": f"{prefix}Min Temp"},
                {"id": f"{city}_tmax", "name": f"{prefix}Max Temp"},
            ]
        )

    protocol.send_response({"result": series})


def handle_get_series_data(series_id: str, preferred_storage: str) -> None:
    """Format and send float vectors containing timeseries outputs."""
    if not state.data:
        protocol.send_error("No data available")
        return

    parts = series_id.split("_")
    if len(parts) < 2:
        protocol.send_error("Invalid series format")
        return

    city = "_".join(parts[:-1])
    var_name = parts[-1]

    df = state.data.get(city)
    if df is None or var_name not in ["tmean", "tmin", "tmax"]:
        protocol.send_error("Series or city not found")
        return

    df = df.select(["date", var_name]).drop_nulls()

    if state.mode in ["Daily Means", "Daily Means (smoothed)"]:
        df = (
            df.with_columns(pl.col("date").str.slice(5).alias("md"))
            .group_by("md")
            .agg(pl.col(var_name).mean())
            .with_columns(
                pl.concat_str([pl.lit("2000-"), pl.col("md")]).alias("date")
            )
            .sort("date")
        )
        if state.mode == "Daily Means (smoothed)":
            df = df.with_columns(
                pl.col(var_name)
                .rolling_mean(window_size=10, center=True)
                .alias(var_name)
            ).drop_nulls()
    elif state.mode == "Monthly Means":
        df = (
            df.with_columns(pl.col("date").str.slice(5, 2).alias("month"))
            .group_by("month")
            .agg(pl.col(var_name).mean())
            .with_columns(
                pl.concat_str(
                    [pl.lit("2000-"), pl.col("month"), pl.lit("-15")]
                ).alias("date")
            )
            .sort("date")
        )
    elif state.mode == "Monthly":
        df = (
            df.with_columns(pl.col("date").str.slice(0, 7).alias("ym"))
            .group_by("ym")
            .agg(pl.col(var_name).mean())
            .with_columns(
                pl.concat_str([pl.col("ym"), pl.lit("-15")]).alias("date")
            )
            .sort("date")
        )

    values: list[float] = []

    for row in df.iter_rows(named=True):
        try:
            dt = datetime.datetime.strptime(row["date"], "%Y-%m-%d")
            timestamp = dt.timestamp()
            val = row[var_name]

            if val is not None:
                values.append(float(timestamp))
                values.append(float(val))
        except Exception:
            continue

    protocol.send_binary_data(values, "interleaved")


def main() -> None:
    """Primary entry block to route plugin execution via stdin."""
    while True:
        req = protocol.read_request()
        if not req:
            break

        method = req.get("method")
        if method == "info":
            protocol.send_response(
                {"name": "Open Meteo Downloader", "version": 1}
            )
        elif method == "initialize":
            handle_initialize(req.get("args", ""))
        elif method == "get_chart_config":
            handle_get_chart_config()
        elif method == "get_series_config":
            handle_get_series_config()
        elif method == "get_series_data":
            handle_get_series_data(
                req.get("series_id", ""),
                req.get("preferred_storage", "interleaved"),
            )
        else:
            protocol.send_error(f"Unknown method: {method}")


if __name__ == "__main__":
    main()
