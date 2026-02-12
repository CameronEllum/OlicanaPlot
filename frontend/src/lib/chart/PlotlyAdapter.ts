import Plotly from "plotly.js-dist-min";
import * as PluginService from "../../../bindings/olicanaplot/internal/plugins/service";
import {
  ChartAdapter,
  type ContextMenuEvent,
  type SeriesConfig,
} from "./ChartAdapter.ts";

// Plotly.js implementation of ChartAdapter using WebGL (scattergl).
// Implements true subplots by dynamically partitioning the Y domain.
export class PlotlyAdapter extends ChartAdapter {
  public container: any = null;
  public darkMode: boolean = false;
  public currentData: SeriesConfig[] | null = null;
  private lastGridKey: string = "";
  private contextMenuHandler: ((event: ContextMenuEvent) => void) | null = null;



  // Store the target container and initial theme for the Plotly instance.
  init(container: HTMLElement, darkMode: boolean) {
    this.container = container;
    this.darkMode = darkMode;
  }

  // Prepare the data traces and layout configuration, then render or update the
  // Plotly chart including subplot partitioning.
  setData(
    seriesData: SeriesConfig[],
    title: string,
    darkMode: boolean,
    getGridRight: (data: SeriesConfig[]) => number,
    lineWidth: number,
    xAxisName: string,
    yAxisNames: Record<string, string>,
    linkX: boolean,
    linkY: boolean,
  ) {
    if (!this.container) return;

    this.darkMode = darkMode;
    this.currentData = seriesData;

    // Always expect an array of series
    const seriesArr = Array.isArray(seriesData) ? seriesData : [seriesData];

    // Find 2D grid dimensions
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
    const numRows = maxRow + 1;
    const numCols = maxCol + 1;

    PluginService.LogDebug(
      "PlotlyAdapter",
      "Rendering 2D subplots",
      `Rows: ${numRows}, Cols: ${numCols}, Unique Cells: ${cells.length}`,
    );

    // Map cell "row,col" to Plotly axis labels
    const cellToAxisMap: Record<string, any> = {};
    for (const [i, cell] of cells.entries()) {
      const axisNum = i === 0 ? "" : (i + 1).toString();
      cellToAxisMap[cell.id] = {
        x: `x${axisNum}`,
        y: `y${axisNum}`,
        xaxisKey: `xaxis${axisNum}`,
        yaxisKey: `yaxis${axisNum}`,
        cell,
        axisIndex: i,
      };
    }

    const traces = seriesArr.map((s) => {
      const pointCount = s.data.length / 2;
      const xData = s.data.subarray(0, pointCount);
      const yData = s.data.subarray(pointCount);
      const cellId = `${s.subplotRow || 0},${s.subplotCol || 0}`;
      const axes = cellToAxisMap[cellId];

      return {
        x: xData,
        y: yData,
        xaxis: axes.x,
        yaxis: axes.y,
        name: s.name,
        type: "scattergl" as const,
        mode: "lines" as const,
        line: {
          color: s.color,
          width: lineWidth || 2,
        },
        hoverinfo: "x+y+name",
      };
    });

    const textColor = darkMode ? "#ccc" : "#333";
    const bgColor = darkMode ? "#2b2b2b" : "#ffffff";
    const gridColor = darkMode ? "#444" : "#e0e0e0";
    const axisLineColor = darkMode ? "#ccc" : "#000";

    const layout: any = {
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
        l: 100,
        r: getGridRight(seriesArr),
        t: 60,
        b: 70,
      },
      dragmode: "pan" as const,
    };

    // Calculate vertical and horizontal domains
    const xGap = numCols > 1 ? 0.12 : 0.04;
    const yGap = numRows > 1 ? 0.12 : 0.06;
    const cellW = (1.0 - (numCols - 1) * xGap) / numCols;
    const cellH = (1.0 - (numRows - 1) * yGap) / numRows;

