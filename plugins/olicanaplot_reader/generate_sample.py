import math
import random

yaml_header = """version: 1

chart:
  title: "Vehicle Performance"
  line_width: 2.0

layout:
  rows: 2
  cols: 1

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
"""

# Generate data for block 1 (RPM and Torque)
num_points = 101
t_start, t_end = 0, 10
t_vals = [t_start + (t_end - t_start) * i / (num_points - 1) for i in range(num_points)]

# Block 1 CSV
csv1 = []
for t in t_vals:
    rpm = 800 + 400 * math.sin(0.5 * t) + 10 * random.uniform(-1, 1)
    torque = 100 + 20 * math.cos(0.5 * t) + 2 * random.uniform(-1, 1)
    csv1.append(f"{t:.3f},{rpm:.1f},{torque:.1f}")

# Block 2 CSV
csv2 = []
for t in t_vals:
    temp = 90 + 5 * math.sin(0.1 * t) + 0.1 * random.uniform(-1, 1)
    csv2.append(f"{t:.3f},{temp:.2f}")

# Combine with form feeds (\x0c)
output = yaml_header + "\f\n" + "\n".join(csv1) + "\n\f\n" + "\n".join(csv2) + "\n"

# Write to file
file_path = "test_data.olicanaplot"
with open(file_path, "w", encoding="utf-8") as f:
    f.write(output)

print(f"Generated {file_path}")
