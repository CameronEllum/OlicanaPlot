// Package sine provides a sine wave generator plugin for OlicanaPlot.
package sine

import (
	"math"
	"olicanaplot/internal/logging"
	"olicanaplot/internal/plugins"
)

const pluginName = "Sine Wave"

// Plugin implements the sine wave generator.
type Plugin struct{}

// New creates a new sine wave plugin.
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

// GetFilePatterns returns the list of file patterns supported by the plugin.
func (p *Plugin) GetFilePatterns() []plugins.FilePattern {
	return nil
}

// Initialize sets up the plugin. No configuration needed for sine wave.
func (p *Plugin) Initialize(ctx interface{}, initStr string, logger logging.Logger) (string, error) {
	logger.Debug("Sine wave plugin initialized")
	return "{}", nil
}

// GetChartConfig returns chart display configuration.
func (p *Plugin) GetChartConfig(args string) (*plugins.ChartConfig, error) {
	return &plugins.ChartConfig{
		Title:      "Sine Wave",
		AxisLabels: []string{"Degrees", "Amplitude"},
	}, nil
}

// GetSeriesConfig returns the list of available series.
func (p *Plugin) GetSeriesConfig() ([]plugins.SeriesConfig, error) {
	return []plugins.SeriesConfig{
		{
			ID:    "sine_0",
			Name:  "Sine Wave",
			Color: plugins.ChartColors[0],
		},
	}, nil
}

// GetSeriesData generates and returns sine wave data.
func (p *Plugin) GetSeriesData(seriesID string) ([]float64, error) {
	numPoints := 361
	result := make([]float64, numPoints*2)

	for i := 0; i < numPoints; i++ {
		x := float64(i)
		y := math.Sin(float64(i) * math.Pi / 180.0)
		result[i*2] = x
		result[i*2+1] = y
	}

	return result, nil
}

// Close cleans up plugin resources.
func (p *Plugin) Close() error {
	return nil
}
