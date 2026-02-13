// Package plugins defines the plugin interface and types for OlicanaPlot.
package plugins

import (
	"olicanaplot/internal/logging"
	"olicanaplot/pkg/ipcplugin"
)

// PluginAPIVersion is the current API version for compatibility checking.
const PluginAPIVersion uint32 = 1

// ChartColors provides a Plotly-inspired color palette for series.
var ChartColors = ipcplugin.ChartColors

// ChartConfig contains chart display configuration.
type ChartConfig = ipcplugin.ChartConfig

// AxisConfig describes an axis within a subplot.
type AxisConfig = ipcplugin.AxisConfig

// AxisGroupConfig describes all axes and series for one subplot cell.
type AxisGroupConfig = ipcplugin.AxisGroupConfig

// SeriesConfig describes a data series available from a plugin.
type SeriesConfig = ipcplugin.SeriesConfig

// FilePattern describes a file type supported by a plugin.
type FilePattern = ipcplugin.FilePattern

// Plugin is the interface that all data source plugins must implement.
type Plugin interface {
	// Name returns the display name of the plugin.
	Name() string

	// Version returns the API version the plugin implements.
	Version() uint32

	// Path returns the path to the plugin executable if it's an external plugin.
	// Returns an empty string for internal plugins.
	Path() string

	// GetFilePatterns returns the list of file patterns supported by the plugin.
	// Returns nil if the plugin is not a file loader.
	GetFilePatterns() []FilePattern

	// Initialize executes plugin initialization and configuration.
	// Plugins may spawn Wails3 modal dialogs for user configuration.
	// The ctx parameter can be cast to the appropriate Wails context type
	// (e.g., *application.WebviewWindow) to access dialog functionality.
	// The logger parameter provides structured logging capabilities.
	// Result is a JSON result string or an error.
	Initialize(ctx interface{}, initStr string, logger logging.Logger) (string, error)

	// GetChartConfig returns chart configuration for display.
	GetChartConfig(args string) (*ChartConfig, error)

	// GetSeriesConfig returns the list of available data series.
	GetSeriesConfig() ([]SeriesConfig, error)

	// GetSeriesData returns binary float64 data for the specified series ID.
	// preferredStorage parameter: "interleaved" or "arrays" ([x...][y...]).
	// Returns the data and the actual storage format used.
	GetSeriesData(seriesID string, preferredStorage string) ([]float64, string, error)

	// Close cleans up plugin resources. Called on shutdown.
	Close() error
}
