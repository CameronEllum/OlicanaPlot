// Package process_model_generator provides a process model data generator plugin for OlicanaPlot.
package process_model_generator

import (
	"fmt"
	"math"
	"math/rand"
	"olicanaplot/internal/logging"
	"olicanaplot/internal/plugins"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const pluginName = "Process Model Generator"

// Plugin implements the synthetic data generator plugin.
type Plugin struct {
	mu              sync.Mutex
	simulationType  string
	numPoints       int
	numSeries       int
	seed            uint64
	noise           float64
	correlationTime float64
	amplitude       float64
}

type ConfigResult struct {
	SimulationType  string
	NumPoints       int
	NumSeries       int
	Noise           float64
	CorrelationTime float64
	Amplitude       float64
	Cancelled       bool
}

type SyntheticDialog struct {
	window *application.WebviewWindow
	result chan ConfigResult
}

func NewSyntheticDialog(app *application.App) *SyntheticDialog {
	d := &SyntheticDialog{
		result: make(chan ConfigResult, 1),
	}

	requestID := fmt.Sprintf("process_model-%p", d)

	schema := map[string]interface{}{
		"type":  "object",
		"title": "Synthetic Data Configuration",
		"properties": map[string]interface{}{
			"simulationType": map[string]interface{}{
				"title":   "Simulation Type",
				"type":    "string",
				"enum":    []string{"Random Walk", "Gauss-Markov", "Random Constant", "White Noise"},
				"default": "Random Walk",
			},
			"numPoints": map[string]interface{}{
				"title":     "Number of Points",
				"type":      "integer",
				"minimum":   100,
				"maximum":   1000000,
				"default":   100000,
				"ui:widget": "range",
				"ui:options": map[string]interface{}{
					"scale": "log10",
				},
			},
			"numSeries": map[string]interface{}{
				"title":     "Number of Series",
				"type":      "integer",
				"minimum":   1,
				"maximum":   20,
				"default":   3,
				"ui:widget": "range",
			},
			"noise": map[string]interface{}{
				"title":   "Noise Level / Sigma",
				"type":    "number",
				"minimum": 0,
				"maximum": 100,
				"default": 1.0,
			},
			"correlationTime": map[string]interface{}{
				"title":   "Correlation Time (for GM)",
				"type":    "number",
				"minimum": 0.1,
				"maximum": 100,
				"default": 10.0,
			},
			"amplitude": map[string]interface{}{
				"title":   "Mean / Constant Value",
				"type":    "number",
				"minimum": -100,
				"maximum": 100,
				"default": 0.0,
			},
		},
	}

	app.Event.On(fmt.Sprintf("ipc-form-result-%s", requestID), func(e *application.CustomEvent) {
		if e.Data != nil {
			if e.Data == "error:cancelled" {
				d.cancel()
				return
			}
			if data, ok := e.Data.(map[string]interface{}); ok {
				result := ConfigResult{
					SimulationType:  data["simulationType"].(string),
					NumPoints:       int(data["numPoints"].(float64)),
					NumSeries:       int(data["numSeries"].(float64)),
					Noise:           data["noise"].(float64),
					CorrelationTime: data["correlationTime"].(float64),
					Amplitude:       data["amplitude"].(float64),
				}
				d.submit(result)
			}
		}
	})

	app.Event.On(fmt.Sprintf("ipc-form-ready-%s", requestID), func(e *application.CustomEvent) {
		app.Event.Emit(fmt.Sprintf("ipc-form-init-%s", requestID), map[string]interface{}{
			"schema": schema,
		})
	})

	// Add resize listener
	app.Event.On(fmt.Sprintf("ipc-form-resize-%s", requestID), func(e *application.CustomEvent) {
		if data, ok := e.Data.(map[string]interface{}); ok {
			width, _ := data["width"].(float64)
			height, _ := data["height"].(float64)
			if width > 0 && height > 0 {
				d.window.SetSize(int(width), int(height)+48)
			}
		}
	})

	d.window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:       "Synthetic Data Configuration",
		Width:       500,
		Height:      750,
		AlwaysOnTop: true,
		URL:         fmt.Sprintf("/dialog.html?requestID=%s", requestID),
	})

	return d
}

func (d *SyntheticDialog) Show() ConfigResult {
	d.window.Show()
	d.window.Center()
	d.window.Focus()
	return <-d.result
}

func (d *SyntheticDialog) submit(res ConfigResult) {
	d.result <- res
	d.window.Close()
}

func (d *SyntheticDialog) cancel() {
	d.result <- ConfigResult{Cancelled: true}
	d.window.Close()
}

