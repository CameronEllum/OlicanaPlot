import Plotly from "plotly.js-dist-min";
import * as PluginService from "../../../bindings/olicanaplot/internal/plugins/service";
import {
  ChartAdapter,
  type ContextMenuEvent,
  type SeriesConfig,
  getCSSVar,
} from "./ChartAdapter.ts";

// Plotly.js implementation of ChartAdapter using WebGL (scattergl).
// Implements true subplots by dynamically partitioning the Y domain.
export class PlotlyAdapter extends ChartAdapter {
  public container: any = null;
  public currentData: SeriesConfig[] | null = null;
  private lastGridKey: string = "";
  private contextMenuHandler: ((event: ContextMenuEvent) => void) | null = null;
  private cells: any[] = [];

  // Initialize the container.
  init(container: HTMLElement) {
    this.container = container;
  }

  // Set the chart data, configuration, and perform the draw operation.
  setData(
    seriesData: SeriesConfig[],
    title: string,
    getGridRight: (data: SeriesConfig[]) => number,
    lineWidth: number,
    xAxisName: string,
    yAxisNames: Record<string, string>,
    linkX: boolean,
    linkY: boolean,
  ) {
    if (!this.container) return;

    this.currentData = seriesData;

    const seriesArr = Array.isArray(seriesData) ? seriesData : [seriesData];
    const grid = this.getGridInfo(seriesArr);
    this.cells = grid.cells;

    PluginService.LogDebug(
      "PlotlyAdapter",
      "Rendering 2D subplots",
      `Rows: ${grid.numRows}, Cols: ${grid.numCols}, Unique Cells: ${grid.cells.length}`,
    );

    const { cellToAxisMap, gridSubplots } = this.createAxisMapping(
      grid.cells,
      grid.numRows,
      grid.numCols,
    );

    const traces = this.createTraces(seriesArr, cellToAxisMap, lineWidth);
    const layout = this.createBaseLayout(
      title,
      getGridRight(seriesArr),
      gridSubplots,
    );

    this.applyAxisConfiguration(
      layout,
      grid,
      cellToAxisMap,
      xAxisName,
      yAxisNames,
      linkX,
      linkY,
    );

    this.handleGridChange(grid.numRows, grid.numCols);
    this.renderPlot(traces, layout);
    this.setupPostPlotHandlers();
  }

  // Determine the dimensions and cell layout of the subplot grid.
  private getGridInfo(seriesArr: SeriesConfig[]) {
    const cells = [
      ...new Set(
        seriesArr.map((s) => `${s.subplotRow || 0},${s.subplotCol || 0}`),
      ),
    ]
      .map((str) => {
        const [r, c] = str.split(",").map(Number);
        return { row: r, col: c, id: str };
      })
      .sort((a, b) => a.row - b.row || a.col - b.col);

    const maxRow = Math.max(0, ...cells.map((c) => c.row));
    const maxCol = Math.max(0, ...cells.map((c) => c.col));

    return {
      cells,
      numRows: maxRow + 1,
      numCols: maxCol + 1,
      maxRow,
      maxCol,
    };
  }

  // Map each grid cell to its corresponding Plotly axis identifier.
  private createAxisMapping(cells: any[], numRows: number, numCols: number) {
    const cellToAxisMap: Record<string, any> = {};
    const gridSubplots: string[][] = Array.from({ length: numRows }, () =>
      Array(numCols).fill(""),
    );

    for (const [i, cell] of cells.entries()) {
      const axisNum = i === 0 ? "" : (i + 1).toString();
      const axes = {
        x: `x${axisNum}`,
        y: `y${axisNum}`,
        xaxisKey: `xaxis${axisNum}`,
        yaxisKey: `yaxis${axisNum}`,
        cell,
        axisIndex: i,
      };
      cellToAxisMap[cell.id] = axes;
      gridSubplots[cell.row][cell.col] = `${axes.x}${axes.y}`;
    }

    return { cellToAxisMap, gridSubplots };
  }