    for (const [i, cell] of cells.entries()) {
      const axes = cellToAxisMap[cell.id];

      const xLeft = cell.col * (cellW + xGap);
      const xRight = xLeft + cellW;

      const yTop = 1.0 - cell.row * (cellH + yGap);
      const yBottom = Math.max(0, yTop - cellH);

      // Link X: Global vs Per Column
      let xMatches: string | undefined;
      if (linkX) {
        xMatches = i === 0 ? undefined : "x";
      } else if (cell.row !== 0) {
        xMatches = cellToAxisMap[`0,${cell.col}`]?.x || undefined;
      }

      layout[axes.xaxisKey] = {
        title:
          cell.row === maxRow
            ? { text: `<b>${xAxisName}</b>`, font: { size: 16, color: textColor } }
            : undefined,
        gridcolor: gridColor,
        zerolinecolor: gridColor,
        tickfont: { color: textColor },
        domain: [Math.max(0, xLeft), Math.min(1, xRight)],
        anchor: axes.y,
        matches: xMatches,
        showticklabels: cell.row === maxRow,
        showline: true,
        linewidth: 2,
        linecolor: axisLineColor,
      };

      const customYName = yAxisNames[cell.id];
      const defaultYName =
        cell.row === 0 && cell.col === 0
          ? "Main"
          : `Subplot ${cell.row},${cell.col}`;

      layout[axes.yaxisKey] = {
        title: {
          text: `<b>${customYName || defaultYName}</b>`,
          font: { size: 14, color: textColor },
          standoff: cell.col === 0 ? 30 : 15,
        },
        gridcolor: gridColor,
        zerolinecolor: gridColor,
        tickfont: { color: textColor },
        domain: [Math.max(0, yBottom), Math.min(1, yTop)],
        anchor: axes.x,
        matches: linkY ? (i === 0 ? undefined : "y") : undefined,
        showline: true,
        linewidth: 2,
        linecolor: axisLineColor,
        showticklabels: cell.col === 0,
      };
    }

    const config = {
      responsive: true,
      scrollZoom: true,
      displayModeBar: true,
      modeBarButtonsToRemove: ["lasso2d", "select2d"],
    };

    // If grid dimensions change, purge
    const gridKey = `${numRows}x${numCols}`;
    if (gridKey !== (this as any).lastGridKey) {
      PluginService.LogDebug(
        "PlotlyAdapter",
        "Purging plot due to grid dimension change",
        `from ${(this as any).lastGridKey || 'none'} to ${gridKey}`,
      );
      Plotly.purge(this.container);
      (this as any).lastGridKey = gridKey;
    }

    Plotly.react(this.container, traces, layout, config);

