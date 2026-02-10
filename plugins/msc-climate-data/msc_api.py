"""MSC GeoMet OGC API client for climate data."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any

import httpx

import ipc_helpers

BASE_URL = "https://api.weather.gc.ca"
MAX_LIMIT = 10000

PROVINCES: dict[str, str] = {
    "AB": "ALBERTA",
    "BC": "BRITISH COLUMBIA",
    "MB": "MANITOBA",
    "NB": "NEW BRUNSWICK",
    "NL": "NEWFOUNDLAND",
    "NS": "NOVA SCOTIA",
    "NT": "NORTHWEST TERRITORIES",
    "NU": "NUNAVUT",
    "ON": "ONTARIO",
    "PE": "PRINCE EDWARD ISLAND",
    "QC": "QUEBEC",
    "SK": "SASKATCHEWAN",
    "YT": "YUKON TERRITORY",
}


@dataclass
class Station:
    """A climate station from the inventory."""

    climate_id: str
    name: str
    province: str
    province_code: str
    first_date: str | None
    last_date: str | None
    latitude: float
    longitude: float


@dataclass
class DailyObservation:
    """A single daily climate observation."""

    date: str
    mean_temp: float | None
    min_temp: float | None
    max_temp: float | None


def search_stations(province_code: str | None = None) -> list[Station]:
    """Search for stations with daily data in a province (with pagination)."""
    url = f"{BASE_URL}/collections/climate-stations/items"
    stations: list[Station] = []
    offset = 0

    with httpx.Client(timeout=30.0) as client:
        while True:
            params: dict[str, Any] = {
                "f": "json",
                "limit": MAX_LIMIT,
                "offset": offset,
            }

            if province_code and province_code in PROVINCES:
                params["ENG_PROV_NAME"] = PROVINCES[province_code]

            ipc_helpers.log("debug", f"OGC API Request: {url} params={params}")
            response = client.get(url, params=params)
            response.raise_for_status()
            data = response.json()

            for feature in data.get("features", []):
                props = feature["properties"]
                # Only include stations that have daily data
                if props.get("DLY_FIRST_DATE"):
                    stations.append(
                        Station(
                            climate_id=props["CLIMATE_IDENTIFIER"],
                            name=props["STATION_NAME"],
                            province=props["ENG_PROV_NAME"],
                            province_code=props["PROV_STATE_TERR_CODE"],
                            first_date=props["DLY_FIRST_DATE"],
                            last_date=props["DLY_LAST_DATE"],
                            latitude=feature["geometry"]["coordinates"][1],
                            longitude=feature["geometry"]["coordinates"][0],
                        )
                    )

            # Pagination logic
            matched = data.get("numberMatched", 0)
            returned = data.get("numberReturned", 0)
            offset += returned

            if offset >= matched or returned == 0:
                break

    # Sort by name
    stations.sort(key=lambda s: s.name)
    return stations


def search_stations_bbox(
    lat: float, lon: float, start_date: str, end_date: str, buffer: float = 0.2
) -> list[Station]:
    """Search for stations with daily data in a bounding box around a point."""
    bbox = f"{lon - buffer},{lat - buffer},{lon + buffer},{lat + buffer}"
    url = f"{BASE_URL}/collections/climate-daily/items"
    stations_map: dict[str, Station] = {}

    # We only need to find unique stations, limit to a reasonable number for discovery
    params: dict[str, Any] = {
        "f": "json",
        "bbox": bbox,
        "datetime": f"{start_date}/{end_date}",
        "limit": 500,
    }

    ipc_helpers.log(
        "debug", f"Discovery OGC API Request: {url} params={params}"
    )
    with httpx.Client(timeout=30.0) as client:
        response = client.get(url, params=params)
        response.raise_for_status()
        data = response.json()

        for feature in data.get("features", []):
            props = feature["properties"]
            cid = props.get("CLIMATE_IDENTIFIER")
            if cid and cid not in stations_map:
                stations_map[cid] = Station(
                    climate_id=cid,
                    # Fallback to cid if name missing
                    name=props.get("STATION_NAME", cid),
                    province=props.get("ENG_PROV_NAME", ""),
                    province_code=props.get("PROV_STATE_TERR_CODE", ""),
                    first_date=None,  # Not directly available in daily view
                    last_date=None,
                    latitude=feature["geometry"]["coordinates"][1],
                    longitude=feature["geometry"]["coordinates"][0],
                )

    return sorted(stations_map.values(), key=lambda s: s.name)


def fetch_daily_data(
    climate_id: str, start_date: str, end_date: str
) -> list[DailyObservation]:
    """Fetch daily climate observations for a station and date range."""
    # Ensure start_date is before end_date
    if start_date > end_date:
        ipc_helpers.log(
            "warn",
            f"Dates were backwards: {start_date} -> {end_date}. Swapping.",
        )
        start_date, end_date = end_date, start_date

    url = f"{BASE_URL}/collections/climate-daily/items"
    observations: list[DailyObservation] = []
    offset = 0

    with httpx.Client(timeout=60.0) as client:
        while True:
            params: dict[str, Any] = {
                "f": "json",
                "CLIMATE_IDENTIFIER": climate_id,
                "datetime": f"{start_date}/{end_date}",
                "limit": MAX_LIMIT,
                "offset": offset,
                "sortby": "LOCAL_DATE",
            }

            ipc_helpers.log("debug", f"OGC API Request: {url} params={params}")
            response = client.get(url, params=params)
            response.raise_for_status()
            data = response.json()

            features = data.get("features", [])
            if not features:
                break

            for feature in features:
                props = feature["properties"]
                # Skip if no temperature data at all
                if (
                    props.get("MEAN_TEMPERATURE") is None
                    and props.get("MIN_TEMPERATURE") is None
                    and props.get("MAX_TEMPERATURE") is None
                ):
                    continue

                observations.append(
                    DailyObservation(
                        date=props["LOCAL_DATE"].split()[0],  # Get YYYY-MM-DD
                        mean_temp=props.get("MEAN_TEMPERATURE"),
                        min_temp=props.get("MIN_TEMPERATURE"),
                        max_temp=props.get("MAX_TEMPERATURE"),
                    )
                )

            # Check if we need to paginate
            matched = data.get("numberMatched", 0)
            returned = data.get("numberReturned", 0)
            offset += returned

            if offset >= matched or returned == 0:
                break

    return observations
