import Plotly from "plotly.js-dist-min";
import { ChartAdapter } from "./ChartAdapter.js";

/**
 * Plotly.js implementation of ChartAdapter using WebGL (scattergl).
 */
export class PlotlyAdapter extends ChartAdapter {
    constructor() {
        super();
        this.container = null;
        this.darkMode = false;
        this.currentData = null;
    }

    init(container, darkMode) {
        this.container = container;
        this.darkMode = darkMode;
    }

    setData(seriesData, title, darkMode, getGridRight) {
        if (!this.container) return;

        this.darkMode = darkMode;
        this.currentData = seriesData;

        // Always expect an array of series
        const seriesArr = Array.isArray(seriesData) ? seriesData : [seriesData];

        const traces = seriesArr.map((s) => {
            const pointCount = s.data.length / 2;
            // Zero-copy: just slice the buffer. Application ensures data is in track format.
            const xData = s.data.subarray(0, pointCount);
            const yData = s.data.subarray(pointCount);

            return {
                x: xData,
                y: yData,
                name: s.name,
                type: "scattergl",
                mode: "lines",
                line: {
                    color: s.color,
                    width: 2,
                },
                hoverinfo: "x+y+name",
            };
        });

        const textColor = darkMode ? "#ccc" : "#333";
        const bgColor = darkMode ? "#2b2b2b" : "#ffffff";
        const gridColor = darkMode ? "#444" : "#e0e0e0";

        const layout = {
            title: {
                text: title,
                font: { color: textColor },
                x: 0.5,
                xanchor: "center",
            },
            paper_bgcolor: bgColor,
            plot_bgcolor: bgColor,
            font: { color: textColor },
            xaxis: {
                title: {
                    text: "Time",
                    font: { size: 16, color: textColor },
                },
                gridcolor: gridColor,
                zerolinecolor: gridColor,
                tickfont: { color: textColor },
            },
            yaxis: {
                title: {
                    text: "Value",
                    font: { size: 16, color: textColor },
                },
                gridcolor: gridColor,
                zerolinecolor: gridColor,
                tickfont: { color: textColor },
            },
            showlegend: true,
            legend: {
                orientation: "v",
                x: 1.02,
                y: 1,
                font: { color: textColor },
            },
            margin: {
                l: 80,
                r: getGridRight(seriesArr),
                t: 60,
                b: 70,
            },
            dragmode: "pan",
        };

        const config = {
            responsive: true,
            scrollZoom: true,
            displayModeBar: true,
            modeBarButtonsToRemove: ["lasso2d", "select2d"],
        };

        Plotly.react(this.container, traces, layout, config);

        // Attach legend context menu listeners after plot/re-plot
        // We use a one-time event or just re-attach on every afterplot
        this.container.removeAllListeners("plotly_afterplot");
        this.container.on("plotly_afterplot", () => {
            const legendItems = this.container.querySelectorAll(".legendtext, .legendtoggle");
            legendItems.forEach((el) => {
                el.oncontextmenu = (e) => {
                    e.preventDefault();
                    e.stopPropagation();

                    // Extract trace name from the specific trace group
                    const traceGroup = el.closest(".traces");
                    const textEl = traceGroup ? traceGroup.querySelector(".legendtext") : null;
                    const traceName = textEl ? textEl.textContent.trim() : null;

                    if (traceName && this.contextMenuHandler) {
                        // Pass a synthetic event-like object
                        this.contextMenuHandler({
                            event: e,
                            plotlyLegendName: traceName,
                        });
                    }
                };
            });
        });
    }

    resize() {
        if (this.container) {
            Plotly.Plots.resize(this.container);
        }
    }

    getDataAtPixel(x, y) {
        // Plotly doesn't have a direct pixel-to-data API like ECharts
        // We can approximate using the axis range and container dimensions
        if (!this.container || !this.container._fullLayout) return null;

        const layout = this.container._fullLayout;
        const xaxis = layout.xaxis;
        const yaxis = layout.yaxis;

        if (!xaxis || !yaxis) return null;

        // Get plot area bounds
        const plotLeft = xaxis._offset;
        const plotWidth = xaxis._length;
        const plotTop = yaxis._offset;
        const plotHeight = yaxis._length;

        // Check if point is in plot area
        if (x < plotLeft || x > plotLeft + plotWidth) return null;
        if (y < plotTop || y > plotTop + plotHeight) return null;

        // Convert pixel to data
        const dataX = xaxis.p2d(x - plotLeft);
        const dataY = yaxis.p2d(plotTop + plotHeight - y);

        return { x: dataX, y: dataY };
    }

    getPixelFromData(x, y) {
        if (!this.container || !this.container._fullLayout) return null;

        const layout = this.container._fullLayout;
        const xaxis = layout.xaxis;
        const yaxis = layout.yaxis;

        if (!xaxis || !yaxis) return null;

        const pixelX = xaxis._offset + xaxis.d2p(x);
        const pixelY = yaxis._offset + yaxis._length - yaxis.d2p(y);

        return { x: pixelX, y: pixelY };
    }

    destroy() {
        if (this.container) {
            Plotly.purge(this.container);
        }
    }

    onLegendClick(handler) {
        if (!this.container) return;
        this.container.on("plotly_legendclick", (event) => {
            handler(event.data[event.curveNumber].name, event);
            return false; // Prevent default toggle
        });
    }

    onContextMenu(handler) {
        this.contextMenuHandler = handler;
        if (this.container) {
            this.container.addEventListener("contextmenu", handler);
        }
    }
}
