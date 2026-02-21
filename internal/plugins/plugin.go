// Package plugins defines the plugin interface and types for OlicanaPlot.
package plugins

import (
	"fmt"
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
	Title string            `json:"title"`
	Grid  *GridConfig       `json:"grid,omitempty"`
	Axes  []AxisGroupConfig `json:"axes,omitempty"`
	LinkX *bool             `json:"link_x,omitempty"`
	LinkY *bool             `json:"link_y,omitempty"`
}

// GridConfig describes the subplot grid layout.
type GridConfig struct {
	Rows int `json:"rows"`
	Cols int `json:"cols"`
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

// SubPlot describes a cell in the chart grid.
type SubPlot struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

// AxisGroupConfig describes all axes and series for one subplot cell.
type AxisGroupConfig struct {
	Title   string       `json:"title,omitempty"`
	Subplot *SubPlot     `json:"subplot,omitempty"`
	XAxes   []AxisConfig `json:"x_axes,omitempty"`
	YAxes   []AxisConfig `json:"y_axes,omitempty"`
}

// SeriesConfig describes a data series metadata.
type SeriesConfig struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Color      string   `json:"color,omitempty"`
	Subplot    *SubPlot `json:"subplot,omitempty"`
	LineType   string   `json:"line_type,omitempty"` // "solid", "dashed", "dotted"
	LineWidth  *float64 `json:"line_width,omitempty"`
	MarkerType string   `json:"marker_type,omitempty"` // "none", "circle", "square", "triangle", "diamond", "cross", "x"
	MarkerSize *float64 `json:"marker_size,omitempty"`
	MarkerFill string   `json:"marker_fill,omitempty"` // "empty" or "solid"
	Unit       string   `json:"unit,omitempty"`
	Visible    *bool    `json:"visible"`
	YAxis      string   `json:"y_axis,omitempty"` // references Y axis title
}

// SetDefaults ensures all required fields have sensible defaults if they are empty
func (s *SeriesConfig) SetDefaults() {
	if s.Subplot == nil {
		s.Subplot = &SubPlot{Row: 0, Col: 0}
	}
	if s.LineType == "" {
		s.LineType = "solid"
	}
	if s.LineWidth == nil {
		w := 2.0
		s.LineWidth = &w
	}
	if s.MarkerType == "" {
		s.MarkerType = "none"
	}
	if s.MarkerSize == nil {
		sz := 8.0
		s.MarkerSize = &sz
	}
	if s.MarkerFill == "" {
		s.MarkerFill = "solid"
	}
	if s.Visible == nil {
		v := true
		s.Visible = &v
	}
}

// SetDefaults ensures all required fields have sensible defaults
func (a *AxisConfig) SetDefaults(defaultTitle string, defaultPos string) {
	if a.Title == "" {
		a.Title = defaultTitle
	}
	if a.Position == "" {
		a.Position = defaultPos
	}
	if a.Type == "" {
		a.Type = "linear"
	}
}

// SetDefaults ensures all sub-configs have defaults
func (ag *AxisGroupConfig) SetDefaults() {
	if ag.Subplot == nil {
		ag.Subplot = &SubPlot{Row: 0, Col: 0}
	}
	if len(ag.XAxes) == 0 {
		ag.XAxes = []AxisConfig{{Title: "X", Position: "bottom", Type: "linear"}}
	}
	if len(ag.YAxes) == 0 {
		ag.YAxes = []AxisConfig{{Title: "Y", Position: "left", Type: "linear"}}
	}
	for i := range ag.XAxes {
		title := "X"
		if i > 0 {
			title = fmt.Sprintf("X%d", i+1)
		}
		ag.XAxes[i].SetDefaults(title, "bottom")
	}
	for i := range ag.YAxes {
		title := "Y"
		if i > 0 {
			title = fmt.Sprintf("Y%d", i+1)
		}
		ag.YAxes[i].SetDefaults(title, "left")
	}
}

// SetDefaults ensures all sub-configs have defaults
func (c *ChartConfig) SetDefaults() {
	if len(c.Axes) == 0 {
		c.Axes = []AxisGroupConfig{
			{
				Subplot: &SubPlot{Row: 0, Col: 0},
				XAxes:   []AxisConfig{{Title: "X", Position: "bottom", Type: "linear"}},
				YAxes:   []AxisConfig{{Title: "Y", Position: "left", Type: "linear"}},
			},
		}
	}

	// Calculate grid dimensions from axes if not provided
	maxRow := 0
	maxCol := 0
	for _, ag := range c.Axes {
		row := 0
		col := 0
		if ag.Subplot != nil {
			row = ag.Subplot.Row
			col = ag.Subplot.Col
		}
		if row > maxRow {
			maxRow = row
		}
		if col > maxCol {
			maxCol = col
		}
	}

	if c.Grid == nil {
		c.Grid = &GridConfig{
			Rows: maxRow + 1,
			Cols: maxCol + 1,
		}
	} else {
		if c.Grid.Rows <= maxRow {
			c.Grid.Rows = maxRow + 1
		}
		if c.Grid.Cols <= maxCol {
			c.Grid.Cols = maxCol + 1
		}
	}

	// Ensure all cells in the grid have axis configurations
	axisMap := make(map[string]bool)
	for _, ag := range c.Axes {
		row := 0
		col := 0
		if ag.Subplot != nil {
			row = ag.Subplot.Row
			col = ag.Subplot.Col
		}
		axisMap[fmt.Sprintf("%d,%d", row, col)] = true
	}

	for r := 0; r < c.Grid.Rows; r++ {
		for col := 0; col < c.Grid.Cols; col++ {
			key := fmt.Sprintf("%d,%d", r, col)
			if !axisMap[key] {
				c.Axes = append(c.Axes, AxisGroupConfig{
					Subplot: &SubPlot{Row: r, Col: col},
					XAxes:   []AxisConfig{{Title: "X", Position: "bottom", Type: "linear"}},
					YAxes:   []AxisConfig{{Title: "Y", Position: "left", Type: "linear"}},
				})
			}
		}
	}

	for i := range c.Axes {
		c.Axes[i].SetDefaults()
	}
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
