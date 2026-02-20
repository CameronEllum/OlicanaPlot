"""Local flat-file search history caching."""

from __future__ import annotations

import json
import os
from typing import Any

HISTORY_FILE = "search_history.json"


def get_past_searches() -> list[dict[str, Any]]:
    """Retrieve unique past searches from local JSON."""
    if not os.path.exists(HISTORY_FILE):
        return []
    try:
        with open(HISTORY_FILE, "r") as f:
            data = json.load(f)
            if isinstance(data, list):
                return data
    except Exception:
        pass
    return []


def log_search(cities: list[str], start_date: str, end_date: str) -> None:
    """Log a search into the JSON history file."""
    history = get_past_searches()

    new_entry = {
        "cities": cities,
        "start_date": start_date,
        "end_date": end_date,
    }

    # Filter out exact duplicate to move it to top
    filtered = []
    for h in history:
        if (
            h.get("cities") == cities
            and h.get("start_date") == start_date
            and h.get("end_date") == end_date
        ):
            continue
        filtered.append(h)

    filtered.insert(0, new_entry)
    # Keep last 15
    filtered = filtered[:15]

    try:
        with open(HISTORY_FILE, "w") as f:
            json.dump(filtered, f, indent=2)
    except Exception:
        pass
