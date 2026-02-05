// Package csv provides a CSV file loading plugin for OlicanaPlot.
package csv

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"olicanaplot/internal/logging"
	"olicanaplot/internal/plugins"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const pluginName = "CSV Connector"

// Plugin implements the CSV file loading plugin.
type Plugin struct {
	mu          sync.Mutex
	currentFile string
	headers     []string
	data        map[string][]float64
	selectedY   []string
	selectedX   string // Empty means use index
}

// New creates a new CSV plugin.
func New() *Plugin {
	return &Plugin{
		data: make(map[string][]float64),
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
	return []plugins.FilePattern{
		{
			Description: "CSV Files",
			Patterns:    []string{"*.csv"},
		},
	}
}

// Initialize sets up the plugin by opening a file dialog and then creating a configuration window.
func (p *Plugin) Initialize(ctx interface{}, initStr string, logger logging.Logger) (string, error) {
	// Cast context to Application
	app, ok := ctx.(*application.App)
	if !ok || app == nil {
		logger.Error("Invalid application context")
		return "{}", fmt.Errorf("invalid application context")
	}

	var selectedFile string
	var err error

	if initStr != "" {
		// Use provided path
		selectedFile = initStr
		logger.Info("Using provided file path", "path", selectedFile)
	} else {
		// Open file dialog using Wails v3 API
		selectedFile, err = app.Dialog.OpenFile().
			SetTitle("Select CSV File").
			AddFilter("CSV Files", "*.csv").
			AddFilter("All Files", "*.*").
			PromptForSingleSelection()

		if err != nil || selectedFile == "" {
			logger.Debug("File dialog cancelled or no file selected")
			return "{}", nil
		}
		logger.Info("File selected", "path", selectedFile)
	}

	// Load the file to get headers
	headers, err := p.LoadFile(selectedFile)
	if err != nil {
		logger.Error("Failed to load CSV file", "path", selectedFile, "error", err)
		return "{}", fmt.Errorf("failed to load CSV file: %w", err)
	}
	logger.Info("CSV file loaded", "path", selectedFile, "columns", len(headers))

	// Create and show dialog
	dialog := NewCsvDialog(app, selectedFile, headers)

	// Register event listeners
	unsubSubmit := app.Event.On("csv-config-submit", func(event *application.CustomEvent) {
		if event.Data != nil {
			if configMap, ok := event.Data.(map[string]interface{}); ok {
				xColumn, _ := configMap["xColumn"].(string)
				yColumnsRaw, _ := configMap["yColumns"].([]interface{})
				yColumns := make([]string, 0, len(yColumnsRaw))
				for _, col := range yColumnsRaw {
					if colStr, ok := col.(string); ok {
						yColumns = append(yColumns, colStr)
					}
				}
				dialog.Submit(xColumn, yColumns)
			}
		}
	})

	unsubCancel := app.Event.On("csv-config-cancel", func(event *application.CustomEvent) {
		dialog.Cancel()
	})

	defer unsubSubmit()
	defer unsubCancel()

	result := dialog.Show()

	if result.Ok {
		p.SetSelection(result.YColumns, result.XColumn)
		logger.Info("CSV configuration complete", "xColumn", result.XColumn, "yColumns", result.YColumns)
	}

	return "{}", nil
}

// encodeHeaders helper (re-added since I inline logic previously)
func encodeHeaders(headers []string) string {
	data, _ := json.Marshal(headers)
	return string(data)
}

// LoadFile loads a CSV file from the given path and returns headers.
// This method can be called from the frontend via the plugin service.
func (p *Plugin) LoadFile(path string) ([]string, error) {
	return p.loadCSVFile(path)
}

// SetSelection configures which columns to use as X and Y series.
func (p *Plugin) SetSelection(yColumns []string, xColumn string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.selectedY = yColumns
	p.selectedX = xColumn
	return nil
}

// loadCSVFile loads a CSV file from a path
func (p *Plugin) loadCSVFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	return p.processCSV(reader, path)
}

// loadCSVContent loads CSV data from a string
func (p *Plugin) loadCSVContent(content string, name string) ([]string, error) {
	reader := csv.NewReader(strings.NewReader(content))
	return p.processCSV(reader, name)
}

// processCSV reads CSV data from a reader and updates the plugin state
func (p *Plugin) processCSV(reader *csv.Reader, name string) ([]string, error) {
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("empty CSV file")
	}

	// First row is headers
	headers := records[0]

	// Initialize data map
	data := make(map[string][]float64)
	for _, h := range headers {
		data[h] = make([]float64, 0, len(records)-1)
	}

	// Parse data rows
	for rowIdx := 1; rowIdx < len(records); rowIdx++ {
		row := records[rowIdx]
		for colIdx, val := range row {
			if colIdx < len(headers) {
				header := headers[colIdx]
				parsed, err := strconv.ParseFloat(val, 64)
				if err != nil {
					parsed = math.NaN()
				}
				data[header] = append(data[header], parsed)
			}
		}
	}

	p.mu.Lock()
	p.currentFile = name
	p.headers = headers
	p.data = data
	p.selectedY = nil
	p.selectedX = ""
	p.mu.Unlock()

	return headers, nil
}

// GetChartConfig returns chart display configuration.
func (p *Plugin) GetChartConfig(args string) (*plugins.ChartConfig, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	title := "CSV Plot"
	if p.currentFile != "" {
		title = fmt.Sprintf("CSV: %s", p.currentFile)
	}

	xLabel := "Index"
	if p.selectedX != "" {
		xLabel = p.selectedX
	}

	return &plugins.ChartConfig{
		Title:      title,
		AxisLabels: []string{xLabel, "Value"},
	}, nil
}

// GetSeriesConfig returns the list of available series.
func (p *Plugin) GetSeriesConfig() ([]plugins.SeriesConfig, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	series := make([]plugins.SeriesConfig, len(p.selectedY))
	for i, yCol := range p.selectedY {
		series[i] = plugins.SeriesConfig{
			ID:    yCol,
			Name:  yCol,
			Color: plugins.ChartColors[i%len(plugins.ChartColors)],
		}
	}
	return series, nil
}

// GetSeriesData returns binary float64 data for the specified series ID.
func (p *Plugin) GetSeriesData(seriesID string, preferredStorage string) ([]float64, string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	yData, ok := p.data[seriesID]
	if !ok {
		return nil, "", fmt.Errorf("series not found: %s", seriesID)
	}

	count := len(yData)
	result := make([]float64, count*2)
	isArrays := preferredStorage == "arrays"
	storage := "interleaved"
	if isArrays {
		storage = "arrays"
	}

	xSrc, hasX := p.data[p.selectedX]
	if p.selectedX == "" || p.selectedX == "Index" {
		hasX = false
	}

	for i := 0; i < count; i++ {
		var x float64
		if hasX && i < len(xSrc) {
			x = xSrc[i]
		} else {
			x = float64(i)
		}

		if isArrays {
			result[i] = x
			result[count+i] = yData[i]
		} else {
			result[i*2] = x
			result[i*2+1] = yData[i]
		}
	}

	return result, storage, nil
}

// Close cleans up plugin resources.
func (p *Plugin) Close() error {
	return nil
}
