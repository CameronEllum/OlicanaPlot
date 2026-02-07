import * as echarts from "echarts";
import { ChartAdapter, type SeriesConfig } from "./ChartAdapter.ts";

/**
 * ECharts implementation of ChartAdapter.
 * Implements true subplots by using multiple grid objects stacked vertically.
 */
export class EChartsAdapter extends ChartAdapter {
    public instance: echarts.ECharts | null = null;
    public container: HTMLElement | null = null;

    constructor() {
        super();
    }

    init(container: HTMLElement, darkMode: boolean) {
        this.container = container;
        if (this.instance) {
            this.instance.dispose();
        }
        this.instance = echarts.init(container, darkMode ? "dark" : undefined);
    }

    setData(seriesData: SeriesConfig[], title: string, darkMode: boolean, getGridRight: (data: SeriesConfig[]) => number) {
        if (!this.instance) return;

        const textColor = darkMode ? "#ccc" : "#333";
        const bgColor = darkMode ? "#2b2b2b" : "#ffffff";

        // Always expect an array of series
        const seriesArr = Array.isArray(seriesData) ? seriesData : [seriesData];

        // Group series by subplotIndex
        const subplotIndices = [...new Set(seriesArr.map((s) => s.subplotIndex || 0))].sort((a, b) => a - b);
        const numSubplots = subplotIndices.length;

        console.log(`[EChartsAdapter] Rendering subplots:`, subplotIndices);

        // Map subplotIndex to actual grid/axis index in ECharts
        const subplotToIndexMap: Record<number, number> = {};
        subplotIndices.forEach((sidx, i) => {
            subplotToIndexMap[sidx] = i;
        });

        // Split vertical space
        const totalHeight = 84;
        const subplotHeight = totalHeight / numSubplots;
        const gap = numSubplots > 1 ? 5 : 0;

        const grids = subplotIndices.map((_, i) => {
            const top = 10 + (i * subplotHeight);
            return {
                left: 80,
                right: getGridRight(seriesArr),
                top: `${top}%`,
                height: `${subplotHeight - gap}%`,
                containLabel: true
            };
        });

        const datasets = seriesArr.map((s) => ({
            source: s.data,
            dimensions: ["x", "y"],
        }));

        const xAxes = subplotIndices.map((_, i) => ({
            type: "value" as const,
            name: i === numSubplots - 1 ? "Time" : "",
            nameLocation: "center" as const,
            nameGap: 30,
            gridIndex: i,
            axisLabel: { show: i === numSubplots - 1 },
            axisLine: { lineStyle: { color: textColor } },
            splitLine: { lineStyle: { color: darkMode ? "#444" : "#e0e0e0" } },
        }));

        const yAxes = subplotIndices.map((sidx, i) => ({
            type: "value" as const,
            name: `Subplot ${sidx}`,
            nameLocation: "center" as const,
            nameGap: 45,
            nameRotate: 90,
            gridIndex: i,
            axisLine: { lineStyle: { color: textColor } },
            splitLine: { lineStyle: { color: darkMode ? "#444" : "#e0e0e0" } },
            nameTextStyle: { color: textColor, fontWeight: "bold" },
            axisTick: { show: true },
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
                lineStyle: { width: 2 },
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
                { type: "inside" as const, xAxisIndex: subplotIndices.map((_, i) => i), filterMode: "none" as const },
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

    resize() {
        if (this.instance) {
            this.instance.resize();
        }
    }

    getDataAtPixel(x: number, y: number) {
        if (!this.instance) return null;
        const coord = this.instance.convertFromPixel({ gridIndex: 0 }, [x, y]) as number[];
        return coord ? { x: coord[0], y: coord[1] } : null;
    }

    getPixelFromData(x: number, y: number) {
        if (!this.instance) return null;
        const pixel = this.instance.convertToPixel({ gridIndex: 0 }, [x, y]) as number[];
        return pixel ? { x: pixel[0], y: pixel[1] } : null;
    }

    destroy() {
        if (this.instance) {
            this.instance.dispose();
            this.instance = null;
        }
    }

    onLegendClick(handler: (seriesName: string, event: any) => void) {
        if (!this.instance) return;
        this.instance.on("legendselectchanged", (params: any) => {
            const option = this.instance!.getOption() as any;
            const selected = option.legend[0].selected || {};
            Object.keys(selected).forEach((name) => (selected[name] = true));
            this.instance!.setOption({ legend: { selected } });
            handler(params.name, params);
        });
    }

    onContextMenu(handler: (event: any) => void) {
        if (this.instance) {
            this.instance.getZr().on("contextmenu", handler);
        }
    }
}
