import * as echarts from "echarts";
import * as PluginService from "../../../bindings/olicanaplot/internal/plugins/service";
import {
  ChartAdapter,
  type ContextMenuEvent,
  type SeriesConfig,
  type GridConfig,
  type ChartConfig,
  getCSSVar,
} from "./ChartAdapter.ts";

// ECharts implementation of ChartAdapter. Implements true subplots by using
// multiple grid objects stacked vertically.
export class EChartsAdapter extends ChartAdapter {
  public instance: echarts.ECharts | null = null;
  public container: HTMLElement | null = null;
  private lastArgs: any = null;
  private cells: any[] = [];

  // Approximate pixel width per character at the default 12px axis label font.
  private static readonly CHAR_WIDTH_PX = 7;

  // Create a new ECharts instance within the provided container.
  init(container: HTMLElement) {
    this.container = container;
    if (this.instance) {
      this.instance.dispose();
    }
    this.instance = echarts.init(container);
  }

  // Configure and render the ECharts visualization, including subplots, axes,
  // data mapping, and stylized components.
  update(
    seriesData: SeriesConfig[],
    getGridRight: (data: SeriesConfig[]) => number,
    config: ChartConfig,
  ) {
    if (!this.instance) return;
    this.lastArgs = { seriesData, getGridRight, config };

    const { title, grid, link_x, link_y } = config;
    const linkX = !!link_x;
    const linkY = !!link_y;
    const xAxisName = config.axes[0]?.x_axes[0]?.title || "X";
    const yAxisNames: Record<string, string> = {};
    const xAxisTypes: Record<string, string> = {};
    const yAxisTypes: Record<string, string> = {};

    config.axes.forEach(ag => {
      const key = `${ag.subplot.row},${ag.subplot.col}`;
      if (ag.y_axes[0]) {
        yAxisNames[key] = ag.y_axes[0].title;
        yAxisTypes[key] = ag.y_axes[0].type;
      }
      if (ag.x_axes[0]) {
        xAxisTypes[key] = ag.x_axes[0].type;
      }
    });

    const textColor = getCSSVar("--chart-text");
    const bgColor = getCSSVar("--chart-bg");
    const gridColor = getCSSVar("--chart-grid");

    // Always expect an array of series
    const seriesArr = Array.isArray(seriesData) ? seriesData : [seriesData];

    // Find 2D grid dimensions
    const cells = [
      ...new Set(
        seriesArr.map((s) => `${s.subplot.row},${s.subplot.col}`),
      ),
    ]
      .map((str) => {
        const [r, c] = str.split(",");
        const row = parseInt(r, 10);
        const col = parseInt(c, 10);
        return { row, col, id: str };
      })
      .sort((a, b) => a.row - b.row || a.col - b.col);

    this.cells = cells;

    this.cells = cells;

    const numRows = grid.rows;
    const numCols = grid.cols;

    PluginService.LogDebug(
      "EChartsAdapter",
      "Rendering 2D subplots",
      `Rows: ${numRows}, Cols: ${numCols}, Unique Cells: ${cells.length}`,
    );

    // Map cell ID "row,col" to actual grid index
    const cellToIndexMap: Record<string, number> = {};
    for (const [i, cell] of cells.entries()) {
      cellToIndexMap[cell.id] = i;
    }

    // Split vertical and horizontal space
    const totalHeight = 84;
    const cellHeight = totalHeight / numRows;

    // Logic for equal horizontal distribution within legend-free space.
    // We convert the pixel-based right margin into a percentage of the current
    // container width so that we can distribute the "inner" columns equally using percentages.
    const containerWidth = this.container?.clientWidth || 800;
    const containerHeight = this.container?.clientHeight || 600;

    // Estimate the widest Y-axis tick label (in px) for column-0 cells so the
    // left margin adapts to the data instead of using a fixed percentage.
    const col0TickWidth = this.estimateYTickWidth(seriesArr, cells.filter(c => c.col === 0));
    const leftMarginPx = col0TickWidth + 30; // tick width + axis title + padding
    const leftMarginPct = (leftMarginPx / containerWidth) * 100;
    const rightMarginPx = getGridRight(seriesArr);
    const rightMarginPct = (rightMarginPx / containerWidth) * 100;

    // Use pixel-based gaps converted to percentages to keep spacing consistent across window sizes
    const hGapPct = numCols > 1 ? (80 / containerWidth) * 100 : 0;
    const vGapPct = numRows > 1 ? (80 / containerHeight) * 100 : 0;

    const totalUsableWidthPct = 100 - leftMarginPct - rightMarginPct;
    const cellWidthPct =
      (totalUsableWidthPct - (numCols - 1) * hGapPct) / numCols;

    const grids = cells.map((cell) => {
      const top = 10 + cell.row * cellHeight;
      const left = leftMarginPct + cell.col * (cellWidthPct + hGapPct);
      const right = 100 - (left + cellWidthPct);

      return {
        left: `${left}%`,
        right: `${right}%`,
        top: `${top}%`,
        bottom:
          cell.row === numRows - 1
            ? "10%"
            : `${100 - (top + cellHeight - vGapPct)}%`,
        containLabel: true,
      };
    });

    const datasets = seriesArr.map((s) => {
      const cellId = `${s.subplot.row},${s.subplot.col}`;
      const isXDate = xAxisTypes[cellId] === "date";
      const isYDate = yAxisTypes[cellId] === "date";

      let source = s.data;
      if (isXDate || isYDate) {
        source = new Float64Array(s.data.length);
        for (let j = 0; j < s.data.length; j += 2) {
          source[j] = isXDate ? s.data[j] * 1000 : s.data[j];
          source[j + 1] = isYDate ? s.data[j + 1] * 1000 : s.data[j + 1];
        }
      }

      return {
        source: source,
        dimensions: ["x", "y"],
      };
    });

    const xAxes = cells.map((cell, i) => ({
      type: (xAxisTypes[cell.id] === "date" ? "time" : "value") as "time" | "value",
      name: cell.row === numRows - 1 ? xAxisName : "",
      nameLocation: "center" as const,
      nameGap: 30,
      gridIndex: i,
      axisLabel: { show: true },
      axisLine: { lineStyle: { color: getCSSVar("--chart-axis") } },
      splitLine: { lineStyle: { color: gridColor } },
      triggerEvent: true,
    }));

    // If linkY is active, calculate global min/max for all Y data
    let globalYMin: number | undefined;
    let globalYMax: number | undefined;
    if (linkY) {
      let min = Infinity;
      let max = -Infinity;
      seriesArr.forEach((s) => {
        if (!s.data) return;
        // Data is interleaved [x, y, x, y ...]
        for (let j = 1; j < s.data.length; j += 2) {
          const val = s.data[j];
          if (!Number.isNaN(val)) {
            if (val < min) min = val;
            if (val > max) max = val;
          }
        }
      });
      if (min !== Infinity) {
        // Add small padding
        const pad = (max - min) * 0.05 || 0.1;
        globalYMin = min - pad;
        globalYMax = max + pad;
      }
    }

    const yAxes = cells.map((cell, i) => {
      const customName = yAxisNames[cell.id];
      const defaultName =
        cell.row === 0 && cell.col === 0
          ? "Main"
          : `Subplot ${cell.row},${cell.col}`;

      // Compute nameGap dynamically from the widest tick label in this cell.
      const cellTickWidth = this.estimateYTickWidth(
        seriesArr.filter(s => `${s.subplot.row},${s.subplot.col}` === cell.id),
        [cell],
      );
      const nameGap = cellTickWidth + 15;

      return {
        type: (yAxisTypes[cell.id] === "date" ? "time" : "value") as "time" | "value",
        name: customName || defaultName,
        nameLocation: "center" as const,
        nameGap,
        nameRotate: 90,
        gridIndex: i,
        min: globalYMin,
        max: globalYMax,
        axisLabel: { show: true },
        axisLine: { lineStyle: { color: getCSSVar("--chart-axis") } },
        splitLine: { lineStyle: { color: gridColor } },
        nameTextStyle: { color: textColor, fontWeight: "bold" },
        axisTick: { show: true },
        triggerEvent: true,
      };
    });

    const series = seriesArr.map((s, i) => {
      const cellId = `${s.subplot.row},${s.subplot.col}`;
      const cellIdx = cellToIndexMap[cellId];
      const echartSymbol = (s.marker_type === "square" ? "rect" : s.marker_type) || "circle";
      const finalSymbol = s.marker_fill === "empty" ? `empty${echartSymbol.charAt(0).toUpperCase() + echartSymbol.slice(1)}` : echartSymbol;

      return {
        name: s.name,
        type: "line" as const,
        showSymbol: !!s.marker_type && s.marker_type !== "none",
        symbol: finalSymbol,
        symbolSize: s.marker_size || 8,
        datasetIndex: i,
        xAxisIndex: cellIdx,
        yAxisIndex: cellIdx,
        encode: { x: "x", y: "y" },
        large: true,
        emphasis: { disabled: true },
        color: s.color,
        lineStyle: {
          width: s.line_width,
          type: s.line_type,
        },
        sampling: "lttb" as const,
      };
    });

    // Generate dataZoom based on link settings
    const dataZoom: any[] = [];
    if (linkX) {
      dataZoom.push({
        type: "inside",
        xAxisIndex: cells.map((_, i) => i),
        filterMode: "none",
      });
    } else {
      // Independent X axes for each subplot
      cells.forEach((_, i) => {
        dataZoom.push({
          type: "inside",
          xAxisIndex: i,
          filterMode: "none",
        });
      });
    }

    if (linkY) {
      dataZoom.push({
        type: "inside",
        yAxisIndex: cells.map((_, i) => i),
        filterMode: "none",
      });
    }

    const currentOption = (this.instance ? this.instance.getOption() : null) as any;
    const currentSelected = currentOption?.legend?.[0]?.selected || {};

    const option = {
      backgroundColor: bgColor,
      animation: false,
      title: {
        text: title,
        left: "center",
        textStyle: { color: textColor },
        triggerEvent: true,
        backgroundColor: "transparent",
        padding: [5, 20],
      },
      tooltip: { trigger: "axis" as const },
      toolbox: {
        feature: {
          dataZoom: {
            xAxisIndex: linkX ? cells.map((_, i) => i) : "auto",
            yAxisIndex: linkY ? cells.map((_, i) => i) : "auto",
          },
          restore: {},
          saveAsImage: {},
        },
        right: 20,
        iconStyle: { borderColor: textColor },
      },
      dataZoom: dataZoom,
      legend: {
        data: seriesArr.map((s) => s.name),
        selected: currentSelected,
        orient: "vertical" as const,
        right: 10,
        top: 60,
        textStyle: { color: textColor },
        type: "scroll" as const,
        triggerEvent: true,
      },
      dataset: datasets,
      grid: grids,
      xAxis: xAxes,
      yAxis: yAxes,
      series: series,
    };

    this.instance.setOption(option, { notMerge: true });
  }

