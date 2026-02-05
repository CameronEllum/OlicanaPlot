// Package synthetic provides a synthetic data generator plugin for OlicanaPlot.
package synthetic

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

const pluginName = "Synthetic Data Generator"

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
	frequency       float64
}

type ConfigResult struct {
	SimulationType  string
	NumPoints       int
	NumSeries       int
	Noise           float64
	CorrelationTime float64
	Amplitude       float64
	Frequency       float64
	Cancelled       bool
}

type SyntheticDialog struct {
	window *application.WebviewWindow
	result chan ConfigResult
}

func NewSyntheticDialog(app *application.App) *SyntheticDialog {
	dialog := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:       "Synthetic Data Configuration",
		Width:       460,
		Height:      700,
		AlwaysOnTop: true,
		Frameless:   true,
		Hidden:      true,
		URL:         "/synthetic-dialog.html",
	})

	d := &SyntheticDialog{
		window: dialog,
		result: make(chan ConfigResult),
	}

	app.Event.On("synthetic-config-submit", func(e *application.CustomEvent) {
		if data, ok := e.Data.(map[string]interface{}); ok {
			result := ConfigResult{
				SimulationType: data["simulationType"].(string),
				NumPoints:      int(data["numPoints"].(float64)),
				NumSeries:      int(data["numSeries"].(float64)),
			}
			// Parse optional parameters with defaults
			if noise, ok := data["noise"].(float64); ok {
				result.Noise = noise
			} else {
				result.Noise = 1.0
			}
			if correlationTime, ok := data["correlationTime"].(float64); ok {
				result.CorrelationTime = correlationTime
			} else {
				result.CorrelationTime = 10.0
			}
			if amplitude, ok := data["amplitude"].(float64); ok {
				result.Amplitude = amplitude
			} else {
				result.Amplitude = 1.0
			}
			if frequency, ok := data["frequency"].(float64); ok {
				result.Frequency = frequency
			} else {
				result.Frequency = 0.1
			}
			d.submit(result)
		}
	})

	app.Event.On("synthetic-config-cancel", func(e *application.CustomEvent) {
		d.cancel()
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
		amplitude:       1.0,
		frequency:       0.1,
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
	p.frequency = result.Frequency
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
	frequency := p.frequency
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
		case "Sinusoidal":
			whiteNoise := rng.NormFloat64() * 0.1
			phase := float64(seriesIdx) * 0.5
			y = amplitude*math.Sin(2*math.Pi*frequency*t+phase) + whiteNoise
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
