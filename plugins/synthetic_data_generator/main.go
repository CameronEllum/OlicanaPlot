// Synthetic IPC Plugin - A standalone Wails app that demonstrates a GUI-enabled IPC plugin.
// This plugin generates synthetic data and presents a configuration dialog via Wails3.
//
// Protocol:
//   - Reads JSON requests from stdin (one per line)
//   - Writes JSON responses to stdout (one per line)
//   - For binary data, writes a JSON header followed by raw bytes
//   - On "initialize" method, opens the main window as a configuration dialog
package main

import (
	"bufio"
	"embed"
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	sdk "olicanaplot/sdk/go"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
//
//go:embed all:frontend/dist
var assets embed.FS

const (
	pluginName    = "Synthetic Data Generator (IPC)"
	pluginVersion = 1
)

// ConfigResult holds the result from the configuration UI.
type ConfigResult struct {
	SimulationType  string  `json:"simulationType"`
	NumPoints       int     `json:"numPoints"`
	NumSeries       int     `json:"numSeries"`
	Noise           float64 `json:"noise"`
	CorrelationTime float64 `json:"correlationTime"`
	Amplitude       float64 `json:"amplitude"`
	Frequency       float64 `json:"frequency"`
	Cancelled       bool    `json:"cancelled"`
}

// SyntheticService provides methods callable from the Svelte frontend.
type SyntheticService struct {
	resultChan chan ConfigResult
}

func NewSyntheticService() *SyntheticService {
	return &SyntheticService{
		resultChan: make(chan ConfigResult, 1),
	}
}

func (s *SyntheticService) Submit(simulationType string, numPoints int, numSeries int, noise float64, correlationTime float64, amplitude float64, frequency float64) {
	s.resultChan <- ConfigResult{
		SimulationType:  simulationType,
		NumPoints:       numPoints,
		NumSeries:       numSeries,
		Noise:           noise,
		CorrelationTime: correlationTime,
		Amplitude:       amplitude,
		Frequency:       frequency,
	}
}

func (s *SyntheticService) Cancel() {
	s.resultChan <- ConfigResult{Cancelled: true}
}

// pluginState stores the current generation parameters.
type pluginState struct {
	simulationType  string
	numPoints       int
	numSeries       int
	seed            uint64
	noise           float64
	correlationTime float64
	amplitude       float64
	frequency       float64
}

var (
	state      *pluginState
	app        *application.App
	service    *SyntheticService
	mainWindow application.Window
)

func main() {
	// Check for --metadata flag
	for _, arg := range os.Args {
		if arg == "--metadata" {
			metadata := map[string]interface{}{
				"name":     pluginName,
				"patterns": []interface{}{},
			}
			jsonBytes, _ := json.Marshal(metadata)
			fmt.Println(string(jsonBytes))
			os.Exit(0)
		}
	}

	ipcLog("info", "Starting Synthetic IPC Plugin")

	state = &pluginState{
		simulationType:  "Random Walk",
		numPoints:       100000,
		numSeries:       3,
		noise:           1.0,
		correlationTime: 10.0,
		amplitude:       1.0,
		frequency:       0.1,
		seed:            uint64(time.Now().UnixNano()),
	}
	service = NewSyntheticService()

	app = application.New(application.Options{
		Name:        "Synthetic IPC",
		Description: "Synthetic data generator plugin",
		Services: []application.Service{
			application.NewService(service),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	// Create the main window immediately
	mainWindow = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:       "Synthetic Data Configuration",
		Width:       460,
		Height:      700,
		AlwaysOnTop: true,
		URL:         "/",
		Frameless:   true,
	})
	mainWindow.Center()

	ipcDone := make(chan struct{})
	go func() {
		handleIPC()
		close(ipcDone)
		app.Quit()
	}()

	if err := app.Run(); err != nil {
		ipcLog("error", fmt.Sprintf("Wails Run error: %v", err))
	}

	<-ipcDone
	os.Exit(0)
}

func ipcLog(level, message string) {
	sdk.Log(level, message)
}

func handleIPC() {
	ipcLog("info", "IPC handler started")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		var req sdk.Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			sdk.SendError(fmt.Sprintf("Invalid JSON: %v", err))
			continue
		}

		switch req.Method {
		case "info":
			sdk.SendResponse(sdk.Response{
				Name:    pluginName,
				Version: pluginVersion,
			})

		case "initialize":
			ipcLog("info", "Waiting for user configuration")
			// Clear any stale results
			select {
			case <-service.resultChan:
			default:
			}

			// Block until user submits or cancels via the UI
			result := <-service.resultChan
			if result.Cancelled {
				ipcLog("info", "Configuration cancelled - quitting")
				sdk.SendError("configuration cancelled")
				if mainWindow != nil {
					mainWindow.Close()
				}
				return // Returning will trigger app.Quit() in the goroutine
			}

			// Update state with configuration
			state.simulationType = result.SimulationType
			state.numPoints = result.NumPoints
			state.numSeries = result.NumSeries
			state.noise = result.Noise
			state.correlationTime = result.CorrelationTime
			state.amplitude = result.Amplitude
			state.frequency = result.Frequency
			state.seed = uint64(time.Now().UnixNano())

			// Close the config window after submission
			if mainWindow != nil {
				mainWindow.Close()
			}

			sdk.SendResponse(sdk.Response{Result: map[string]interface{}{}})

		case "get_chart_config":
			config := sdk.ChartConfig{
				Title:      "Synthetic Data (IPC)",
				AxisLabels: []string{"Time (s)", state.simulationType},
			}
			sdk.SendResponse(sdk.Response{Result: config})

		case "get_series_config":
			series := make([]sdk.SeriesConfig, state.numSeries)
			for i := 0; i < state.numSeries; i++ {
				series[i] = sdk.SeriesConfig{
					ID:   fmt.Sprintf("synthetic_%d", i),
					Name: fmt.Sprintf("IPC Series %d", i+1),
				}
			}
			sdk.SendResponse(sdk.Response{Result: series})

		case "get_series_data":
			data, storage := generateData(state, req.SeriesID, req.PreferredStorage)
			sdk.SendBinaryData(data, storage)

		default:
			sdk.SendError(fmt.Sprintf("Unknown method: %s", req.Method))
		}
	}

	if err := scanner.Err(); err != nil {
		ipcLog("error", fmt.Sprintf("Scanner error: %v", err))
	}
}

func generateData(st *pluginState, seriesID string, preferredStorage string) ([]float64, string) {
	simType := st.simulationType
	numPoints := st.numPoints
	seed := st.seed
	noise := st.noise
	correlationTime := st.correlationTime
	amplitude := st.amplitude
	frequency := st.frequency

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
			theta := 1.0 / correlationTime
			if correlationTime <= 0 {
				theta = 0.1
			}
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
	return result, storage
}
