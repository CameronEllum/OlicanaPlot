import * as echarts from "echarts";
import { ChartAdapter } from "./ChartAdapter.js";

/**
 * ECharts implementation of ChartAdapter.
 * Implements true facets (subplots) by using multiple grid objects.
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

        // Group series by facetIndex
        const facetIndices = [...new Set(seriesArr.map((s) => s.facetIndex || 0))].sort((a, b) => a - b);
        const numFacets = facetIndices.length;

        console.log(`EChartsAdapter: Rendering ${numFacets} facets...`);

        // Map facetIndex to actual grid/axis index in ECharts
        const facetToIndexMap = {};
        facetIndices.forEach((fidx, i) => {
            facetToIndexMap[fidx] = i;
        });

        // Split the vertical space into N grids
        const totalHeight = 84;
        const facetHeight = totalHeight / numFacets;
        const gap = numFacets > 1 ? 5 : 0;

        const grids = facetIndices.map((_, i) => {
            const top = 10 + (i * facetHeight);
            return {
                left: 80,
                right: getGridRight(seriesArr),
                top: `${top}%`,
                height: `${facetHeight - gap}%`,
                containLabel: true
            };
        });

        const datasets = seriesArr.map((s) => ({
            source: s.data,
            dimensions: ["x", "y"],
        }));

        const xAxes = facetIndices.map((_, i) => ({
            type: "value",
            name: i === numFacets - 1 ? "Time" : "",
            nameLocation: "center",
            nameGap: 30,
            gridIndex: i,
            axisLabel: { show: i === numFacets - 1 },
            axisLine: { lineStyle: { color: textColor } },
            splitLine: { lineStyle: { color: darkMode ? "#444" : "#e0e0e0" } },
        }));

        const yAxes = facetIndices.map((fidx, i) => ({
            type: "value",
            name: `Facet ${fidx}`,
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
            const facetIdx = facetToIndexMap[s.facetIndex || 0];
            return {
                name: s.name,
                type: "line",
                showSymbol: false,
                datasetIndex: i,
                xAxisIndex: facetIdx,
                yAxisIndex: facetIdx,
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
                    dataZoom: { xAxisIndex: facetIndices.map((_, i) => i) },
                    restore: {},
                    saveAsImage: {},
                },
                right: 20,
                iconStyle: { borderColor: textColor },
            },
            dataZoom: [
                { type: "inside", xAxisIndex: facetIndices.map((_, i) => i), filterMode: "none" },
                { type: "slider", xAxisIndex: facetIndices.map((_, i) => i), bottom: 10, height: 20 },
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
        // In multi-grid, find coordinates for initial grid (usually where interaction happens)
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
