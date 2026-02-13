// AxisConfig describes an axis within a subplot.
export interface AxisConfig {
  title?: string;
  position?: "bottom" | "top" | "left" | "right";
  unit?: string;
  type?: "linear" | "log" | "date";
  min?: number;
  max?: number;
}

// AxisGroupConfig describes all axes and series for one subplot cell.
export interface AxisGroupConfig {
  title?: string;
  subplot: number[]; // [row, col]
  x_axes?: AxisConfig[];
  y_axes?: AxisConfig[];
}
// ChartConfig contains chart display configuration.
export interface ChartConfig {
  title: string;
  axis_labels: string[];
  line_width?: number;
  axes?: AxisGroupConfig[];
  link_x?: boolean;
  link_y?: boolean;
  rows?: number;
  cols?: number;
}

// Define the structure for a single data series to be plotted, including its
// identity, display name, color, and raw data.
export interface SeriesConfig {
  id: string;
  name: string;
  color: string;
  data: Float64Array;
  subplot?: number[]; // [row, col]
  line_type?: "solid" | "dashed" | "dotted";
  line_width?: number;
  unit?: string;
  visible?: boolean;
  y_axis?: string; // references Y axis title
  // Compatibility fields
  subplotRow?: number;
  subplotCol?: number;
}

// Define the standardized structure for context menu events across chart
// adapters.
export interface ContextMenuEvent {
  type: "title" | "legend" | "grid" | "xAxis" | "yAxis" | "other";
  rawEvent: MouseEvent;
  x: number;
  y: number;
  seriesName?: string; // For legend items
  dataPoint?: { x: number; y: number }; // For grid clicks
  axisLabel?: string; // For axis items
  axisIndex?: number; // For identifying which axis
  row?: number;
  col?: number;
}

// Define the interface for different chart library implementations (e.g.,
// ECharts, Plotly).
export abstract class ChartAdapter {
  // Initialize the chart within the specified DOM container.
  abstract init(container: HTMLElement): void;

  // Render the provided series data on the chart with a title
  // and dynamic grid calculation.
  abstract setData(
    seriesData: SeriesConfig[],
    title: string,
    getGridRight: (data: SeriesConfig[]) => number,
    lineWidth: number,
    xAxisName: string,
    yAxisNames: Record<string, string>,
    linkX: boolean,
    linkY: boolean,
  ): void;

  // Trigger the chart to update its dimensions to fit its container.
  abstract resize(): void;

  // Convert screen pixel coordinates to the corresponding data values in the
  // chart's coordinate system.
  abstract getDataAtPixel(x: number, y: number): { x: number; y: number } | null;

  // Translate data values into screen pixel coordinates.
  abstract getPixelFromData(
    x: number,
    y: number,
  ): { x: number; y: number } | null;

  // Clean up any resources, event listeners, or DOM elements created by the
  // chart instance.
  abstract destroy(): void;

  // Register a callback to be executed when a legend item is clicked.
  abstract onLegendClick(
    handler: (seriesName: string, event: any) => void,
  ): void;

  // Register a callback for handling right-click context menu events on the
  // chart.
  abstract onContextMenu(handler: (event: ContextMenuEvent) => void): void;
}

export function getCSSVar(name: string): string {
  if (typeof window === 'undefined') return '';
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
}
