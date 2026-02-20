# Open Meteo Climate Downloader

A data source plugin for OlicanaPlot that bulk-downloads and visualizes historical temperature data from the [Open-Meteo Historical Weather API](https://open-meteo.com/).

## Overview
This Python-based IPC plugin enables analysts to pull daily `Temperature Max`, `Temperature Min`, and `Temperature Mean` variables instantly and directly into OlicanaPlot. Built to support rapid comparisons:

- Provides a unified config panel to add one or multiple urban centers dynamically
- Queries Open-Meteo's internal geocoding system intelligently to automatically retrieve the exact coordinates behind complex town and city location strings
- Features a rich Svelte-driven mapping integration that drops physical pins on a world map in real-time as you generate your location list inside the app UI
- Persists all past queries efficiently to local disk (`open_meteo_cache.sq3`) avoiding duplicated expensive network requests so graphs re-load locally and quickly over months of usage

## Configuration
Requires `uv` to be installed on the host system to run.

```bash
uv sync   # Sets up the Python environment containing `requests` and `polars`
```

Upon plugin selection inside OlicanaPlot, you are prompted to enter:
1. **Locations**: Any number of named city strings (e.g. "Calgary", "Paris", "Austin Texas").
2. **Start Date** & **End Date**: Ranges anywhere back to 1940 up until 7 days ago (native Open-Meteo API limit). 
3. **History**: Easily replay or extend recent requests via the auto-maintained dropdown memory slot mapping.

## Development 
The Python execution handles its own dependencies within its local directory wrapper to isolate environments efficiently for the master app orchestrator. 

Check for programmatic formatting and warnings by running:
```bash
uv run ruff check
uv run ruff format
```
