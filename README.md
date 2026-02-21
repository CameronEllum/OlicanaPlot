# OlicanaPlot

A high-performance plotting application built with **Wails v3** (Go) and **Svelte 5** using Apache ECharts and Plotly.js for visualization. Features efficient binary data transfer via Asset Middleware and a hybrid plugin architecture supporting both built-in and IPC-based plugins.

Currently supports loading data from CSV and an internal multi-axes file format.

Through its inter-process communication other formats can be added by plugins written in any language. Example plugins in Go, C++, and Python are included. Plug-ins can specify a UI schema that the main application can display, or they can use their own UI.

## Screenshots

### Mutliple Axes

<img width="1206" height="971" alt="image" src="https://github.com/user-attachments/assets/214bd377-2ee6-4eac-b98e-765932366e7f" />

### Adding Data

<img width="655" height="720" alt="image" src="https://github.com/user-attachments/assets/dabf1fae-3c74-4552-9dfd-def0ff1be038" />

### Plug-in Management

<img width="800" height="600" alt="image" src="https://github.com/user-attachments/assets/c92dc87b-d72c-4bfa-b7db-1f878d111adc" />

### Internal File Format


```yaml
version: 1

chart:
  title: "Vehicle Performance"
  line_width: 2.0

layout:
  rows: 2
  cols: 2

behaviour:
  link_x: true
  link_y: false

axes:
  - title: "Engine Dynamics"
    subplot: [0, 0]
    x_axes:
      - title: "Time"
        unit: "s"
        position: "bottom"
    y_axes:
      - title: "RPM"
        unit: "rpm"
        position: "left"
      - title: "Torque"
        unit: "Nm"
        position: "right"
    series:
      - title: "Engine Speed"
        column: 1
        y_axis: "RPM"
        color: "#1f77b4"
      - title: "Output Torque"
        column: 2
        y_axis: "Torque"
        line_type: "dashed"

  - title: "Temperature Sensors"
    subplot: [1, 0]
    y_axes:
      - title: "Temperature"
        unit: "degC"
    series:
      - title: "Coolant Temp"
        column: 1
        y_axis: "Temperature"
        color: "#d62728"

  - title: "Line Types Demo"
    subplot: [0, 1]
    y_axes:
      - title: "Amplitude"
    series:
      - title: "Solid Line"
        column: 1
        line_type: "solid"
      - title: "Dashed Line"
        column: 2
        line_type: "dashed"
      - title: "Dotted Line"
        column: 3
        line_type: "dotted"

  - title: "Line Widths Demo"
    subplot: [1, 1]
    y_axes:
      - title: "Width"
    series:
      - title: "Thin (1px)"
        column: 1
        line_width: 1.0
      - title: "Medium (4px)"
        column: 2
        line_width: 4.0
      - title: "Thick (8px)"
        column: 3
        line_width: 8.0
```

### Axis Types & Date Requirements

Both the Internal File Format and the Plugin System support specifying the `type` of an axis:

- **`value`** (default): Standard linear numeric scale.
- **`date`**: Renders numeric values as formatted dates/times.
- **`log`**: Logarithmic scale.

#### Date Requirements as Unix Seconds

Because OlicanaPlot uses the Asset Middleware to transfer all chart data as raw `float64` binary arrays for maximum performance, **all timestamp or date data must be transferred as Unix timestamps in seconds (as `float64`)** via IPC or from built-in plugins. The frontend charts will automatically convert these numeric timestamps back into formatted dates if the corresponding axis is set to `type: "date"`.

For files like `.olicanaplot` and `.csv`, representations can be explicitly specified on axes or series to help parse string dates during loading:
- `unix_seconds_timepoint` (default for numbers)
- `iso8601_timepoint` (e.g., `2026-01-02T15:04:05Z`)
- `iso8601_basic_timepoint` (e.g., `20260102150405`)

These textual dates are parsed into `float64` Unix seconds upon loading.

## Features

- **High-Performance Charting**: Uses Apache ECharts with canvas renderer or Plotly.js with ScatterGL for smooth visualization of large datasets
- **Efficient Binary Data Transfer**: Asset Middleware pattern - binary `float64` arrays are fetched directly via HTTP, bypassing JSON serialization overhead
- **Plugin Architecture**: Hybrid system supporting:
  - **Built-in plugins**: Compiled into the application as Go packages
  - **IPC plugins**: External executables communicating via stdin/stdout
- **Multiple Data Sources**:
  - CSV file loading with column selection
  - Synthetic data generation (Gauss-Markov, Random Walk, Sinusoidal)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Frontend (Svelte 5)                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │   ECharts   │  │   Dialogs   │  │   fetch('/data/..') │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
           │ Wails Bindings      │ Binary Data (fetch)
           ▼                     ▼
