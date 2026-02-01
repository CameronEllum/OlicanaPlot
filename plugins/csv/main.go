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
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

const (
	pluginName    = "CSV IPC"
	pluginVersion = 1
)

// Standard Plotly colors
var ChartColors = []string{
	"#636EFA", "#EF553B", "#00CC96", "#AB63FA", "#FFA15A",
	"#19D3F3", "#FF6692", "#B6E880", "#FF97FF", "#FECB52",
}

// Plugin state
var (
	currentFile string
	headers     []string
	data        map[string][]float64
	selectedX   string
	selectedY   []string
)

// Request represents an IPC request from the host.
type Request struct {
	Method   string `json:"method"`
	Args     string `json:"args,omitempty"`
	SeriesID string `json:"series_id,omitempty"`
}

// Response represents an IPC response.
type Response struct {
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
	Type    string      `json:"type,omitempty"`
	Length  int         `json:"length,omitempty"`
	Name    string      `json:"name,omitempty"`
	Version uint32      `json:"version,omitempty"`
}

// ChartConfig holds chart display configuration.
type ChartConfig struct {
	Title      string   `json:"title"`
	AxisLabels []string `json:"axis_labels"`
}

// SeriesConfig describes a data series.
type SeriesConfig struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

func main() {
	// Check for --metadata flag
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
			os.Exit(0)
		}
	}

	data = make(map[string][]float64)
	handleIPC()
}

func log(level, message string) {
	msg := map[string]string{
		"method":  "log",
		"level":   level,
		"message": message,
	}
	bytes, _ := json.Marshal(msg)
	os.Stdout.Write(bytes)
	os.Stdout.Write([]byte("\n"))
	os.Stdout.Sync()
}

func sendResponse(resp Response) {
	respJSON, _ := json.Marshal(resp)
	os.Stdout.Write(respJSON)
	os.Stdout.Write([]byte("\n"))
	os.Stdout.Sync()
}

func sendError(msg string) {
	sendResponse(Response{Error: msg})
}

func sendBinaryData(floats []float64) {
	binaryData := floatsToBytes(floats)
	header := Response{
		Type:   "binary",
		Length: len(binaryData),
	}
	headerJSON, _ := json.Marshal(header)
	os.Stdout.Write(headerJSON)
	os.Stdout.Write([]byte("\n"))
	os.Stdout.Sync()

	os.Stdout.Write(binaryData)
	os.Stdout.Sync()
}

func floatsToBytes(data []float64) []byte {
	result := make([]byte, len(data)*8)
	for i, f := range data {
		binary.LittleEndian.PutUint64(result[i*8:], math.Float64bits(f))
	}
	return result
}

func handleIPC() {
	log("info", "CSV IPC Plugin started")
	scanner := bufio.NewScanner(os.Stdin)
	// Increase buffer for large JSON messages
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			sendError(fmt.Sprintf("Invalid JSON: %v", err))
			continue
		}

		switch req.Method {
		case "info":
			sendResponse(Response{
				Name:    pluginName,
				Version: pluginVersion,
			})

		case "initialize":
			if err := handleInitialize(req.Args, scanner); err != nil {
				sendError(err.Error())
			} else {
				sendResponse(Response{Result: map[string]interface{}{}})
			}

		case "get_chart_config":
			title := "CSV Plot"
			if currentFile != "" {
				title = fmt.Sprintf("CSV: %s", currentFile)
			}
			xLabel := "Index"
			if selectedX != "" {
				xLabel = selectedX
			}
			sendResponse(Response{
				Result: ChartConfig{
					Title:      title,
					AxisLabels: []string{xLabel, "Value"},
				},
			})

		case "get_series_config":
			series := make([]SeriesConfig, len(selectedY))
			for i, yCol := range selectedY {
				series[i] = SeriesConfig{
					ID:    yCol,
					Name:  yCol,
					Color: ChartColors[i%len(ChartColors)],
				}
			}
			sendResponse(Response{Result: series})

		case "get_series_data":
			handleGetSeriesData(req.SeriesID)

		default:
			sendError(fmt.Sprintf("Unknown method: %s", req.Method))
		}
	}

	if err := scanner.Err(); err != nil {
		log("error", fmt.Sprintf("Scanner error: %v", err))
	}
}