  // Inform ECharts that the container size has changed and update the layout
  // accordingly.
  resize() {
    if (this.instance) {
      this.instance.resize();
      // Re-trigger update if we have args, to re-calculate equal percentages
      // for subplots based on the new container width.
      if (this.lastArgs) {
        this.update(
          this.lastArgs.seriesData,
          this.lastArgs.getGridRight,
          this.lastArgs.config
        );
      }
    }
  }

  // Convert screen space pixel coordinates into data coordinate space based
  // on the first grid's scale.
  getDataAtPixel(x: number, y: number) {
    if (!this.instance) return null;
    const coord = this.instance.convertFromPixel({ gridIndex: 0 }, [
      x,
      y,
    ]) as number[];
    return coord ? { x: coord[0], y: coord[1] } : null;
  }

  // Convert data values into screen space pixel coordinates for use in
  // overlaying external elements.
  getPixelFromData(x: number, y: number) {
    if (!this.instance) return null;
    const pixel = this.instance.convertToPixel({ gridIndex: 0 }, [
      x,
      y,
    ]) as number[];
    return pixel ? { x: pixel[0], y: pixel[1] } : null;
  }

  // Estimate the pixel width of the widest Y-axis tick label for series
  // belonging to the given cells. Scans Y data to find min/max, formats
  // them, and returns an approximate pixel width.
  private estimateYTickWidth(
    seriesArr: SeriesConfig[],
    cells: { row: number; col: number; id: string }[],
  ): number {
    const cellIds = new Set(cells.map((c) => c.id));
    let min = Infinity;
    let max = -Infinity;

    for (const s of seriesArr) {
      const id = `${s.subplot.row},${s.subplot.col}`;
      if (!cellIds.has(id) || !s.data) continue;
      // Data is interleaved [x, y, x, y ...]
      for (let j = 1; j < s.data.length; j += 2) {
        const val = s.data[j];
        if (!Number.isNaN(val)) {
          if (val < min) min = val;
          if (val > max) max = val;
        }
      }
    }

    if (min === Infinity) return 30; // No data fallback

    // Format extreme values to estimate the longest label ECharts will produce
    const candidates = [min, max].map((v) => {
      // Use a compact format similar to ECharts' default axis label formatter
      if (Math.abs(v) >= 1e6) return v.toExponential(1);
      if (Number.isInteger(v)) return v.toString();
      // Limit to a reasonable number of decimal places
      const s = v.toString();
      return s.length > 8 ? v.toPrecision(5) : s;
    });

    const maxLen = Math.max(...candidates.map((s) => s.length));
    return maxLen * EChartsAdapter.CHAR_WIDTH_PX;
  }

