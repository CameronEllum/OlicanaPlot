"""IPC protocol helpers for OlicanaPlot plugins."""

from __future__ import annotations

import json
import os
import struct
import sys
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    pass

CHART_COLORS: list[str] = [
    "#636EFA",
    "#EF553B",
    "#00CC96",
    "#AB63FA",
    "#FFA15A",
    "#19D3F3",
    "#FF6692",
    "#B6E880",
    "#FF97FF",
    "#FECB52",
]


def send_response(data: dict[str, Any]) -> None:
    """Send a JSON response to the host."""
    sys.stdout.write(json.dumps(data) + "\n")
    sys.stdout.flush()


def send_error(msg: str) -> None:
    """Send an error response to the host."""
    send_response({"error": msg})


def send_show_form(
    title: str,
    schema: dict[str, Any],
    ui_schema: dict[str, Any],
    data: dict[str, Any] | None = None,
    handle_form_change: bool = False,
) -> None:
    """Request the host to show an interactive form."""
    resp = {
        "method": "show_form",
        "title": title,
        "schema": schema,
        "uiSchema": ui_schema,
        "handle_form_change": handle_form_change,
    }
    if data:
        resp["data"] = data
    send_response(resp)


def send_binary_data(values: list[float], storage: str = "interleaved") -> None:
    """Send binary float64 data following a JSON header."""
    # JSON header
    header = {
        "type": "binary",
        "length": len(values) * 8,  # bytes
        "storage": storage,
    }
    sys.stdout.write(json.dumps(header) + "\n")
    sys.stdout.flush()

    # Binary payload
    if os.name == "nt":
        import msvcrt

        msvcrt.setmode(sys.stdout.fileno(), os.O_BINARY)

    # '<' for little-endian, 'd' for float64
    payload = struct.pack(f"<{len(values)}d", *values)
    sys.stdout.buffer.write(payload)
    sys.stdout.buffer.flush()

    if os.name == "nt":
        import msvcrt

        msvcrt.setmode(sys.stdout.fileno(), os.O_TEXT)


def log(level: str, message: str) -> None:
    """Send an asynchronous log message to the host."""
    send_response({"method": "log", "level": level, "message": message})


def read_request() -> dict[str, Any] | None:
    """Read a JSON request from stdin."""
    line = sys.stdin.readline()
    if not line:
        return None
    try:
        return json.loads(line.strip())
    except json.JSONDecodeError:
        return None
