// Define the structure for a single data series to be plotted, including its
// identity, display name, color, and raw data.
export interface SeriesConfig {
  id: string;
  name: string;
  color: string;
  data: Float64Array;
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
  // Initialize the chart within the specified DOM container and set the
  // initial theme mode.
  abstract init(container: HTMLElement, darkMode: boolean): void;

  // Render the provided series data on the chart with a title, theme
  // configuration, and dynamic grid calculation.
  abstract setData(
    seriesData: SeriesConfig[],
    title: string,
    darkMode: boolean,
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