┌─────────────────────────────────────────────────────────────┐
│                      Go Backend                             │
│  ┌─────────────────────┐  ┌─────────────────────────────┐   │
│  │   Plugin Manager    │  │   Asset Middleware          │   │
│  └─────────────────────┘  │   (/data/series?...)        │   │
│           │               └─────────────────────────────┘   │
│  ┌────────┴────────┐                                        │
│  │                 │                                        │
│  ▼                 ▼                                        │
│  Built-in      IPC Plugins                                  │
│  Plugins       (subprocess)                                 │
└─────────────────────────────────────────────────────────────┘
```

### Data Transfer: Asset Middleware Pattern

OlicanaPlot uses **Wails v3's Asset Middleware** for efficient binary data transfer:

1. **Frontend requests data**: `fetch('/data/series?plugin=X&series=Y')`
2. **Middleware intercepts**: Go middleware at `/data/series` handles the request
3. **Plugin returns data**: Plugin's `GetSeriesData()` returns `[]float64`
4. **Binary response**: Middleware writes raw bytes (8 bytes per float64, little-endian)
5. **Frontend receives**: `response.arrayBuffer()` → `new Float64Array(buffer)`

This approach is simpler and more efficient than WebSocket:
- Uses existing asset server (no separate port/connection)
- Pull-based (frontend fetches on demand)
- Zero serialization overhead for numerical data

### Plugin System

#### Built-in Plugins

Built-in plugins are compiled as part of the application. Each plugin is a separate Go package in `internal/plugins/`:

- **CSV Connector** (`internal/plugins/csv_reader/`): Load and plot CSV files
- **Synthetic Data Generator** (`internal/plugins/synthetic/`): Generate test data

#### IPC Plugins

External plugins communicate via stdin/stdout using a simple JSON + binary protocol:

**Request format** (JSON line):
```json
{"method": "info|initialize|call|get_chart_config|get_series_config|get_series_data", ...}
```

**Response format**:
- For JSON responses: `{"result": ..., "error": null}`
- For binary data: `{"type": "binary", "length": 8000}` followed by raw bytes

Example IPC plugin: `plugins/synthetic-ipc/`

## Prerequisites

1. **Go 1.21+** - [Install Go](https://go.dev/doc/install)
2. **Node.js 18+ LTS** - [Install Node.js](https://nodejs.org/)
3. **Wails v3 CLI**:
   ```bash
   go install github.com/wailsapp/wails/v3/cmd/wails3@latest
   ```
## Setup

1. **Clone and enter the project directory**:
   ```bash
   cd OlicanaPlot
   ```

2. **Install frontend dependencies**:
   ```bash
   cd frontend
   npm install --legacy-peer-deps
   cd ..
   ```

3. **Generate Wails bindings**:
   ```bash
   wails3 generate bindings
   ```

4. **Build the IPC plugin** (optional, for demonstrating IPC functionality):
   ```bash
   cd plugins/synthetic-ipc
   go build -o synthetic-ipc.exe .
   cd ../..
   ```

## Running

### Quick Start (Windows)

Double-click `run.bat` or:

```bash
.\run.bat
```

### Development Mode

```bash
wails3 dev
```

This starts the application with hot-reload for frontend changes.

### Production Build

```bash
wails3 build
```

The compiled application will be in `bin/`.

## Usage

1. **Load CSV Data**: Click "Load CSV" to select a CSV file, then configure which columns to plot
2. **Generate Synthetic Data**: Click "Generate Synthetic" to create test data with configurable parameters
3. **IPC Plugin Demo**: If the IPC plugin is built, click "Synthetic (IPC)" to use the external plugin

### Chart Interaction

- **Pan**: Drag with mouse
- **Zoom**: Scroll wheel
- **Box Zoom**: Use the toolbox zoom feature
- **Reset**: Click restore in the toolbox

## Creating New Plugins

### Built-in Plugin

1. Create a new package in `internal/plugins/yourplugin/`
2. Implement the `plugins.Plugin` interface:

```go
type Plugin interface {
    Name() string
    Version() uint32
    Initialize() (bool, error)
    Call(cmd string, argsJSON string) (string, error)
    GetChartConfig(args string) (*ChartConfig, error)
    GetSeriesConfig() ([]SeriesConfig, error)
    GetSeriesData(seriesID string) ([]float64, error)
    Close() error
}
```

3. Register in `main.go`:
```go
pluginManager.Register(yourplugin.New())
```

### IPC Plugin

1. Create a new directory in `plugins/yourplugin/`
2. Create an executable that:
   - Reads JSON requests from stdin (one per line)
   - Writes JSON responses to stdout
   - For binary data, writes header then raw bytes

See `plugins/synthetic-ipc/main.go` as a reference implementation.

## License

MIT