  // Generate the data traces for each series in the chart.
  private createTraces(
    seriesArr: SeriesConfig[],
    cellToAxisMap: Record<string, any>,
    lineWidth: number,
  ) {
    return seriesArr.map((s) => {
      const pointCount = s.data.length / 2;
      const xData = s.data.subarray(0, pointCount);
      const yData = s.data.subarray(pointCount);
      const cellId = `${s.subplotRow || 0},${s.subplotCol || 0}`;
      const axes = cellToAxisMap[cellId];

      const hasMarker = !!(s.marker_type && s.marker_type !== "none");
      const mode = hasMarker ? "lines+markers" : "lines";

      // Map marker types to Plotly symbols
      const markerSymbol = s.marker_type === "square" ? "square" :
        s.marker_type === "triangle" ? "triangle-up" :
          s.marker_type === "x" ? "x" :
            (s.marker_type || "circle");

      return {
        x: xData,
        y: yData,
        xaxis: axes.x,
        yaxis: axes.y,
        name: s.name,
        type: "scattergl" as const,
        mode: mode as any,
        line: {
          color: s.color,
          width: s.line_width || lineWidth || 2,
          dash: s.line_type === "dashed" ? "dash" : s.line_type === "dotted" ? "dot" : "solid",
        },
        // Using the spread operator to completely omit the 'marker' key if not needed.
        // This prevents Plotly's WebGL scatter from choking on undefined/null objects.
        ...(hasMarker && {
          marker: {
            color: s.color,
            size: 8,
            symbol: markerSymbol,
          }
        }),
        hoverinfo: "x+y+name",
      };
    });
  }

  // Create the base layout configuration object.
  private createBaseLayout(
    title: string,
    marginRight: number,
    gridSubplots: string[][],
  ) {
    const textColor = getCSSVar("--chart-text");
    const bgColor = getCSSVar("--chart-bg");

    return {
      title: {
        text: `<b>${title}</b>`,
        font: { color: textColor, size: 20 },
        x: 0.5,
        xanchor: "center" as const,
      },
      paper_bgcolor: bgColor,
      plot_bgcolor: bgColor,
      font: { color: textColor },
      showlegend: true,
      legend: {
        orientation: "v" as const,
        x: 1.05,
        y: 1,
        font: { color: textColor },
      },
      margin: {
        l: 60,
        r: marginRight,
        t: 50,
        b: 50,
      },
      dragmode: "pan" as const,
      grid: {
        subplots: gridSubplots,
        pattern: "independent",
        xgap: 0.22,
        ygap: 0.28,
        roworder: "top to bottom",
      },
    };
  }

  // Apply specific axis settings and linking to the layout.
  private applyAxisConfiguration(
    layout: any,
    grid: { cells: any[]; maxRow: number },
    cellToAxisMap: Record<string, any>,
    xAxisName: string,
    yAxisNames: Record<string, string>,
    linkX: boolean,
    linkY: boolean,
  ) {
    const textColor = getCSSVar("--chart-text");
    const gridColor = getCSSVar("--chart-grid");
    const axisLineColor = getCSSVar("--chart-axis");

    for (const [i, cell] of grid.cells.entries()) {
      const axes = cellToAxisMap[cell.id];

      layout[axes.xaxisKey] = {
        title:
          cell.row === grid.maxRow
            ? {
              text: `<b>${xAxisName}</b>`,
              font: { size: 14, color: textColor },
            }
            : undefined,
        gridcolor: gridColor,
        zerolinecolor: gridColor,
        tickfont: { color: textColor, size: 11 },
        anchor: axes.y,
        matches: linkX ? (i === 0 ? undefined : "x") : undefined,
        showticklabels: true,
        showline: true,
        linewidth: 2,
        linecolor: axisLineColor,
        automargin: true,
      };

      const customYName = yAxisNames[cell.id];
      const defaultYName =
        cell.row === 0 && cell.col === 0
          ? "Main"
          : `Subplot ${cell.row},${cell.col}`;

      layout[axes.yaxisKey] = {
        title: {
          text: `<b>${customYName || defaultYName}</b>`,
          font: { size: 12, color: textColor },
        },
        gridcolor: gridColor,
        zerolinecolor: gridColor,
        tickfont: { color: textColor },
        anchor: axes.x,
        matches: linkY ? (i === 0 ? undefined : "y") : undefined,
        showline: true,
        linewidth: 2,
        linecolor: axisLineColor,
        showticklabels: true,
        automargin: true,
      };
    }
  }

  // Purge the plot if the grid dimensions have changed.
  private handleGridChange(numRows: number, numCols: number) {
    const gridKey = `${numRows}x${numCols}`;
    if (gridKey !== this.lastGridKey) {
      PluginService.LogDebug(
        "PlotlyAdapter",
        "Purging plot due to grid dimension change",
        `from ${this.lastGridKey || "none"} to ${gridKey}`,
      );
      Plotly.purge(this.container);
      this.lastGridKey = gridKey;
    }
  }

