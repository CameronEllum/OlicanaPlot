/**
 * ChartAdapter interface for abstracting chart libraries.
 * Each implementation must provide these methods.
 */
export class ChartAdapter {
    /**
     * Initialize the chart in a container.
     * @param {HTMLElement} container - DOM element to render into
     * @param {boolean} darkMode - Whether dark theme is active
     */
    init(container, darkMode) {
        throw new Error("Not implemented");
    }

    /**
     * Render series data on the chart.
     * @param {Array|Object} seriesData - Series data with Float64Array x/y values
     * @param {string} title - Chart title
     * @param {boolean} darkMode - Whether dark theme is active
     * @param {Function} getGridRight - Function to calculate right margin
     */
    setData(seriesData, title, darkMode, getGridRight) {
        throw new Error("Not implemented");
    }

    /**
     * Handle container resize.
     */
    resize() {
        throw new Error("Not implemented");
    }

    /**
     * Convert pixel coordinates to data coordinates.
     * @param {number} x - Pixel X
     * @param {number} y - Pixel Y
     * @returns {{x: number, y: number}|null}
     */
    getDataAtPixel(x, y) {
        throw new Error("Not implemented");
    }

    /**
     * Get pixel coordinates from data values.
     * @param {number} x - Data X
     * @param {number} y - Data Y
     * @returns {{x: number, y: number}|null}
     */
    getPixelFromData(x, y) {
        throw new Error("Not implemented");
    }

    /**
     * Clean up resources.
     */
    destroy() {
        throw new Error("Not implemented");
    }

    /**
     * Register a legend click handler.
     * @param {Function} handler - Callback with (seriesName, event)
     */
    onLegendClick(handler) {
        throw new Error("Not implemented");
    }

    /**
     * Register a context menu handler.
     * @param {Function} handler - Callback with (event)
     */
    onContextMenu(handler) {
        throw new Error("Not implemented");
    }
}