    // Legend context menu logic
    // Clean up old listeners before re-registering
    this.container.off?.("plotly_afterplot");
    this.container.on("plotly_afterplot", () => {
      // Legend context menu
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

      // Title context menu discovery
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
    });
  }

  // Trigger Plotly's internal resizing logic to fit the container.
  resize() {
    if (this.container) {
      Plotly.Plots.resize(this.container);
    }
  }

  // Calculate data-space coordinates from screen pixel values by using Plotly's
  // axis scaling functions.
  getDataAtPixel(x: number, y: number) {
    if (!this.container || !this.container._fullLayout) return null;
    const layout = this.container._fullLayout;
    const xaxis = layout.xaxis;
    const yaxis = layout.yaxis;
    if (!xaxis || !yaxis) return null;
    const plotLeft = xaxis._offset;
    const plotWidth = xaxis._length;
    const plotTop = yaxis._offset;
    const plotHeight = yaxis._length;
    if (x < plotLeft || x > plotLeft + plotWidth) return null;
    if (y < plotTop || y > plotTop + plotHeight) return null;
    const dataX = xaxis.p2d(x - plotLeft);
    const dataY = yaxis.p2d(plotTop + plotHeight - y);
    return { x: dataX, y: dataY };
  }
  // Map screen coordinates to chart regions (title, axes, grid).
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
    const yAxisKeys = Object.keys(layout).filter(k => k.startsWith("yaxis"));
    for (const yKey of yAxisKeys) {
      const yAx = layout[yKey];
      if (!yAx || yAx._offset === undefined) continue;

      // Determine the corresponding x-axis for this y-axis
      // Plotly links axes via the 'anchor' property. If anchor is 'x', it's xaxis.
      // Otherwise, it's 'xaxisN' where N is the anchor value.
      const xKey = yAx.anchor === "x" ? "xaxis" : `xaxis${yAx.anchor.replace('x', '')}`;
      const xAx = layout[xKey];
      if (!xAx || xAx._offset === undefined) continue;

      // Check vertical axis hit (left side)
      // The y-axis label area is typically to the left of the x-axis's start.
      // We check if the click is within the y-axis's vertical span and to the left of the plot area.
      if (x < xAx._offset && y >= yAx._offset && y <= yAx._offset + yAx._length) {
        // Extract axis index from key (e.g., "yaxis2" -> 2, "yaxis" -> 0)
        const axisIndex = yKey === "yaxis" ? 0 : parseInt(yKey.replace("yaxis", ""), 10) - 1;
        return {
          type: "yAxis",
          rawEvent: e,
          axisLabel: yAx.title?.text || "",
          axisIndex: axisIndex,
          x: e.clientX,
          y: e.clientY,
        };
      }

      // Check horizontal axis hit (bottom side)
      // The x-axis label area is typically below the y-axis's end.
      // We check if the click is within the x-axis's horizontal span and below the plot area.
      if (y > yAx._offset + yAx._length && x >= xAx._offset && x <= xAx._offset + xAx._length) {
        // Extract axis index from key (e.g., "xaxis2" -> 2, "xaxis" -> 0)
        const axisIndex = xKey === "xaxis" ? 0 : parseInt(xKey.replace("xaxis", ""), 10) - 1;
        return {
          type: "xAxis",
          rawEvent: e,
          axisLabel: xAx.title?.text || "",
          axisIndex: axisIndex,
          x: e.clientX,
          y: e.clientY
        };
      }

      // Check grid hit for this specific subplot
      if (x >= xAx._offset && x <= xAx._offset + xAx._length &&
        y >= yAx._offset && y <= yAx._offset + yAx._length) {
        const dataPoint = {
          x: xAx.p2d(x - xAx._offset),
          y: yAx.p2d(yAx._offset + yAx._length - y)
        };
        return { type: "grid", rawEvent: e, dataPoint, x: e.clientX, y: e.clientY };
      }
    }

    return { type: "other", rawEvent: e, x: e.clientX, y: e.clientY };
  }

  // Convert data coordinates into screen pixel coordinates for use by external
  // UI overlays.
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

  // Release all Plotly resources and clear the container.
  destroy() {
    if (this.container) {
      Plotly.purge(this.container);
    }
  }

  // Attach a handler for legend click events and return false to prevent
  // Plotly's default toggling behavior.
  onLegendClick(handler: (seriesName: string, event: any) => void) {
    if (!this.container) {
      PluginService.LogDebug("PlotlyAdapter", "onLegendClick: container is null", "");
      return;
    }

    try {
      PluginService.LogDebug("PlotlyAdapter", "Attaching plotly_legendclick", "");
      if (typeof this.container.on === "function") {
        this.container.on("plotly_legendclick", (event: any) => {
          try {
            const name = event.data[event.curveNumber].name;
            PluginService.LogDebug("PlotlyAdapter", `Legend click detected: ${name}`, "");
            handler(name, event);
          } catch (e: any) {
            PluginService.LogDebug("PlotlyAdapter", "Error in legend click handler callback", e.toString());
          }
          return false;
        });
        PluginService.LogDebug("PlotlyAdapter", "Successfully attached plotly_legendclick", "");
      } else {
        PluginService.LogDebug("PlotlyAdapter", "onLegendClick: container exists but .on is not a function", "");
      }
    } catch (e: any) {
      PluginService.LogDebug("PlotlyAdapter", "Failed to attach plotly_legendclick", e.toString());
    }
  }

  // Register a callback for handling right-click context menu events on the
  // chart.
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
