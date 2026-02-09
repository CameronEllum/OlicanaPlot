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
  public lastSubplotCount: number = 0;
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
    yAxisNames: Record<number, string>,
  ) {
    if (!this.container) return;

    this.darkMode = darkMode;
    this.currentData = seriesData;

    // Always expect an array of series
    const seriesArr = Array.isArray(seriesData) ? seriesData : [seriesData];

    // Group series by subplotIndex
    const subplotIndices = [
      ...new Set(seriesArr.map((s) => s.subplotIndex || 0)),
    ].sort((a, b) => a - b);
    const numSubplots = subplotIndices.length;

    PluginService.LogDebug(
      "PlotlyAdapter",
      "Rendering subplots",
      subplotIndices.join(", "),
    );

    // Map subplotIndex to Plotly axis labels
    const subplotToAxisMap: Record<number, any> = {};
    for (const [i, sidx] of subplotIndices.entries()) {
      const axisNum = i === 0 ? "" : (i + 1).toString();
      subplotToAxisMap[sidx] = {
        x: `x${axisNum}`,
        y: `y${axisNum}`,
        xaxisKey: `xaxis${axisNum}`,
        yaxisKey: `yaxis${axisNum}`,
      };
    }

    const traces = seriesArr.map((s) => {
      const pointCount = s.data.length / 2;
      const xData = s.data.subarray(0, pointCount);
      const yData = s.data.subarray(pointCount);
      const axes = subplotToAxisMap[s.subplotIndex || 0];

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
        l: 80,
        r: getGridRight(seriesArr),
        t: 60,
        b: 70,
      },
      dragmode: "pan" as const,
    };

    // Calculate vertical domains
    const gap = 0.05;
    const h = (1.0 - (numSubplots - 1) * gap) / numSubplots;

    for (const [i, sidx] of subplotIndices.entries()) {
      const axes = subplotToAxisMap[sidx];
      // Stack from top down
      const rowTop = 1.0 - i * (h + gap);
      const rowBottom = rowTop - h;

      layout[axes.xaxisKey] = {
        title:
          i === numSubplots - 1
            ? { text: xAxisName, font: { size: 16, color: textColor } }
            : undefined,
        gridcolor: gridColor,
        zerolinecolor: gridColor,
        tickfont: { color: textColor },
        anchor: axes.y,
        matches: i === 0 ? undefined : "x", // link for sync zoom
        showticklabels: i === numSubplots - 1,
        showline: true,
        linewidth: 2,
        linecolor: axisLineColor,
      };

      layout[axes.yaxisKey] = {
        title: {
          text: yAxisNames[sidx] || `Subplot ${sidx}`,
          font: { size: 14, color: textColor },
        },
        gridcolor: gridColor,
        zerolinecolor: gridColor,
        tickfont: { color: textColor },
        domain: [Math.max(0, rowBottom), Math.min(1, rowTop)],
        anchor: axes.x,
        showline: true,
        linewidth: 2,
        linecolor: axisLineColor,
      };
    }

    const config = {
      responsive: true,
      scrollZoom: true,
      displayModeBar: true,
      modeBarButtonsToRemove: ["lasso2d", "select2d"],
    };

    // If the number of subplots changed, purge the plot to ensure a clean layout update
    if (numSubplots !== this.lastSubplotCount) {
      PluginService.LogDebug(
        "PlotlyAdapter",
        "Purging plot due to subplot count change",
        `from ${this.lastSubplotCount} to ${numSubplots}`,
      );
      Plotly.purge(this.container);
      this.lastSubplotCount = numSubplots;
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
      return { type: "other", rawEvent: e };
    }

    const layout = this.container._fullLayout;
    const rect = this.container.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    // 1. Check Main Title
    if (y < layout.margin.t) {
      // Precise check for centered title area if needed, but margin.t is usually safe
      return { type: "title", rawEvent: e };
    }

    // 2. Check X Axes (Bottom margin)
    // In our stacked layout, only the bottom-most axis has a title.
    if (y > layout.height - layout.margin.b) {
      // Find the primary x axis (highest index or simply xaxis)
      return { type: "xAxis", rawEvent: e, axisLabel: layout.xaxis.title?.text || "", axisIndex: 0 };
    }

    // 3. Check Y Axes (Left margin)
    if (x < layout.margin.l) {
      // Iterate through all y-axes to see which one's vertical span matches y
      const yAxisKeys = Object.keys(layout).filter(k => k.startsWith("yaxis"));
      for (const key of yAxisKeys) {
        const ax = layout[key];
        if (ax && ax._offset !== undefined && ax._length !== undefined) {
          if (y >= ax._offset && y <= ax._offset + ax._length) {
            const index = key === "yaxis" ? 0 : parseInt(key.replace("yaxis", ""), 10) - 1;
            return { type: "yAxis", rawEvent: e, axisLabel: ax.title?.text || "", axisIndex: index };
          }
        }
      }
    }

    // 4. Check Grid Area
    const dataPoint = this.getDataAtPixel(x, y);
    if (dataPoint) {
      return { type: "grid", rawEvent: e, dataPoint };
    }

    return { type: "other", rawEvent: e };
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
