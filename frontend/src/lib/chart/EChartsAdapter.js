import * as echarts from "echarts";
import { ChartAdapter } from "./ChartAdapter.js";

/**
 * ECharts implementation of ChartAdapter.
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

        const isMulti = Array.isArray(seriesData);

        if (isMulti) {
            this._setMultiSeries(seriesData, title, textColor, bgColor, darkMode, getGridRight);
        } else {
            this._setSingleSeries(seriesData, title, textColor, bgColor, darkMode, getGridRight);
        }
    }

    _setSingleSeries(seriesInfo, title, textColor, bgColor, darkMode, getGridRight) {
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
                    dataZoom: {},
                    restore: {},
                    saveAsImage: {},
                },
                right: 20,
                iconStyle: { borderColor: textColor },
            },
            dataZoom: [
                { type: "inside", xAxisIndex: [0], filterMode: "none" },
                { type: "inside", yAxisIndex: [0], filterMode: "none" },
            ],
            legend: {
                data: [seriesInfo.name],
                orient: "vertical",
                right: 10,
                top: 60,
                textStyle: { color: textColor },
                type: "scroll",
                triggerEvent: true,
            },
            dataset: {
                source: seriesInfo.data,
                dimensions: ["x", "y"],
            },
            xAxis: {
                type: "value",
                name: "Time",
                nameLocation: "center",
                nameGap: 40,
                nameTextStyle: { color: textColor, fontWeight: "bold", fontSize: 16 },
                axisLine: { lineStyle: { color: textColor } },
                splitLine: { lineStyle: { color: darkMode ? "#444" : "#e0e0e0" } },
            },
            yAxis: {
                type: "value",
                name: "Value",
                nameLocation: "center",
                nameGap: 55,
                nameRotate: 90,
                nameTextStyle: { color: textColor, fontWeight: "bold", fontSize: 16 },
                axisLine: { lineStyle: { color: textColor } },
                splitLine: { lineStyle: { color: darkMode ? "#444" : "#e0e0e0" } },
            },
            series: [
                {
                    name: seriesInfo.name,
                    type: "line",
                    showSymbol: false,
                    encode: { x: "x", y: "y" },
                    large: true,
                    emphasis: { disabled: true },
                    color: seriesInfo.color,
                    lineStyle: { width: 2 },
                    sampling: "lttb",
                },
            ],
            grid: {
                containLabel: true,
                top: 60,
                bottom: 70,
                left: 80,
                right: getGridRight(seriesInfo),
            },
        };

        this.instance.setOption(option, { notMerge: true });
    }

    _setMultiSeries(seriesDataArray, title, textColor, bgColor, darkMode, getGridRight) {
        const datasets = seriesDataArray.map((s) => ({
            source: s.data,
            dimensions: ["x", "y"],
        }));

        const series = seriesDataArray.map((s, i) => ({
            name: s.name,
            type: "line",
            showSymbol: false,
            datasetIndex: i,
            encode: { x: "x", y: "y" },
            large: true,
            emphasis: { disabled: true },
            color: s.color,
            lineStyle: { width: 2 },
            sampling: "lttb",
        }));

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
                    dataZoom: {},
                    restore: {},
                    saveAsImage: {},
                },
                right: 20,
                iconStyle: { borderColor: textColor },
            },
            dataZoom: [
                { type: "inside", xAxisIndex: [0], filterMode: "none" },
                { type: "inside", yAxisIndex: [0], filterMode: "none" },
            ],
            legend: {
                data: seriesDataArray.map((s) => s.name),
                orient: "vertical",
                right: 10,
                top: 60,
                textStyle: { color: textColor },
                type: "scroll",
                triggerEvent: true,
            },
            dataset: datasets,
            xAxis: {
                type: "value",
                name: "Time",
                nameLocation: "center",
                nameGap: 40,
                nameTextStyle: { color: textColor, fontWeight: "bold", fontSize: 16 },
                axisLine: { lineStyle: { color: textColor } },
                splitLine: { lineStyle: { color: darkMode ? "#444" : "#e0e0e0" } },
            },
            yAxis: {
                type: "value",
                name: "Value",
                nameLocation: "center",
                nameGap: 55,
                nameRotate: 90,
                nameTextStyle: { color: textColor, fontWeight: "bold", fontSize: 16 },
                axisLine: { lineStyle: { color: textColor } },
                splitLine: { lineStyle: { color: darkMode ? "#444" : "#e0e0e0" } },
            },
            series: series,
            grid: {
                containLabel: true,
                top: 60,
                bottom: 70,
                left: 80,
                right: getGridRight(seriesDataArray),
            },
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
        const coord = this.instance.convertFromPixel("grid", [x, y]);
        return coord ? { x: coord[0], y: coord[1] } : null;
    }

    getPixelFromData(x, y) {
        if (!this.instance) return null;
        const pixel = this.instance.convertToPixel("grid", [x, y]);
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
            // Prevent default toggle behavior by restoring selection
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