  // Update the chart using the react method.
  private renderPlot(traces: any[], layout: any) {
    const config = {
      responsive: true,
      scrollZoom: true,
      displayModeBar: true,
      modeBarButtonsToRemove: ["lasso2d", "select2d"],
    };
    Plotly.react(this.container, traces, layout, config);
  }

  // Register event listeners for interactions after the chart renders.
  private setupPostPlotHandlers() {
    this.container.off?.("plotly_afterplot");
    this.container.on("plotly_afterplot", () => {
      this.setupLegendContextMenu();
      this.findTitleElement();
    });
  }

  // Attach context menu listeners to legend items.
  private setupLegendContextMenu() {
    const legendItems = this.container.querySelectorAll(
      ".legendtext, .legendtoggle",
    );
    for (const el of legendItems) {
      (el as any).oncontextmenu = (e: MouseEvent) => {
        e.preventDefault();
        e.stopPropagation();

        const traceGroup = (el as any).closest(".traces") as HTMLElement;
        const textEl = traceGroup
          ? traceGroup.querySelector(".legendtext")
          : null;
        const traceName = textEl ? textEl.textContent?.trim() : null;

        if (traceName && this.contextMenuHandler) {
          PluginService.LogDebug(
            "PlotlyAdapter",
            "Standardized legend context menu",
            traceName,
          );
          this.contextMenuHandler({
            type: "legend",
            rawEvent: e,
            seriesName: traceName,
            x: e.clientX,
            y: e.clientY,
          });
        }
      };
    }
  }

  // Locate and store a reference to the chart title element.
  private findTitleElement() {
    const titleEl = this.container.querySelector(
      ".gtitle, .g-title, .titletext, .main-title",
    );
    if (titleEl) {
      PluginService.LogDebug(
        "PlotlyAdapter",
        "Title element found, storing reference",
        "",
      );
      (this as any).titleElement = titleEl;
    } else {
      PluginService.LogDebug("PlotlyAdapter", "Title element NOT found", "");
    }
  }


  // Resize the chart to fit its container.
  resize() {
    if (this.container) {
      Plotly.Plots.resize(this.container);
    }
  }

  // Convert pixel coordinates to data coordinates using Plotly scaling functions.
  getDataAtPixel(x: number, y: number) {
    const layout = this.container?._fullLayout;
    const { xaxis, yaxis } = layout || {};

    // Early exit if the chart infrastructure isn't ready
    if (!xaxis || !yaxis) {
      return null;
    }

    // Extract dimensions for local coordinate mapping
    const { _offset: plotLeft, _length: plotWidth } = xaxis;
    const { _offset: plotTop, _length: plotHeight } = yaxis;

    // Verify the point resides within the active plotting region
    const isInsideX = x >= plotLeft && x <= plotLeft + plotWidth;
    const isInsideY = y >= plotTop && y <= plotTop + plotHeight;

    if (!isInsideX || !isInsideY) {
      return null;
    }

    // Map pixel offsets back to data values
    // Y-axis is inverted to account for screen coordinates (0 at top)
    const dataX = xaxis.p2d(x - plotLeft);
    const dataY = yaxis.p2d(plotHeight - (y - plotTop));

    return { x: dataX, y: dataY };
  }
  // Remove HTML tags from the given string.
  private stripTags(str: string): string {
    return str.replace(/<\/?b>/gi, "");
  }

