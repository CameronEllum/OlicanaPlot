// CSV IPC Plugin - A standalone CSV file loader plugin using host-controlled UI.
//
// Protocol:
//   - Reads JSON requests from stdin (one per line)
//   - Writes JSON responses to stdout (one per line)
//   - Uses show_form for host-controlled column selection UI
//   - For binary data, writes a JSON header followed by raw bytes

package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	sdk "olicanaplot/sdk/go"
)

const (
	pluginName    = "CSV IPC"
	pluginVersion = 1
)

// Plugin state
var (
	currentFile string
	headers     []string
	data        map[string][]float64
	selectedX   string
	selectedY   []string
)

func main() {
	// Check for --metadata flag (Discovery Protocol)
	if handleMetadata() {
		return
	}

	data = make(map[string][]float64)
	processIPC()
}

// handleMetadata checks for the --metadata flag and exits if found.
func handleMetadata() bool {
	for _, arg := range os.Args[1:] {
		if arg == "--metadata" {
			metadata := map[string]interface{}{
				"name": pluginName,
				"patterns": []map[string]interface{}{
					{
						"description": "CSV Files",
						"patterns":    []string{"*.csv"},
					},
				},
			}
			jsonBytes, _ := json.Marshal(metadata)
			fmt.Println(string(jsonBytes))
			return true
		}
	}
	return false
}

// processIPC runs the main communication loop reading from stdin.
func processIPC() {
	sdk.Log("info", "CSV IPC Plugin started")
	scanner := bufio.NewScanner(os.Stdin)
	// Increase buffer for large JSON messages
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req sdk.Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			sdk.SendError(fmt.Sprintf("Invalid JSON: %v", err))
			continue
		}

		handleMethod(req, scanner)
	}

	if err := scanner.Err(); err != nil {
		sdk.Log("error", fmt.Sprintf("Scanner error: %v", err))
	}
}

// handleMethod dispatches incoming IPC calls to specific handlers.
func handleMethod(req sdk.Request, scanner *bufio.Scanner) {
	switch req.Method {
	case "info":
		sdk.SendResponse(sdk.Response{
			Name:    pluginName,
			Version: pluginVersion,
		})

	case "initialize":
		if err := handleInitialize(req.Args, scanner); err != nil {
			sdk.SendError(err.Error())
		} else {
			sdk.SendResponse(sdk.Response{Result: map[string]interface{}{}})
		}

	case "get_chart_config":
		sdk.SendResponse(sdk.Response{
			Result: getChartConfig(),
		})

	case "get_series_config":
		sdk.SendResponse(sdk.Response{
			Result: getSeriesConfig(),
		})

	case "get_series_data":
		handleGetSeriesData(req.SeriesID, req.PreferredStorage)

	default:
		sdk.SendError(fmt.Sprintf("Unknown method: %s", req.Method))
	}
}

// handleInitialize manages the multi-step initialization process (file selection -> column selection).
func handleInitialize(initStr string, scanner *bufio.Scanner) error {
	filePath, err := resolveFilePath(initStr, scanner)
	if err != nil {
		return err
	}

	// Read ONLY headers initially (Lazy Loading)
	h, err := readCSVHeaders(filePath)
	if err != nil {
		return fmt.Errorf("failed to read headers: %w", err)
	}
	headers = h
	currentFile = filePath

	// Show column selection UI
	result, err := showColumnSelection(scanner)
	if err != nil {
		return err
	}

	// Apply selection
	selectedX = result.XColumn
	selectedY = result.YColumns

	// Load the actual data ONLY after user confirms
	sdk.Log("info", fmt.Sprintf("Loading CSV data from %s...", filePath))
	d, err := loadCSVData(filePath, headers)
	if err != nil {
		return fmt.Errorf("failed to load data: %w", err)
	}
	data = d

	sdk.Log("info", fmt.Sprintf("CSV loaded: %d columns, X=%s, Y=%v", len(headers), selectedX, selectedY))
	return nil
}

