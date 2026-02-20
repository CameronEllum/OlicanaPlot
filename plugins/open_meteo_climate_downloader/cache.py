"""SQLite caching for open-meteo climate data."""

from __future__ import annotations

import datetime
import sqlite3
from typing import Any, Final

import polars as pl

CACHE_DB: Final[str] = "open_meteo_cache.sq3"

DAILY_SCHEMA: Final[dict[str, Any]] = {
    "city_name": pl.String,
    "date": pl.String,
    "year": pl.Int64,
    "month": pl.Int64,
    "day": pl.Int64,
    "tmean": pl.Float64,
    "tmin": pl.Float64,
    "tmax": pl.Float64,
}


class ClimateCache:
    """Manages SQLite caching for daily climate data."""

    def __init__(self, db_path: str = CACHE_DB) -> None:
        self.db_path = db_path
        self._init_db()

    def _init_db(self) -> None:
        with sqlite3.connect(self.db_path) as conn:
            conn.execute(
                """
                CREATE TABLE IF NOT EXISTS cities (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    name TEXT UNIQUE,
                    lat REAL,
                    lng REAL
                )
                """
            )
            conn.execute(
                """
                CREATE TABLE IF NOT EXISTS daily_data (
                    city_id INTEGER,
                    date INTEGER, -- YYYYMMDD
                    tmean REAL,
                    tmin REAL,
                    tmax REAL,
                    PRIMARY KEY (city_id, date)
                ) WITHOUT ROWID
                """
            )

    def save_city(self, name: str, lat: float, lng: float) -> int:
        """Save a city and return its internal ID."""
        with sqlite3.connect(self.db_path) as conn:
            conn.execute(
                """
                INSERT INTO cities (name, lat, lng)
                VALUES (?, ?, ?)
                ON CONFLICT(name) DO UPDATE SET
                    lat=excluded.lat,
                    lng=excluded.lng
                """,
                (name, lat, lng),
            )
            cursor = conn.cursor()
            cursor.execute("SELECT id FROM cities WHERE name = ?", (name,))
            return cursor.fetchone()[0]

    def _get_city_id(self, conn: sqlite3.Connection, name: str) -> int | None:
        cursor = conn.cursor()
        cursor.execute("SELECT id FROM cities WHERE name = ?", (name,))
        row = cursor.fetchone()
        return row[0] if row else None

    def get_city_location(self, name: str) -> tuple[float, float] | None:
        """Get the cached coordinates of a city."""
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.cursor()
            cursor.execute("SELECT lat, lng FROM cities WHERE name = ?", (name,))
            row = cursor.fetchone()
            return (row[0], row[1]) if row else None

    def has_date_range(
        self, city_name: str, start_date: str, end_date: str
    ) -> bool:
        """Check if we have the fully required date range for a city."""
        with sqlite3.connect(self.db_path) as conn:
            cid = self._get_city_id(conn, city_name)
            if cid is None:
                return False

            start_int = int(start_date.replace("-", ""))
            end_int = int(end_date.replace("-", ""))

            cursor = conn.cursor()
            cursor.execute(
                "SELECT MIN(date), MAX(date) FROM daily_data WHERE city_id = ?",
                (cid,),
            )
            row = cursor.fetchone()
            if not row or not row[0] or not row[1]:
                return False

            # Since Open-Meteo data lags
            today = datetime.date.today()
            lag_date = today - datetime.timedelta(days=7)
            lag_int = lag_date.year * 10000 + lag_date.month * 100 + lag_date.day

            target_end_date = min(end_int, lag_int)

            return row[0] <= start_int and row[1] >= target_end_date

    def get_daily_data(
        self, city_name: str, start_date: str, end_date: str
    ) -> pl.DataFrame:
        """Retrieve cached daily records for a city."""
        with sqlite3.connect(self.db_path) as conn:
            cid = self._get_city_id(conn, city_name)
            if cid is None:
                return pl.DataFrame(schema=DAILY_SCHEMA)

            start_int = int(start_date.replace("-", ""))
            end_int = int(end_date.replace("-", ""))

            query = """
                SELECT date, tmean, tmin, tmax
                FROM daily_data
                WHERE city_id = ? AND date BETWEEN ? AND ?
            """
            cursor = conn.cursor()
            cursor.execute(query, (cid, start_int, end_int))
            rows = cursor.fetchall()

            if not rows:
                return pl.DataFrame(schema=DAILY_SCHEMA)

            data = []
            for r in rows:
                d_int = r[0]
                s = str(d_int)
                y, m, d = int(s[:4]), int(s[4:6]), int(s[6:])
                d_str = f"{y:04d}-{m:02d}-{d:02d}"
                data.append(
                    {
                        "city_name": city_name,
                        "date": d_str,
                        "year": y,
                        "month": m,
                        "day": d,
                        "tmean": r[1],
                        "tmin": r[2],
                        "tmax": r[3],
                    }
                )
            df = pl.from_dicts(data, schema=DAILY_SCHEMA)
            return df.sort("date")

    def save_daily(self, city_name: str, df: pl.DataFrame) -> None:
        """Save a polars DataFrame to the daily data cache."""
        if df.is_empty():
            return

        with sqlite3.connect(self.db_path) as conn:
            cid = self._get_city_id(conn, city_name)
            if cid is None:
                return

            rows_to_insert = []
            for row in df.iter_rows(named=True):
                # parse YYYY-MM-DD
                d_str = row["date"].replace("-", "")
                if len(d_str) != 8:
                    continue
                d_int = int(d_str)

                rows_to_insert.append(
                    (cid, d_int, row["tmean"], row["tmin"], row["tmax"])
                )

            conn.executemany(
                """
                INSERT INTO daily_data
                (city_id, date, tmean, tmin, tmax)
                VALUES (?, ?, ?, ?, ?)
                ON CONFLICT(city_id, date) DO UPDATE SET
                    tmean=excluded.tmean,
                    tmin=excluded.tmin,
                    tmax=excluded.tmax
                """,
                rows_to_insert,
            )
