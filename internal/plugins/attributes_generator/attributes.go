// Package attributes_generator provides a plugin to demonstrate line attributes.
package attributes_generator

import (
	"fmt"
	"math"
	"olicanaplot/internal/logging"
	"olicanaplot/internal/plugins"
	"strings"
)

const pluginName = "Attributes Demo"

// Plugin implements the attributes generator.
type Plugin struct {
	logger logging.Logger
}

// New creates a new attributes plugin.
func New() *Plugin {
	return &Plugin{}
}

// Name returns the display name of the plugin.
func (p *Plugin) Name() string {
	return pluginName
}

// Version returns the API version.
func (p *Plugin) Version() uint32 {
	return plugins.PluginAPIVersion
}

// Path returns an empty string for internal plugins.
func (p *Plugin) Path() string {
	return ""
}

// GetFilePatterns returns the list of file patterns supported by the plugin.
func (p *Plugin) GetFilePatterns() []plugins.FilePattern {
	return nil
}

// Initialize sets up the plugin.
func (p *Plugin) Initialize(ctx interface{}, initStr string, logger logging.Logger) (string, error) {
	p.logger = logger
	logger.Debug("Attributes demo plugin initialized")
	return "{}", nil
}

// GetChartConfig returns chart display configuration.
func (p *Plugin) GetChartConfig(args string) (*plugins.ChartConfig, error) {
	return &plugins.ChartConfig{
		Title:      "Line Attributes Demonstration",
		AxisLabels: []string{"Time", "Value"},
		Rows:       2,
		Cols:       2,
	}, nil
}

// GetSeriesConfig returns the list of available series.
func (p *Plugin) GetSeriesConfig() ([]plugins.SeriesConfig, error) {
	return []plugins.SeriesConfig{
		// Subplot (0,0) - Line Types
		{
			ID:       "types_0",
			Name:     "Solid Line",
			Subplot:  []int{0, 0},
			LineType: "solid",
		},
		{
			ID:       "types_1",
			Name:     "Dashed Line",
			Subplot:  []int{0, 0},
			LineType: "dashed",
		},
		{
			ID:       "types_2",
			Name:     "Dotted Line",
			Subplot:  []int{0, 0},
			LineType: "dotted",
		},
		// Subplot (0,1) - Marker Sizes & Fills
		{
			ID:         "markers_0",
			Name:       "Small Filled (4px)",
			Subplot:    []int{0, 1},
			MarkerType: "circle",
			MarkerSize: floatPtr(4.0),
		},
		{
			ID:         "markers_1",
			Name:       "Medium Empty (8px)",
			Subplot:    []int{0, 1},
			MarkerType: "square",
			MarkerSize: floatPtr(8.0),
			MarkerFill: "empty",
		},
		{
			ID:         "markers_2",
			Name:       "Large Filled (14px)",
			Subplot:    []int{0, 1},
			MarkerType: "triangle",
			MarkerSize: floatPtr(14.0),
		},
		// Subplot (1,1) - Line Widths
		{
			ID:        "widths_0",
			Name:      "Thin (1px)",
			Subplot:   []int{1, 0},
			LineWidth: floatPtr(1.0),
		},
		{
			ID:        "widths_1",
			Name:      "Medium (4px)",
			Subplot:   []int{1, 0},
			LineWidth: floatPtr(4.0),
		},
		{
			ID:        "widths_2",
			Name:      "Thick (8px)",
			Subplot:   []int{1, 0},
			LineWidth: floatPtr(8.0),
		},
		// Subplot (1,1) - Opacity Variations
		{
			ID:        "opacity_0",
			Name:      "Full Opacity (100%)",
			Subplot:   []int{1, 1},
			Color:     "rgba(255, 0, 0, 1.0)",
			LineWidth: floatPtr(4.0),
		},
		{
			ID:        "opacity_1",
			Name:      "Medium Opacity (50%)",
			Subplot:   []int{1, 1},
			Color:     "rgba(255, 0, 0, 0.5)",
			LineWidth: floatPtr(4.0),
		},
		{
			ID:        "opacity_2",
			Name:      "Low Opacity (25%)",
			Subplot:   []int{1, 1},
			Color:     "rgba(255, 0, 0, 0.25)",
			LineWidth: floatPtr(4.0),
		},
	}, nil
}

func floatPtr(f float64) *float64 {
	return &f
}

// GetSeriesData generates and returns data.
func (p *Plugin) GetSeriesData(seriesID string, preferredStorage string) ([]float64, string, error) {
	var index int
	if strings.HasPrefix(seriesID, "types_") {
		fmt.Sscanf(seriesID, "types_%d", &index)
	} else if strings.HasPrefix(seriesID, "widths_") {
		fmt.Sscanf(seriesID, "widths_%d", &index)
	} else if strings.HasPrefix(seriesID, "markers_") {
		fmt.Sscanf(seriesID, "markers_%d", &index)
	} else if strings.HasPrefix(seriesID, "opacity_") {
		fmt.Sscanf(seriesID, "opacity_%d", &index)
	}

	if p.logger != nil {
		p.logger.Info("Attributes plugin data request", "seriesID", seriesID, "index", index)
	}

	numPoints := 400
	result := make([]float64, numPoints*2)
	isArrays := preferredStorage == "arrays"
	storage := "interleaved"
	if isArrays {
		storage = "arrays"
	}

	offset := float64(index) * math.Pi / 4.0

	for i := 0; i < numPoints; i++ {
		t := float64(i) * 0.1
		y := math.Sin(t + offset)

		if isArrays {
			result[i] = t
			result[numPoints+i] = y
		} else {
			result[i*2] = t
			result[i*2+1] = y
		}
	}

	return result, storage, nil
}

// Close cleans up plugin resources.
func (p *Plugin) Close() error {
	return nil
}
