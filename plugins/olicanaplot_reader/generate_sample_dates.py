import datetime
import math
import random

yaml_header = """version: 1

chart:
  title: "Date Performance"
  line_width: 2.0

layout:
  rows: 3
  cols: 1

behaviour:
  link_x: true
  link_y: false

axes:
  - title: "ISO8601 Base Format (Z)"
    subplot: [0, 0]
    x_axes:
      - title: "Time (RFC3339)"
        type: "date"
        position: "bottom"
        representation: "iso8601_timepoint"
    y_axes:
      - title: "Value"
        position: "left"
    series:
      - title: "Sin/Cos combination"
        column: 1
        color: "#1f77b4"
        marker_type: "circle"
        
  - title: "ISO8601 Basic (Compact)"
    subplot: [1, 0]
    x_axes:
      - title: "Time (Compact)"
        type: "date"
        position: "bottom"
        representation: "iso8601_basic_timepoint"
    y_axes:
      - title: "Value"
        position: "left"
    series:
      - title: "Sin/Cos combination 2"
        column: 1
        color: "#ff7f0e"
        marker_type: "square"
        marker_fill: "empty"
        marker_size: 8.0

  - title: "Unix Timestamps & Styling"
    subplot: [2, 0]
    x_axes:
      - title: "Time"
        type: "date"
        position: "bottom"
        representation: "unix_seconds_timepoint"
    y_axes:
      - title: "Opacity & Thickness"
        position: "left"
    series:
      - title: "Unix Epoch Base"
        column: 1
        color: "rgba(255, 0, 0, 1.0)"
        line_width: 1.0
      - title: "Unix 50% Opacity Thick"
        column: 2
        color: "rgba(255, 0, 0, 0.5)"
        line_width: 4.0
      - title: "Unix 25% Opacity Thickest"
        column: 3
        color: "rgba(255, 0, 0, 0.25)"
        line_width: 8.0
"""

num_points = 50
start_time = datetime.datetime(2026, 1, 1, 12, 0, 0)
time_step = datetime.timedelta(hours=1)

# Block 1 CSV: ISO8601 Base (Z)
csv1 = []
for i in range(num_points):
    t_obj = start_time + i * time_step
    t_str = t_obj.strftime("%Y-%m-%dT%H:%M:%SZ")
    val1 = 100 + 40 * math.sin(i * 0.2) + random.uniform(-2, 2)
    csv1.append(f"{t_str},{val1:.1f}")

# Block 2 CSV: ISO8601 Basic (Compact)
csv2 = []
for i in range(num_points):
    t_obj = start_time + i * time_step
    t_str = t_obj.strftime("%Y%m%d%H%M%S.000")
    val1 = 100 + 40 * math.cos(i * 0.2) + random.uniform(-2, 2)
    csv2.append(f"{t_str},{val1:.1f}")

# Block 3 CSV: Unix Epoch
csv3 = []
for i in range(num_points):
    t_obj = start_time + i * time_step
    t_unix = t_obj.timestamp()
    val1 = math.sin(i * 0.1)
    val2 = math.sin(i * 0.1)
    val3 = math.sin(i * 0.1)
    csv3.append(f"{t_unix:.3f},{val1:.3f},{val2:.3f},{val3:.3f}")

output = (
    yaml_header
    + "\f\n"
    + "\n".join(csv1)
    + "\n\f\n"
    + "\n".join(csv2)
    + "\n\f\n"
    + "\n".join(csv3)
    + "\n"
)

file_path = "test_date_data.olicanaplot"
with open(file_path, "w", encoding="utf-8") as f:
    f.write(output)

print(f"Generated {file_path}")
