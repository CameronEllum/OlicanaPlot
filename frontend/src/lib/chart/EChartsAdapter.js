import * as echarts from "echarts";
import { ChartAdapter } from "./ChartAdapter.js";

/**
 * ECharts implementation of ChartAdapter.
 * Implements true subplots by using multiple grid objects stacked vertically.
 */
export class EChartsAdapter extends ChartAdapter {
    constructor() {
        super();
        this.instance = null;
        this.container = null;
    }

    init(container, darkMode) {
        this.container = container;
        if (this.instance) {
            this.instance.dispose();
        }
        this.instance = echarts.init(container, darkMode ? "dark" : null);
    }

    setData(seriesData, title, darkMode, getGridRight) {
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
        const subplotToIndexMap = {};
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
            type: "value",
            name: i === numSubplots - 1 ? "Time" : "",
            nameLocation: "center",
            nameGap: 30,
            gridIndex: i,
            axisLabel: { show: i === numSubplots - 1 },
            axisLine: { lineStyle: { color: textColor } },
            splitLine: { lineStyle: { color: darkMode ? "#444" : "#e0e0e0" } },
        }));

        const yAxes = subplotIndices.map((sidx, i) => ({
            type: "value",
            name: `Subplot ${sidx}`,
            nameLocation: "center",
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
                type: "line",
                showSymbol: false,
                datasetIndex: i,
                xAxisIndex: subplotIdx,
                yAxisIndex: subplotIdx,
                encode: { x: "x", y: "y" },
                large: true,
                emphasis: { disabled: true },
                color: s.color,
                lineStyle: { width: 2 },
                sampling: "lttb",
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
            tooltip: { trigger: "axis" },
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
                { type: "inside", xAxisIndex: subplotIndices.map((_, i) => i), filterMode: "none" },
            ],
            legend: {
                data: seriesArr.map((s) => s.name),
                orient: "vertical",
                right: 10,
                top: 60,
                textStyle: { color: textColor },
                type: "scroll",
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

    getDataAtPixel(x, y) {
        if (!this.instance) return null;
        const coord = this.instance.convertFromPixel({ gridIndex: 0 }, [x, y]);
        return coord ? { x: coord[0], y: coord[1] } : null;
    }

    getPixelFromData(x, y) {
        if (!this.instance) return null;
        const pixel = this.instance.convertToPixel({ gridIndex: 0 }, [x, y]);
        return pixel ? { x: pixel[0], y: pixel[1] } : null;
    }

    destroy() {
        if (this.instance) {
            this.instance.dispose();
            this.instance = null;
        }
    }

    onLegendClick(handler) {
        if (!this.instance) return;
        this.instance.on("legendselectchanged", (params) => {
            const option = this.instance.getOption();
            const selected = option.legend[0].selected || {};
            Object.keys(selected).forEach((name) => (selected[name] = true));
            this.instance.setOption({ legend: { selected } });
            handler(params.name, params);
        });
    }

    onContextMenu(handler) {
        if (this.instance) {
            this.instance.getZr().on("contextmenu", handler);
        }
    }
}
