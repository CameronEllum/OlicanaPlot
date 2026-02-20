"""Open Meteo API Client."""

import datetime

import polars as pl
import requests


def geocode_city(name: str) -> tuple[float, float] | None:
    """Geocode a city name using Open-Meteo Geocoding API."""
    url = "https://geocoding-api.open-meteo.com/v1/search"
    params = {"name": name, "count": 1, "language": "en", "format": "json"}
    resp = requests.get(url, params=params, timeout=10)
    resp.raise_for_status()
    data = resp.json()
    if "results" in data and len(data["results"]) > 0:
        res = data["results"][0]
        return float(res["latitude"]), float(res["longitude"])
    return None


def fetch_historical(
    lat: float, lng: float, start_date: str, end_date: str
) -> pl.DataFrame:
    """Fetch historical daily data for a location."""
    url = "https://archive-api.open-meteo.com/v1/archive"

    today_str = str(datetime.date.today())
    target_end_date = min(end_date, today_str)

    params = {
        "latitude": lat,
        "longitude": lng,
        "start_date": start_date,
        "end_date": target_end_date,
        "daily": "temperature_2m_max,temperature_2m_min,temperature_2m_mean",
        "timezone": "auto",
    }

    resp = requests.get(url, params=params, timeout=30)
    resp.raise_for_status()
    data = resp.json()

    daily = data.get("daily", {})
    dates = daily.get("time", [])
    tmax = daily.get("temperature_2m_max", [])
    tmin = daily.get("temperature_2m_min", [])
    tmean = daily.get("temperature_2m_mean", [])

    if not dates or not tmax or not tmin or not tmean:
        return pl.DataFrame()

    return pl.DataFrame(
        {"date": dates, "tmean": tmean, "tmin": tmin, "tmax": tmax}
    )
