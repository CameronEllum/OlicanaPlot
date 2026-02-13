# OlicanaPlot File Format (.olicanaplot)

The OlicanaPlot format consists of a YAML header for metadata and layout, followed by one or more headerless CSV data blocks. Sections are separated by the form feed character (`\f`, `0x0C`).

## Structure

```
[YAML Header]
\f
[CSV Block 0]
\f
[CSV Block 1]
...
```

## YAML Schema

### `chart` (Optional)
- `title`: Main chart title.
- `line_width`: Default line width for all series.

### `layout` (Optional)
- `rows`: Number of subplot rows (default: 1).
- `cols`: Number of subplot columns (default: 1).

### `behaviour` (Optional)
- `link_x`: Whether X axes are linked across subplots (default: true).
- `link_y`: Whether Y axes are linked across subplots (default: false).

### `axes` (Required)
A list of subplot definitions:
- `title`: Subplot title.
- `subplot`: `[row, col]` position.
- `x_axes`: List of X axis definitions (`title`, `unit`, `position`, `type`, `min`, `max`).
- `y_axes`: List of Y axis definitions (`title`, `unit`, `position`, `type`, `min`, `max`).
- `series`: List of series definitions:
  - `title`: Series name.
  - `column`: 0-indexed column index in the corresponding CSV block (0 is usually X).
  - `y_axis`: Title of the target Y axis from the `y_axes` list.
  - `color`: Hex color string.
  - `line_type`: `solid`, `dashed`, or `dotted`.

---

## Minimal Example

A single subplot using mostly defaults.

```yaml
version: 1
axes:
  - subplot: [0, 0]
    series:
      - column: 1
```
`\f`
```csv
0,10.5
1,11.2
2,10.8
```

---

## Complete Example

Multiple subplots with custom linking and axis configuration.

```yaml
version: 1

chart:
  title: "Vehicle Telemetry"

layout:
  rows: 2
  cols: 1

behaviour:
  link_x: true
  link_y: false

axes:
  - title: "Engine Performance"
    subplot: [0, 0]
    x_axes:
      - title: "Time"
        unit: "s"
    y_axes:
      - title: "RPM"
        unit: "rpm"
        position: left
      - title: "Torque"
        unit: "Nm"
        position: right
    series:
      - title: "Engine Speed"
        column: 1
        y_axis: "RPM"
        color: "#1f77b4"
      - title: "Output Torque"
        column: 2
        y_axis: "Torque"
        line_type: "dashed"

  - title: "Temperature"
    subplot: [1, 0]
    y_axes:
      - title: "Coolant"
        unit: "°C"
    series:
      - title: "Temp"
        column: 1
        y_axis: "Coolant"
```
`\f`
```csv
0,800,120
0.1,850,125
0.2,900,130
```
`\f`
```csv
0,85.5
0.1,85.7
0.2,86.0
```