// New creates a new synthetic data plugin with default parameters.
func New() *Plugin {
	return &Plugin{
		simulationType:  "Random Walk",
		numPoints:       100000,
		numSeries:       3,
		seed:            uint64(time.Now().UnixNano()),
		noise:           1.0,
		correlationTime: 10.0,
		amplitude:       0.0,
	}
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

// Initialize sets up the plugin by creating a custom dialog window.
func (p *Plugin) Initialize(ctx interface{}, initStr string, logger logging.Logger) (string, error) {
	logger.Debug("Initializing synthetic data generator")

	// Cast context to Application
	app, ok := ctx.(*application.App)
	if !ok || app == nil {
		logger.Warn("No application context provided")
		return "{}", nil
	}

	dialog := NewSyntheticDialog(app)
	result := dialog.Show()

	if result.Cancelled {
		logger.Debug("Configuration cancelled by user")
		return "{}", fmt.Errorf("configuration cancelled")
	}

	p.SetParameters(result)
	logger.Info("Synthetic data configured", "type", result.SimulationType, "points", result.NumPoints, "series", result.NumSeries)

	return "{}", nil
}

// GetChartConfig returns chart display configuration.
func (p *Plugin) GetChartConfig(args string) (*plugins.ChartConfig, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return &plugins.ChartConfig{
		Title:      "Synthetic Data",
		AxisLabels: []string{"Time (s)", p.simulationType},
	}, nil
}

// SetParameters updates the simulation parameters.
func (p *Plugin) SetParameters(result ConfigResult) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.simulationType = result.SimulationType
	p.numPoints = result.NumPoints
	p.numSeries = result.NumSeries
	p.noise = result.Noise
	p.correlationTime = result.CorrelationTime
	p.amplitude = result.Amplitude
	// Reset seed for new generation
	p.seed = uint64(time.Now().UnixNano())
	return nil
}

// GetSeriesConfig returns the list of available series.
func (p *Plugin) GetSeriesConfig() ([]plugins.SeriesConfig, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	series := make([]plugins.SeriesConfig, p.numSeries)
	for i := 0; i < p.numSeries; i++ {
		series[i] = plugins.SeriesConfig{
			ID:    fmt.Sprintf("synthetic_%d", i),
			Name:  fmt.Sprintf("Series %d", i+1),
			Color: plugins.ChartColors[i%len(plugins.ChartColors)],
		}
	}
	return series, nil
}

// GetSeriesData generates and returns synthetic data.
func (p *Plugin) GetSeriesData(seriesID string, preferredStorage string) ([]float64, string, error) {
	p.mu.Lock()
	simType := p.simulationType
	numPoints := p.numPoints
	seed := p.seed
	noise := p.noise
	correlationTime := p.correlationTime
	amplitude := p.amplitude
	p.mu.Unlock()

	// Parse series index from ID to create unique seed per series
	var seriesIdx int
	fmt.Sscanf(seriesID, "synthetic_%d", &seriesIdx)

	// Use standard math/rand with unique seed for each series
	rng := rand.New(rand.NewSource(int64(seed + uint64(seriesIdx)*12345)))

	result := make([]float64, (numPoints+1)*2)
	isArrays := preferredStorage == "arrays"
	storage := "interleaved"
	if isArrays {
		storage = "arrays"
	}

	var t, y float64
	// Gauss-Markov theta is inverse of correlation time
	theta := 1.0 / correlationTime

	// Initial point
	if isArrays {
		result[0] = t
		result[numPoints+1] = y
	} else {
		result[0] = t
		result[1] = y
	}

	for i := 1; i <= numPoints; i++ {
		// Time increment (uniform)
		dt := 0.1 + rng.Float64()*9.9
		t += dt

		switch simType {
		case "Random Walk":
			y += rng.NormFloat64() * math.Sqrt(dt) * noise
		case "Gauss-Markov":
			noiseVal := rng.NormFloat64() * math.Sqrt(dt) * noise
			y = y - theta*y*dt + noiseVal
		case "Random Constant":
			// amplitude is the mean, noise is the variation of the constant
			if i == 1 {
				y = amplitude + rng.NormFloat64()*noise
			}
			// stay at the same y for subsequent points
		case "White Noise":
			// amplitude is the mean, noise is the sigma
			y = amplitude + rng.NormFloat64()*noise
		default:
			y += rng.NormFloat64() * math.Sqrt(dt)
		}

		if isArrays {
			result[i] = t
			result[numPoints+1+i] = y
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
