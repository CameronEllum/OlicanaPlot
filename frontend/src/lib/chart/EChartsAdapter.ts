import * as echarts from "echarts";
import * as PluginService from "../../../bindings/olicanaplot/internal/plugins/service";
import {
  ChartAdapter,
  type ContextMenuEvent,
  type SeriesConfig,
  getCSSVar,
} from "./ChartAdapter.ts";

// ECharts implementation of ChartAdapter. Implements true subplots by using
// multiple grid objects stacked vertically.
export class EChartsAdapter extends ChartAdapter {
  public instance: echarts.ECharts | null = null;
  public container: HTMLElement | null = null;
  private lastArgs: any = null;
  private cells: any[] = [];

  // Create a new ECharts instance within the provided container and apply the
  // appropriate theme.
  init(container: HTMLElement, darkMode: boolean) {
    this.container = container;
    if (this.instance) {
      this.instance.dispose();
    }
    this.instance = echarts.init(container, darkMode ? "dark" : undefined);
  }

  // Configure and render the ECharts visualization, including subplots, axes,
  // data mapping, and stylized components.
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
    if (!this.instance) return;
    this.lastArgs = {
      seriesData,
      title,
      darkMode,
      getGridRight,
      lineWidth,
      xAxisName,
      yAxisNames,
      linkX,
      linkY,
    };

    const textColor = getCSSVar("--chart-text");
    const bgColor = getCSSVar("--chart-bg");
    const gridColor = getCSSVar("--chart-grid");

    // Always expect an array of series
    const seriesArr = Array.isArray(seriesData) ? seriesData : [seriesData];

    // Find 2D grid dimensions
    const cells = [
      ...new Set(
        seriesArr.map((s) => `${s.subplotRow || 0},${s.subplotCol || 0}`),
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

    const maxRow = Math.max(0, ...cells.map((c) => c.row));
    const maxCol = Math.max(0, ...cells.map((c) => c.col));
    const numRows = maxRow + 1;
    const numCols = maxCol + 1;

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
    const leftMarginPct = 8;
    const rightMarginPx = getGridRight(seriesArr);
    const rightMarginPct = (rightMarginPx / containerWidth) * 100;

    // Use pixel-based gaps converted to percentages to keep spacing consistent across window sizes
    const hGapPct = numCols > 1 ? (80 / containerWidth) * 100 : 0;
    const vGapPct = numRows > 1 ? (60 / containerHeight) * 100 : 0;

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
          cell.row === maxRow
            ? "10%"
            : `${100 - (top + cellHeight - vGapPct)}%`,
        containLabel: true,
      };
    });

    const datasets = seriesArr.map((s) => ({
      source: s.data,
      dimensions: ["x", "y"],
    }));

    const xAxes = cells.map((cell, i) => ({
      type: "value" as const,
      name: cell.row === maxRow ? xAxisName : "",
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

      return {
        type: "value" as const,
        name: customName || defaultName,
        nameLocation: "center" as const,
        nameGap: cell.col === 0 ? 45 : 20,
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
      const cellId = `${s.subplotRow || 0},${s.subplotCol || 0}`;
      const cellIdx = cellToIndexMap[cellId];
      return {
        name: s.name,
        type: "line" as const,
        showSymbol: false,
        datasetIndex: i,
        xAxisIndex: cellIdx,
        yAxisIndex: cellIdx,
        encode: { x: "x", y: "y" },
        large: true,
        emphasis: { disabled: true },
        color: s.color,
        lineStyle: { width: lineWidth || 2 },
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
      // Re-trigger setData if we have args, to re-calculate equal percentages
      // for subplots based on the new container width.
      if (this.lastArgs) {
        this.setData(
          this.lastArgs.seriesData,
          this.lastArgs.title,
          this.lastArgs.darkMode,
          this.lastArgs.getGridRight,
          this.lastArgs.lineWidth,
          this.lastArgs.xAxisName,
          this.lastArgs.yAxisNames,
          this.lastArgs.linkX,
          this.lastArgs.linkY,
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
      const option = this.instance!.getOption() as any;
      const selected = option.legend[0].selected || {};
      // Maintain visibility by overriding the automatic hide behavior
      for (const name of Object.keys(selected)) {
        selected[name] = true;
      }
      this.instance!.setOption({ legend: { selected } });
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
