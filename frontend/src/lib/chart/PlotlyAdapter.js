import Plotly from "plotly.js-dist-min";
import { ChartAdapter } from "./ChartAdapter.js";

/**
 * Plotly.js implementation of ChartAdapter using WebGL (scattergl).
 * Implements true facets (subplots) by dynamically partitioning the Y domain.
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

        // Group series by facetIndex
        const facetIndices = [...new Set(seriesArr.map((s) => s.facetIndex || 0))].sort((a, b) => a - b);
        const numFacets = facetIndices.length;

        console.log(`PlotlyAdapter: Rendering ${numFacets} facets...`);

        // Map facetIndex to Plotly axis IDs (x, y, x2, y2, etc.)
        const facetToAxisMap = {};
        facetIndices.forEach((fidx, i) => {
            const suffix = i === 0 ? "" : (i + 1).toString();
            facetToAxisMap[fidx] = {
                x: `x${suffix}`,
                y: `y${suffix}`,
                xaxisKey: `xaxis${suffix}`,
                yaxisKey: `yaxis${suffix}`
            };
        });

        // Create traces and link to their respective axes
        const traces = seriesArr.map((s) => {
            const pointCount = s.data.length / 2;
            const xData = s.data.subarray(0, pointCount);
            const yData = s.data.subarray(pointCount);
            const axes = facetToAxisMap[s.facetIndex || 0];

            return {
                x: xData,
                y: yData,
                xaxis: axes.x,
                yaxis: axes.y,
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
            showlegend: true,
            legend: {
                orientation: "v",
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
            dragmode: "pan",
        };

        // Calculate vertical domains for the subplot grid
        const gap = 0.06; // gap between plots
        const h = (1.0 - (numFacets - 1) * gap) / numFacets;

        facetIndices.forEach((fidx, i) => {
            const axes = facetToAxisMap[fidx];
            // Stack from top down. Row 0 is at top.
            const rowTop = 1.0 - (i * (h + gap));
            const rowBottom = rowTop - h;

            // X-axis for this row
            layout[axes.xaxisKey] = {
                title: i === numFacets - 1 ? { text: "Time", font: { size: 16, color: textColor } } : undefined,
                gridcolor: gridColor,
                zerolinecolor: gridColor,
                tickfont: { color: textColor },
                anchor: axes.y,
                matches: i === 0 ? undefined : "x", // Link all X axes for synchronized zoom/pan
                showticklabels: i === numFacets - 1,
            };

            // Y-axis for this row
            layout[axes.yaxisKey] = {
                title: { text: `Facet ${fidx}`, font: { size: 14, color: textColor } },
                gridcolor: gridColor,
                zerolinecolor: gridColor,
                tickfont: { color: textColor },
                domain: [Math.max(0, rowBottom), Math.min(1, rowTop)],
                anchor: axes.x,
            };
        });

        const config = {
            responsive: true,
            scrollZoom: true,
            displayModeBar: true,
            modeBarButtonsToRemove: ["lasso2d", "select2d"],
        };

        // Use react to update without destroying state if possible
        Plotly.react(this.container, traces, layout, config);

        // Attach legend context menu listeners
        this.container.removeAllListeners("plotly_afterplot");
        this.container.on("plotly_afterplot", () => {
            const legendItems = this.container.querySelectorAll(".legendtext, .legendtoggle");
            legendItems.forEach((el) => {
                el.oncontextmenu = (e) => {
                    e.preventDefault();
                    e.stopPropagation();

                    const traceGroup = el.closest(".traces");
                    const textEl = traceGroup ? traceGroup.querySelector(".legendtext") : null;
                    const traceName = textEl ? textEl.textContent.trim() : null;

                    if (traceName && this.contextMenuHandler) {
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
        if (!this.container || !this.container._fullLayout) return null;
        const layout = this.container._fullLayout;
        // In multi-subplot, we need to find the specific axis pair.
        // For simplicity, find the first available pair.
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
            return false;
        });
    }

    onContextMenu(handler) {
        this.contextMenuHandler = handler;
        if (this.container) {
            this.container.addEventListener("contextmenu", handler);
        }
    }
}
