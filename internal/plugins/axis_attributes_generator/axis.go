// Package axis_attributes_generator provides a plugin to demonstrate axis attributes.
package axis_attributes_generator

import (
	"fmt"
	"math"
	"olicanaplot/internal/logging"
	"olicanaplot/internal/plugins"
	"time"
)

const pluginName = "Axis Attributes Demo"

// Plugin implements the axis attributes generator.
type Plugin struct {
	logger logging.Logger
}

// New creates a new axis attributes plugin.
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
	logger.Debug("Axis Attributes demo plugin initialized")
	return "{}", nil
}

// GetChartConfig returns chart display configuration with rich axes.
func (p *Plugin) GetChartConfig(args string) (*plugins.ChartConfig, error) {
	return &plugins.ChartConfig{
		Title: "Axis Attributes Demonstration",
		Rows:  1,
		Cols:  2,
		Axes: []plugins.AxisGroupConfig{
			{
				Title:   "Time Representation",
				Subplot: []int{0, 0},
				XAxes: []plugins.AxisConfig{
					{
						Title: "Date",
						Type:  "date",
					},
				},
				YAxes: []plugins.AxisConfig{
					{
						Title: "Amplitude",
					},
				},
			},
			{
				Title:   "Linear vs Log",
				Subplot: []int{0, 1},
				XAxes: []plugins.AxisConfig{
					{
						Title: "Linear Scale",
					},
				},
				YAxes: []plugins.AxisConfig{
					{
						Title: "Log Scale",
						Type:  "log",
					},
				},
			},
		},
	}, nil
}

// GetSeriesConfig returns the list of available series.
func (p *Plugin) GetSeriesConfig() ([]plugins.SeriesConfig, error) {
	return []plugins.SeriesConfig{
		// Subplot (0,0) - Time Axis
		{
			ID:         "time_0",
			Name:       "Daily Variation",
			Subplot:    []int{0, 0},
			MarkerType: "circle",
		},
		// Subplot (0,1) - Log Y Axis
		{
			ID:       "log_0",
			Name:     "Exponential Growth",
			Subplot:  []int{0, 1},
			LineType: "solid",
		},
	}, nil
}

// GetSeriesData generates and returns data based on the requested ID.
func (p *Plugin) GetSeriesData(seriesID string, preferredStorage string) ([]float64, string, error) {
	points := 100
	var data []float64

	switch seriesID {
	case "time_0":
		start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < points; i++ {
			t := start.Add(time.Hour * 12 * time.Duration(i))
			x := float64(t.Unix()) + float64(t.Nanosecond())/1e9
			y := math.Sin(float64(i) * 0.1)

			if preferredStorage == "interleaved" {
				data = append(data, x, y)
			} else {
				if len(data) == 0 {
					data = make([]float64, points*2)
				}
				data[i] = x
				data[points+i] = y
			}
		}

	case "log_0":
		for i := 0; i < points; i++ {
			x := float64(i)
			y := math.Exp(x * 0.1)

			if preferredStorage == "interleaved" {
				data = append(data, x, y)
			} else {
				if len(data) == 0 {
					data = make([]float64, points*2)
				}
				data[i] = x
				data[points+i] = y
			}
		}

	default:
		return nil, "", fmt.Errorf("unknown series: %s", seriesID)
	}

	return data, preferredStorage, nil
}

// Close cleans up any resources (none for this plugin).
func (p *Plugin) Close() error {
	return nil
}
