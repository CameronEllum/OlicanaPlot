/**
 * ChartAdapter interface for abstracting chart libraries.
 * Each implementation must provide these methods.
 */
export interface SeriesConfig {
    id: string;
    name: string;
    color: string;
    data: Float64Array;
    subplotIndex?: number;
}

export abstract class ChartAdapter {
    /**
     * Initialize the chart in a container.
     * @param container - DOM element to render into
     * @param darkMode - Whether dark theme is active
     */
    abstract init(container: HTMLElement, darkMode: boolean): void;

    /**
     * Render series data on the chart.
     * @param seriesData - Series data with Float64Array x/y values
     * @param title - Chart title
     * @param darkMode - Whether dark theme is active
     * @param getGridRight - Function to calculate right margin
     */
    abstract setData(seriesData: SeriesConfig[], title: string, darkMode: boolean, getGridRight: (data: SeriesConfig[]) => number): void;

    /**
     * Handle container resize.
     */
    abstract resize(): void;

    /**
     * Convert pixel coordinates to data coordinates.
     * @param x - Pixel X
     * @param y - Pixel Y
     * @returns {{x: number, y: number}|null}
     */
    abstract getDataAtPixel(x: number, y: number): { x: number, y: number } | null;

    /**
     * Get pixel coordinates from data values.
     * @param x - Data X
     * @param y - Data Y
     * @returns {{x: number, y: number}|null}
     */
    abstract getPixelFromData(x: number, y: number): { x: number, y: number } | null;

    /**
     * Clean up resources.
     */
    abstract destroy(): void;

    /**
     * Register a legend click handler.
     * @param handler - Callback with (seriesName, event)
     */
    abstract onLegendClick(handler: (seriesName: string, event: any) => void): void;

    /**
     * Register a context menu handler.
     * @param handler - Callback with (event)
     */
    abstract onContextMenu(handler: (event: any) => void): void;
}
