import Plotly from "plotly.js-dist-min";
import { ChartAdapter, type SeriesConfig } from "./ChartAdapter.ts";

// Plotly.js implementation of ChartAdapter using WebGL (scattergl).
// Implements true subplots by dynamically partitioning the Y domain.
export class PlotlyAdapter extends ChartAdapter {
    public container: any = null;
    public darkMode: boolean = false;
    public currentData: SeriesConfig[] | null = null;
    public lastSubplotCount: number = 0;
    private contextMenuHandler: ((event: any) => void) | null = null;

    constructor() {
        super();
    }

    // Store the target container and initial theme for the Plotly instance.
    init(container: HTMLElement, darkMode: boolean) {
        this.container = container;
        this.darkMode = darkMode;
    }

    // Prepare the data traces and layout configuration, then render or update the
    // Plotly chart including subplot partitioning.
    setData(seriesData: SeriesConfig[], title: string, darkMode: boolean, getGridRight: (data: SeriesConfig[]) => number) {
        if (!this.container) return;

        this.darkMode = darkMode;
        this.currentData = seriesData;

        // Always expect an array of series
        const seriesArr = Array.isArray(seriesData) ? seriesData : [seriesData];

        // Group series by subplotIndex
        const subplotIndices = [...new Set(seriesArr.map((s) => s.subplotIndex || 0))].sort((a, b) => a - b);
        const numSubplots = subplotIndices.length;

        console.log(`[PlotlyAdapter] Rendering subplots:`, subplotIndices);

        // Map subplotIndex to Plotly axis labels
        const subplotToAxisMap: Record<number, any> = {};
        subplotIndices.forEach((sidx, i) => {
            const axisNum = i === 0 ? "" : (i + 1).toString();
            subplotToAxisMap[sidx] = {
                x: `x${axisNum}`,
                y: `y${axisNum}`,
                xaxisKey: `xaxis${axisNum}`,
                yaxisKey: `yaxis${axisNum}`
            };
        });

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
                    width: 2,
                },
                hoverinfo: "x+y+name",
            };
        });

        const textColor = darkMode ? "#ccc" : "#333";
        const bgColor = darkMode ? "#2b2b2b" : "#ffffff";
        const gridColor = darkMode ? "#444" : "#e0e0e0";

        const layout: any = {
            title: {
                text: title,
                font: { color: textColor },
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

        subplotIndices.forEach((sidx, i) => {
            const axes = subplotToAxisMap[sidx];
            // Stack from top down
            const rowTop = 1.0 - (i * (h + gap));
            const rowBottom = rowTop - h;

            layout[axes.xaxisKey] = {
                title: i === numSubplots - 1 ? { text: "Time", font: { size: 16, color: textColor } } : undefined,
                gridcolor: gridColor,
                zerolinecolor: gridColor,
                tickfont: { color: textColor },
                anchor: axes.y,
                matches: i === 0 ? undefined : "x", // link for sync zoom
                showticklabels: i === numSubplots - 1,
            };

            layout[axes.yaxisKey] = {
                title: { text: `Subplot ${sidx}`, font: { size: 14, color: textColor } },
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

        // If the number of subplots changed, purge the plot to ensure a clean layout update
        if (numSubplots !== this.lastSubplotCount) {
            console.log(`[PlotlyAdapter] Subplot count changed from ${this.lastSubplotCount} to ${numSubplots}. Purging plot.`);
            Plotly.purge(this.container);
            this.lastSubplotCount = numSubplots;
        }

        Plotly.react(this.container, traces, layout, config);

        // Legend context menu logic
        this.container.removeAllListeners("plotly_afterplot");
        this.container.on("plotly_afterplot", () => {
            const legendItems = this.container.querySelectorAll(".legendtext, .legendtoggle");
            legendItems.forEach((el: any) => {
                el.oncontextmenu = (e: MouseEvent) => {
                    e.preventDefault();
                    e.stopPropagation();

                    const traceGroup = el.closest(".traces") as HTMLElement;
                    const textEl = traceGroup ? traceGroup.querySelector(".legendtext") : null;
                    const traceName = textEl ? textEl.textContent?.trim() : null;

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
        if (!this.container) return;
        this.container.on("plotly_legendclick", (event: any) => {
            handler(event.data[event.curveNumber].name, event);
            return false;
        });
    }

    // Register a handler for the browser's context menu event on the Plotly
    // container.
    onContextMenu(handler: (event: any) => void) {
        this.contextMenuHandler = handler;
        if (this.container) {
            this.container.addEventListener("contextmenu", handler);
        }
    }
}