  // Release all resources held by the ECharts instance and disconnect from
  // the DOM.
  destroy() {
    if (this.instance) {
      this.instance.dispose();
      this.instance = null;
    }
  }

  // Attach a listener for legend selection changes and force selected items
  // to remain visible while triggering the handler.
  onLegendClick(handler: (seriesName: string, event: any) => void) {
    if (!this.instance) return;
    this.instance.on("legendselectchanged", (params: any) => {
      handler(params.name, params);
    });
  }

  // Register a callback for handling right-click context menu events on the
  // chart.
  onContextMenu(handler: (event: ContextMenuEvent) => void) {
    if (!this.instance) return;

    // Flag to prevent double-firing between component and global surface listeners
    let handled = false;

    // Internal ECharts component events (Title, Legend, etc)
    this.instance.on("contextmenu", (params: any) => {
      const rawEvent = params.event?.event || params.event;
      if (rawEvent && typeof rawEvent.stopPropagation === "function") {
        rawEvent.stopPropagation();
      }
      handled = true;
      setTimeout(() => { handled = false; }, 50);

      if (params.componentType === "title") {
        PluginService.LogDebug(
          "EChartsAdapter",
          "Standardized title context menu",
          "",
        );
        handler({ type: "title", rawEvent, x: rawEvent.clientX, y: rawEvent.clientY });
      } else if (params.componentType === "legend") {
        // Resolve series name from dataIndex if name is missing
        let seriesName = params.name;
        if (!seriesName && params.dataIndex !== undefined) {
          const option = this.instance!.getOption() as any;
          const legendData = option.legend?.[0]?.data || [];
          const item = legendData[params.dataIndex];
          seriesName = typeof item === "string" ? item : item?.name;
        }

        PluginService.LogDebug(
          "EChartsAdapter",
          "Standardized legend context menu",
          seriesName || "unknown",
        );
        handler({ type: "legend", rawEvent, seriesName, x: rawEvent.clientX, y: rawEvent.clientY });
      } else if (
        params.componentType === "xAxis" ||
        params.componentType === "yAxis"
      ) {
        const axisIndex = params.componentIndex;
        const cell = this.cells[axisIndex];
        // Search for the axis in the current options to get its name
        const option = this.instance!.getOption() as any;
        const axisConfig = option[params.componentType][axisIndex];
        const axisLabel = axisConfig?.name || "";

        PluginService.LogDebug(
          "EChartsAdapter",
          `Standardized ${params.componentType} context menu`,
          `Index: ${axisIndex}, Name: ${axisLabel}`,
        );
        handler({
          type: params.componentType === "xAxis" ? "xAxis" : "yAxis",
          rawEvent,
          axisLabel,
          axisIndex,
          row: cell?.row,
          col: cell?.col,
          x: rawEvent.clientX,
          y: rawEvent.clientY,
        });
      } else {
        PluginService.LogDebug(
          "EChartsAdapter",
          `Standardized other component context menu: ${params.componentType}`,
          "",
        );
        handler({ type: "other", rawEvent, x: rawEvent.clientX, y: rawEvent.clientY });
      }
    });

    // Global ZRender surface events
    this.instance.getZr().on("contextmenu", (e: any) => {
      // (handled by high-level 'contextmenu' event above).
      if (e.target || handled) return;

      const rawEvent = e.event || e;
      if (rawEvent && typeof rawEvent.stopPropagation === "function") {
        rawEvent.stopPropagation();
      }
      const x = e.offsetX;
      const y = e.offsetY;

      // Check if the click is over a grid area
      const isOverGrid = this.instance!.containPixel(
        { componentType: "grid" } as any,
        [x, y],
      );

      if (isOverGrid) {
        const dataPoint = this.getDataAtPixel(x, y);
        PluginService.LogDebug(
          "EChartsAdapter",
          "Standardized grid context menu",
          "",
        );
        handler({ type: "grid", rawEvent, dataPoint: dataPoint || undefined, x: rawEvent.clientX, y: rawEvent.clientY });
      } else {
        PluginService.LogDebug(
          "EChartsAdapter",
          "Standardized other surface context menu",
          "",
        );
        handler({ type: "other", rawEvent, x: rawEvent.clientX, y: rawEvent.clientY });
      }
    });
  }
}
