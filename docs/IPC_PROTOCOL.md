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
  "series_id": "string (optional)",
  "data": "object (optional - for form_change)"
}
```

### Response (Plugin -> Host)
```json
{
  "method": "string (optional - for host-bound calls like log/show_form)",
  "result": any (optional),
  "error": "string (optional)",
  "type": "string (optional)",
  "length": number (optional),
  "name": "string (optional)",
  "version": number (optional),
  "title": "string (optional - for show_form)",
  "schema": "object (optional - for show_form/update)",
  "uiSchema": "object (optional - for show_form/update)",
  "data": "object (optional - for form updates)"
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
Returns [x, y] data for a series.
- **Request**: `{"method": "get_series_data", "series_id": "s1", "preferred_storage": "interleaved|arrays"}`
  - `preferred_storage`: (Optional) Hint for preferred data layout.
- **Response (Header)**: `{"type": "binary", "length": N, "storage": "interleaved|arrays"}`
  - `storage`: The actual layout used in the follow-up binary data.
- **Followed by**: N bytes of raw binary data (float64, little-endian).

### 6. `show_form` (Plugin -> Host Request)
During initialization, a plugin may request the host to show a configuration form. This is a rare case where the host acts as a server to the plugin's request.
- **Request (Plugin to Host stdout)**:
  ```json
  {
    "method": "show_form",
    "title": "Config Title",
    "schema": { ... JSON Schema ... },
    "uiSchema": { ... Optional UI hints ... },
    "handle_form_change": true  // Optional: Set to true to receive dynamic 'form_change' notifications
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

### 7. Form Change Notifications (Host <-> Plugin)
During an active `show_form` session, the host sends field change notifications if the plugin supports it. The plugin can respond with an updated UI (schema/uiSchema) or an empty response if no update is needed.

- **Notification (Host to Plugin stdin)**:
  ```json
  {
    "method": "form_change",
    "data": { "field1": "val1", ... }
  }
  ```

- **Response (Plugin to Host stdout)**:
  ```json
  {
    "schema": { ... },
    "uiSchema": { ... },
    "data": { ... }
  }
  ```
  *Note: An empty JSON object `{}` indicates no UI update is required.*

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
The binary data should be a sequence of 64-bit IEEE 754 floating-point numbers in **Little Endian** format. 

If `storage` is `interleaved`: `x0, y0, x1, y1, ...`.
If `storage` is `arrays`: `x0, x1, ... xn, y0, y1, ... yn`.

Total number of points is `length / 16`.