// resolveFilePath either uses the provided path or requests one from the host via show_form.
func resolveFilePath(initStr string, scanner *bufio.Scanner) (string, error) {
	if initStr != "" {
		sdk.Log("info", fmt.Sprintf("Using provided file path: %s", initStr))
		return initStr, nil
	}

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"filePath": map[string]interface{}{
				"type":  "string",
				"title": "CSV File Path",
			},
		},
	}
	uiSchema := map[string]interface{}{
		"filePath": map[string]interface{}{
			"ui:widget": "file",
			"ui:options": map[string]interface{}{
				"accept": ".csv",
			},
		},
	}

	sdk.SendShowForm("Select CSV File", schema, uiSchema, nil)

	if !scanner.Scan() {
		return "", fmt.Errorf("failed to read file selection response")
	}

	var resp struct {
		Result struct {
			FilePath string `json:"filePath"`
		} `json:"result"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return "", fmt.Errorf("failed to parse file selection response: %v", err)
	}
	if resp.Error != "" {
		return "", fmt.Errorf("file selection cancelled: %s", resp.Error)
	}

	return resp.Result.FilePath, nil
}

type ColumnSelectionResult struct {
	XColumn  string   `json:"xColumn"`
	YColumns []string `json:"yColumns"`
}

// showColumnSelection requests and parses the user's column choices.
func showColumnSelection(scanner *bufio.Scanner) (*ColumnSelectionResult, error) {
	// Build column selection options
	columnOptions := make([]map[string]interface{}, 0, len(headers)+1)
	columnOptions = append(columnOptions, map[string]interface{}{
		"const": "Index",
		"title": "Index (row number)",
	})
	for _, h := range headers {
		columnOptions = append(columnOptions, map[string]interface{}{
			"const": h,
			"title": h,
		})
	}

	yColumnItems := make([]map[string]interface{}, 0, len(headers))
	for _, h := range headers {
		yColumnItems = append(yColumnItems, map[string]interface{}{
			"const": h,
			"title": h,
		})
	}

	// Defaults based on heuristics
	defaultX := "Index"
	defaultY := headers
	if len(headers) > 1 {
		defaultX = headers[0]
		defaultY = headers[1:]
	} else if len(headers) == 1 {
		defaultX = "Index"
		defaultY = headers
	}

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"xColumn": map[string]interface{}{
				"type":    "string",
				"title":   "X-Axis Column",
				"oneOf":   columnOptions,
				"default": defaultX,
			},
			"yColumns": map[string]interface{}{
				"type":    "array",
				"title":   "Y-Axis Columns",
				"default": defaultY,
				"items": map[string]interface{}{
					"type":  "string",
					"oneOf": yColumnItems,
				},
				"uniqueItems": true,
				"minItems":    1,
			},
		},
	}
	uiSchema := map[string]interface{}{
		"xColumn":  map[string]interface{}{"ui:widget": "select"},
		"yColumns": map[string]interface{}{"ui:widget": "checkboxes"},
	}

	sdk.SendShowForm("Select Columns", schema, uiSchema, map[string]interface{}{
		"xColumn":  defaultX,
		"yColumns": defaultY,
	})

	if !scanner.Scan() {
		return nil, fmt.Errorf("failed to read column selection response")
	}

	var resp struct {
		Result ColumnSelectionResult `json:"result"`
		Error  string                `json:"error"`
	}
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse column selection response: %v", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("column selection cancelled")
	}

	return &resp.Result, nil
}

func getChartConfig() sdk.ChartConfig {
	title := "CSV Plot"
	if currentFile != "" {
		title = fmt.Sprintf("CSV: %s", currentFile)
	}
	xLabel := "Index"
	if selectedX != "" {
		xLabel = selectedX
	}
	return sdk.ChartConfig{
		Title:      title,
		AxisLabels: []string{xLabel, "Value"},
	}
}

func getSeriesConfig() []sdk.SeriesConfig {
	series := make([]sdk.SeriesConfig, len(selectedY))
	for i, yCol := range selectedY {
		series[i] = sdk.SeriesConfig{
			ID:   yCol,
			Name: yCol,
		}
	}
	return series
}

// readCSVHeaders reads only the first line of the file to extract column names.
func readCSVHeaders(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}

	for i, h := range headers {
		headers[i] = strings.TrimSpace(h)
	}
	return headers, nil
}

// loadCSVData loads the entire file content into memory.
func loadCSVData(path string, headers []string) (map[string][]float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return make(map[string][]float64), nil
	}

	// Initialize data map
	resultMap := make(map[string][]float64)
	for _, h := range headers {
		resultMap[h] = make([]float64, 0, len(records)-1)
	}

	// Parse data rows
	for rowIdx := 1; rowIdx < len(records); rowIdx++ {
		row := records[rowIdx]
		for colIdx, val := range row {
			if colIdx < len(headers) {
				header := headers[colIdx]
				parsed, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
				if err != nil {
					parsed = math.NaN()
				}
				resultMap[header] = append(resultMap[header], parsed)
			}
		}
	}

	return resultMap, nil
}

// handleGetSeriesData retrieves and sends binary data for a specific series.
func handleGetSeriesData(seriesID string, preferredStorage string) {
	yData, ok := data[seriesID]
	if !ok {
		sdk.SendError(fmt.Sprintf("series not found: %s", seriesID))
		return
	}

	count := len(yData)
	result := make([]float64, count*2)
	isArrays := preferredStorage == "arrays"
	storage := "interleaved"
	if isArrays {
		storage = "arrays"
	}

	xSrc, hasX := data[selectedX]
	if selectedX == "" || selectedX == "Index" {
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

	sdk.SendBinaryData(result, storage)
}
