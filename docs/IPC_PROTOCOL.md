# OlicanaPlot IPC Plugin Protocol (v1)

This document specifies the communication protocol between the OlicanaPlot host application and external plugins running as subprocesses.

## Communication Channel
Plugins communicate with the host via **Standard Input (stdin)** and **Standard Output (stdout)**.

- **Host to Plugin**: JSON-encoded request on a single line (followed by `\n`).
- **Plugin to Host**: JSON-encoded response on a single line (followed by `\n`), OR a binary data stream following a JSON header.

## Message Formats

### Request (Host -> Plugin)
```json
{
  "method": "string",
  "args": "string (optional)",
  "series_id": "string (optional)"
}
```

### Response (Plugin -> Host)
```json
{
  "result": any (optional),
  "error": "string (optional)",
  "type": "string (optional)",
  "length": number (optional),
  "name": "string (optional)",
  "version": number (optional)
}
```

## Required Methods

### 1. `info`
Returns plugin basic information.
- **Request**: `{"method": "info"}`
- **Response**: `{"name": "Plugin Name", "version": 1}`

### 2. `initialize`
Initializes the plugin. This is where the plugin should show its configuration dialog if needed.
- **Request**: `{"method": "initialize", "args": "init_string"}`
- **Response**: `{"result": "success_message"}`

### 3. `get_chart_config`
Returns the chart title and axis labels.
- **Request**: `{"method": "get_chart_config"}`
- **Response**: `{"result": {"title": "Chart Title", "axis_labels": ["X", "Y"]}}`

### 4. `get_series_config`
Returns the list of series available.
- **Request**: `{"method": "get_series_config"}`
- **Response**: `{"result": [{"id": "s1", "name": "Series 1", "color": "#hex"}]}`

### 5. `get_series_data`
Returns interleaved [x, y] data for a series.
- **Request**: `{"method": "get_series_data", "series_id": "s1"}`
- **Response (Header)**: `{"type": "binary", "length": N}`
- **Followed by**: N bytes of raw binary data (float64, little-endian).

### 6. `show_form` (Plugin -> Host Request)
During initialization, a plugin may request the host to show a configuration form. This is a rare case where the host acts as a server to the plugin's request.
- **Request (Plugin to Host stdout)**:
  ```json
  {
    "method": "show_form",
    "title": "Config Title",
    "schema": { ... JSON Schema ... },
    "uiSchema": { ... Optional UI hints ... }
  }
  ```
- **Response (Host to Plugin stdin)**:
  ```json
  {
    "result": { "field1": "value", ... }
  }
  ```
- **Error (Host to Plugin stdin)**:
  ```json
  {
    "error": "cancelled"
  }
  ```

## Logging (Plugin -> Host)
Plugins can send asynchronous log messages at any time (except during binary transfer) by sending a JSON line:
```json
{
  "method": "log",
  "level": "info|warn|error|debug",
  "message": "log message"
}
```

## Binary Data Format
The binary data should be a sequence of 64-bit IEEE 754 floating-point numbers in **Little Endian** format. The data must be interleaved: `x0, y0, x1, y1, ...`.
Total number of points is `length / 16`.