  // Determine the chart element located at the click coordinates.
  private getClickTarget(e: MouseEvent): ContextMenuEvent {
    if (!this.container || !this.container._fullLayout) {
      return { type: "other", rawEvent: e, x: e.clientX, y: e.clientY };
    }

    const layout = this.container._fullLayout;
    const rect = this.container.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    if (y < layout.margin.t) {
      return { type: "title", rawEvent: e, x: e.clientX, y: e.clientY };
    }

    // Check all axis pairs
    const yAxisKeys = Object.keys(layout).filter((k) => k.startsWith("yaxis"));
    for (const yKey of yAxisKeys) {
      const yAx = layout[yKey];
      if (!yAx || yAx._offset === undefined) continue;

      // Determine the corresponding x-axis for this y-axis
      const xKey =
        yAx.anchor === "x" ? "xaxis" : `xaxis${yAx.anchor.replace("x", "")}`;
      const xAx = layout[xKey];
      if (!xAx || xAx._offset === undefined) continue;

      // Check vertical axis hit (left side)
      if (
        x < xAx._offset &&
        y >= yAx._offset &&
        y <= yAx._offset + yAx._length
      ) {
        const axisIndex =
          yKey === "yaxis"
            ? 0
            : parseInt(yKey.replace("yaxis", ""), 10) - 1;
        const cell = this.cells[axisIndex];
        return {
          type: "yAxis",
          rawEvent: e,
          axisLabel: this.stripTags(yAx.title?.text || ""),
          axisIndex: axisIndex,
          row: cell?.row,
          col: cell?.col,
          x: e.clientX,
          y: e.clientY,
        };
      }

      // Check horizontal axis hit (bottom side)
      if (
        y > yAx._offset + yAx._length &&
        x >= xAx._offset &&
        x <= xAx._offset + xAx._length
      ) {
        const axisIndex =
          xKey === "xaxis"
            ? 0
            : parseInt(xKey.replace("xaxis", ""), 10) - 1;
        const cell = this.cells[axisIndex];
        return {
          type: "xAxis",
          rawEvent: e,
          axisLabel: this.stripTags(xAx.title?.text || ""),
          axisIndex: axisIndex,
          row: cell?.row,
          col: cell?.col,
          x: e.clientX,
          y: e.clientY,
        };
      }

      // Check grid hit for this specific subplot
      if (
        x >= xAx._offset &&
        x <= xAx._offset + xAx._length &&
        y >= yAx._offset &&
        y <= yAx._offset + yAx._length
      ) {
        const dataPoint = {
          x: xAx.p2d(x - xAx._offset),
          y: yAx.p2d(yAx._offset + yAx._length - y),
        };
        return {
          type: "grid",
          rawEvent: e,
          dataPoint,
          x: e.clientX,
          y: e.clientY,
        };
      }
    }

    return { type: "other", rawEvent: e, x: e.clientX, y: e.clientY };
  }

  // Convert data coordinates to pixel coordinates.
  getPixelFromData(x: number, y: number) {
    if (!this.container || !this.container._fullLayout) return null;
    const layout = this.container._fullLayout;
    const xaxis = layout.xaxis;
    const yaxis = layout.yaxis;
    if (!xaxis || !yaxis) return null;
    const pixelX = xaxis._offset + xaxis.d2p(x);
    const pixelY = yaxis._offset + yaxis._length - yaxis.d2p(y);
    return { x: pixelX, y: pixelY };
  }

  // Clean up chart resources and purge the plot.
  destroy() {
    if (this.container) {
      Plotly.purge(this.container);
    }
  }

  // Register a handler for legend click events.
  onLegendClick(handler: (seriesName: string, event: any) => void) {
    if (!this.container) {
      PluginService.LogDebug(
        "PlotlyAdapter",
        "onLegendClick: container is null",
        "",
      );
      return;
    }

    try {
      PluginService.LogDebug(
        "PlotlyAdapter",
        "Attaching plotly_legendclick",
        "",
      );
      if (typeof this.container.on === "function") {
        this.container.on("plotly_legendclick", (event: any) => {
          try {
            const name = event.data[event.curveNumber].name;
            PluginService.LogDebug(
              "PlotlyAdapter",
              `Legend click detected: ${name}`,
              "",
            );
            handler(name, event);
          } catch (e: any) {
            PluginService.LogDebug(
              "PlotlyAdapter",
              "Error in legend click handler callback",
              e.toString(),
            );
          }
          return false;
        });
        PluginService.LogDebug(
          "PlotlyAdapter",
          "Successfully attached plotly_legendclick",
          "",
        );
      } else {
        PluginService.LogDebug(
          "PlotlyAdapter",
          "onLegendClick: container exists but .on is not a function",
          "",
        );
      }
    } catch (e: any) {
      PluginService.LogDebug(
        "PlotlyAdapter",
        "Failed to attach plotly_legendclick",
        e.toString(),
      );
    }
  }

  // Register a handler for right-click context menu events.
  onContextMenu(handler: (event: ContextMenuEvent) => void) {
    this.contextMenuHandler = handler;
    if (this.container) {
      this.container.addEventListener("contextmenu", (e: MouseEvent) => {
        e.preventDefault();
        e.stopPropagation();
        PluginService.LogDebug(
          "PlotlyAdapter",
          "Container context menu event",
          "",
        );

        const target = this.getClickTarget(e);
        handler(target);
      });
    }
  }
}
