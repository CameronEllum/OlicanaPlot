import * as echarts from "echarts";
import * as PluginService from "../../../bindings/olicanaplot/internal/plugins/service";
import {
  ChartAdapter,
  type ContextMenuEvent,
  type SeriesConfig,
} from "./ChartAdapter.ts";

// ECharts implementation of ChartAdapter. Implements true subplots by using
// multiple grid objects stacked vertically.
export class EChartsAdapter extends ChartAdapter {
  public instance: echarts.ECharts | null = null;
  public container: HTMLElement | null = null;


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
    yAxisNames: Record<number, string>,
  ) {
    if (!this.instance) return;

    const textColor = darkMode ? "#ccc" : "#333";
    const bgColor = darkMode ? "#2b2b2b" : "#ffffff";

    // Always expect an array of series
    const seriesArr = Array.isArray(seriesData) ? seriesData : [seriesData];

    // Group series by subplotIndex
    const subplotIndices = [
      ...new Set(seriesArr.map((s) => s.subplotIndex || 0)),
    ].sort((a, b) => a - b);
    const numSubplots = subplotIndices.length;

    PluginService.LogDebug(
      "EChartsAdapter",
      "Rendering subplots",
      subplotIndices.join(", "),
    );

    // Map subplotIndex to actual grid/axis index in ECharts
    const subplotToIndexMap: Record<number, number> = {};
    for (const [i, sidx] of subplotIndices.entries()) {
      subplotToIndexMap[sidx] = i;
    }

    // Split vertical space
    const totalHeight = 84;
    const subplotHeight = totalHeight / numSubplots;
    const gap = numSubplots > 1 ? 5 : 0;

    const grids = subplotIndices.map((_, i) => {
      const top = 10 + i * subplotHeight;
      return {
        left: 80,
        right: getGridRight(seriesArr),
        top: `${top}%`,
        height: `${subplotHeight - gap}%`,
        containLabel: true,
      };
    });

    const datasets = seriesArr.map((s) => ({
      source: s.data,
      dimensions: ["x", "y"],
    }));

    const xAxes = subplotIndices.map((_, i) => ({
      type: "value" as const,
      name: i === numSubplots - 1 ? xAxisName : "",
      nameLocation: "center" as const,
      nameGap: 30,
      gridIndex: i,
      axisLabel: { show: i === numSubplots - 1 },
      axisLine: { lineStyle: { color: textColor } },
      splitLine: { lineStyle: { color: darkMode ? "#444" : "#e0e0e0" } },
      triggerEvent: true,
    }));

    const yAxes = subplotIndices.map((sidx, i) => ({
      type: "value" as const,
      name: yAxisNames[sidx] || `Subplot ${sidx}`,
      nameLocation: "center" as const,
      nameGap: 45,
      nameRotate: 90,
      gridIndex: i,
      axisLine: { lineStyle: { color: textColor } },
      splitLine: { lineStyle: { color: darkMode ? "#444" : "#e0e0e0" } },
      nameTextStyle: { color: textColor, fontWeight: "bold" },
      axisTick: { show: true },
      triggerEvent: true,
    }));

    const series = seriesArr.map((s, i) => {
      const subplotIdx = subplotToIndexMap[s.subplotIndex || 0];
      return {
        name: s.name,
        type: "line" as const,
        showSymbol: false,
        datasetIndex: i,
        xAxisIndex: subplotIdx,
        yAxisIndex: subplotIdx,
        encode: { x: "x", y: "y" },
        large: true,
        emphasis: { disabled: true },
        color: s.color,
        lineStyle: { width: lineWidth || 2 },
        sampling: "lttb" as const,
      };
    });

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
          dataZoom: { xAxisIndex: subplotIndices.map((_, i) => i) },
          restore: {},
          saveAsImage: {},
        },
        right: 20,
        iconStyle: { borderColor: textColor },
      },
      dataZoom: [
        {
          type: "inside" as const,
          xAxisIndex: subplotIndices.map((_, i) => i),
          filterMode: "none" as const,
        },
      ],
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
        handler({ type: "title", rawEvent });
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
        handler({ type: "legend", rawEvent, seriesName });
      } else if (
        params.componentType === "xAxis" ||
        params.componentType === "yAxis"
      ) {
        const axisIndex = params.componentIndex;
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
        });
      } else {
        PluginService.LogDebug(
          "EChartsAdapter",
          `Standardized other component context menu: ${params.componentType}`,
          "",
        );
        handler({ type: "other", rawEvent });
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
        handler({ type: "grid", rawEvent, dataPoint: dataPoint || undefined });
      } else {
        PluginService.LogDebug(
          "EChartsAdapter",
          "Standardized other surface context menu",
          "",
        );
        handler({ type: "other", rawEvent });
      }
    });
  }
}
