// Package plugins defines the plugin interface and types for OlicanaPlot.
package plugins

import (
	"olicanaplot/internal/logging"
)

// PluginAPIVersion is the current API version for compatibility checking.
const PluginAPIVersion uint32 = 1

// IMPORTANT: The following structs are intentionally duplicated from pkg/sdk
// instead of using type aliases. This is a workaround for a Wails 3 binding
// generation bug where cross-package type aliases result in broken JavaScript
// imports (e.g., undefined "$0" references).
//
// If you modify these, you MUST also modify the corresponding structs in
// pkg/sdk/protocol.go and ensure internal/plugins/sync_test.go passes.

// ChartConfig holds chart display configuration.
type ChartConfig struct {
	Title      string            `json:"title"`
	AxisLabels []string          `json:"axis_labels"`
	LineWidth  *float64          `json:"line_width,omitempty"`
	Axes       []AxisGroupConfig `json:"axes,omitempty"`
	LinkX      *bool             `json:"link_x,omitempty"`
	LinkY      *bool             `json:"link_y,omitempty"`
	Rows       int               `json:"rows,omitempty"`
	Cols       int               `json:"cols,omitempty"`
}

// AxisConfig describes an axis within a subplot.
type AxisConfig struct {
	Title    string   `json:"title,omitempty"`
	Position string   `json:"position,omitempty"` // "bottom", "top", "left", "right"
	Unit     string   `json:"unit,omitempty"`
	Type     string   `json:"type,omitempty"` // "linear", "log", "date"
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
}

// AxisGroupConfig describes all axes and series for one subplot cell.
type AxisGroupConfig struct {
	Title   string       `json:"title,omitempty"`
	Subplot []int        `json:"subplot"` // [row, col]
	XAxes   []AxisConfig `json:"x_axes,omitempty"`
	YAxes   []AxisConfig `json:"y_axes,omitempty"`
}

// SeriesConfig describes a data series metadata.
type SeriesConfig struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Color      string   `json:"color,omitempty"`
	Subplot    []int    `json:"subplot,omitempty"`   // [row, col]
	LineType   string   `json:"line_type,omitempty"` // "solid", "dashed", "dotted"
	LineWidth  *float64 `json:"line_width,omitempty"`
	MarkerType string   `json:"marker_type,omitempty"` // "none", "circle", "square", "triangle", "diamond", "cross", "x"
	Unit       string   `json:"unit,omitempty"`
	Visible    *bool    `json:"visible,omitempty"`
	YAxis      string   `json:"y_axis,omitempty"` // references Y axis title
}

// FilePattern describes a file type supported by a plugin.
type FilePattern struct {
	Description string   `json:"description"`
	Patterns    []string `json:"patterns"`
}

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