func handleInitialize(initStr string, scanner *bufio.Scanner) error {
	var filePath string

	if initStr != "" {
		// File path provided via drag-drop or recent files
		filePath = initStr
		log("info", fmt.Sprintf("Using provided file path: %s", filePath))
	} else {
		// Request file selection from host using show_form
		fileSchema := map[string]interface{}{
			"method": "show_form",
			"title":  "Select CSV File",
			"schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"filePath": map[string]interface{}{
						"type":  "string",
						"title": "CSV File Path",
					},
				},
			},
			"uiSchema": map[string]interface{}{
				"filePath": map[string]interface{}{
					"ui:widget": "file",
					"ui:options": map[string]interface{}{
						"accept": ".csv",
					},
				},
			},
		}

		schemaJSON, _ := json.Marshal(fileSchema)
		os.Stdout.Write(schemaJSON)
		os.Stdout.Write([]byte("\n"))
		os.Stdout.Sync()

		// Wait for response from host
		if !scanner.Scan() {
			return fmt.Errorf("failed to read file selection response")
		}
		response := scanner.Text()

		if strings.Contains(response, `"error"`) {
			return fmt.Errorf("file selection cancelled")
		}

		// Parse response
		var resp map[string]interface{}
		if err := json.Unmarshal([]byte(response), &resp); err != nil {
			return fmt.Errorf("failed to parse file selection response: %v", err)
		}

		if result, ok := resp["result"].(map[string]interface{}); ok {
			if fp, ok := result["filePath"].(string); ok {
				filePath = fp
			}
		}

		if filePath == "" {
			return fmt.Errorf("no file selected")
		}
	}

	// Load the CSV file
	if err := loadCSVFile(filePath); err != nil {
		return fmt.Errorf("failed to load CSV: %v", err)
	}

	log("info", fmt.Sprintf("Loaded CSV with %d columns", len(headers)))

	// Build column selection form schema
	columnOptions := make([]map[string]interface{}, 0, len(headers)+1)
	columnOptions = append(columnOptions, map[string]interface{}{
		"const": "",
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

	columnSchema := map[string]interface{}{
		"method": "show_form",
		"title":  "Select Columns",
		"schema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"xColumn": map[string]interface{}{
					"type":    "string",
					"title":   "X-Axis Column",
					"oneOf":   columnOptions,
					"default": "",
				},
				"yColumns": map[string]interface{}{
					"type":  "array",
					"title": "Y-Axis Columns",
					"items": map[string]interface{}{
						"type":  "string",
						"oneOf": yColumnItems,
					},
					"uniqueItems": true,
					"minItems":    1,
				},
			},
		},
		"uiSchema": map[string]interface{}{
			"xColumn": map[string]interface{}{
				"ui:widget": "select",
			},
			"yColumns": map[string]interface{}{
				"ui:widget": "checkboxes",
			},
		},
	}

	schemaJSON, _ := json.Marshal(columnSchema)
	os.Stdout.Write(schemaJSON)
	os.Stdout.Write([]byte("\n"))
	os.Stdout.Sync()

	// Wait for column selection response
	if !scanner.Scan() {
		return fmt.Errorf("failed to read column selection response")
	}
	response := scanner.Text()

	if strings.Contains(response, `"error"`) {
		return fmt.Errorf("column selection cancelled")
	}

	// Parse column selection
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(response), &resp); err != nil {
		return fmt.Errorf("failed to parse column selection response: %v", err)
	}

	if result, ok := resp["result"].(map[string]interface{}); ok {
		if xCol, ok := result["xColumn"].(string); ok {
			selectedX = xCol
		}
		if yCols, ok := result["yColumns"].([]interface{}); ok {
			selectedY = make([]string, 0, len(yCols))
			for _, col := range yCols {
				if colStr, ok := col.(string); ok {
					selectedY = append(selectedY, colStr)
				}
			}
		}
	}

	if len(selectedY) == 0 {
		return fmt.Errorf("no Y columns selected")
	}

	log("info", fmt.Sprintf("Selected X=%s, Y=%v", selectedX, selectedY))
	return nil
}

func loadCSVFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) == 0 {
		return fmt.Errorf("empty CSV file")
	}

	// First row is headers
	headers = records[0]
	currentFile = path

	// Initialize data map
	data = make(map[string][]float64)
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

	return nil
}

func handleGetSeriesData(seriesID string) {
	yData, ok := data[seriesID]
	if !ok {
		sendError(fmt.Sprintf("series not found: %s", seriesID))
		return
	}

	count := len(yData)
	result := make([]float64, 0, count*2)

	if selectedX == "" || selectedX == "Index" {
		// Use row index as X
		for i, y := range yData {
			result = append(result, float64(i), y)
		}
	} else if xData, ok := data[selectedX]; ok {
		// Use selected column as X
		minLen := count
		if len(xData) < minLen {
			minLen = len(xData)
		}
		for i := 0; i < minLen; i++ {
			result = append(result, xData[i], yData[i])
		}
	} else {
		// Fallback to index
		for i, y := range yData {
			result = append(result, float64(i), y)
		}
	}

	sendBinaryData(result)
}
