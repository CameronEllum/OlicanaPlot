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

	"olicanaplot/pkg/ipcplugin"
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
	// Check for --metadata flag
	for _, arg := range os.Args[1:] {
		if arg == "--metadata" {
			metadata := map[string]interface{}{
				"name": pluginName,
				"patterns": []map[string]interface{}{
					{
						"description": "CSV Files (IPC plug-in)",
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

func handleIPC() {
	ipcplugin.Log("info", "CSV IPC Plugin started")
	scanner := bufio.NewScanner(os.Stdin)
	// Increase buffer for large JSON messages
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req ipcplugin.Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			ipcplugin.SendError(fmt.Sprintf("Invalid JSON: %v", err))
			continue
		}

		switch req.Method {
		case "info":
			ipcplugin.SendResponse(ipcplugin.Response{
				Name:    pluginName,
				Version: pluginVersion,
			})

		case "initialize":
			if err := handleInitialize(req.Args, scanner); err != nil {
				ipcplugin.SendError(err.Error())
			} else {
				ipcplugin.SendResponse(ipcplugin.Response{Result: map[string]interface{}{}})
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
			ipcplugin.SendResponse(ipcplugin.Response{
				Result: ipcplugin.ChartConfig{
					Title:      title,
					AxisLabels: []string{xLabel, "Value"},
				},
			})

		case "get_series_config":
			series := make([]ipcplugin.SeriesConfig, len(selectedY))
			for i, yCol := range selectedY {
				series[i] = ipcplugin.SeriesConfig{
					ID:    yCol,
					Name:  yCol,
					Color: ipcplugin.ChartColors[i%len(ipcplugin.ChartColors)],
				}
			}
			ipcplugin.SendResponse(ipcplugin.Response{Result: series})

		case "get_series_data":
			handleGetSeriesData(req.SeriesID, req.PreferredStorage)

		default:
			ipcplugin.SendError(fmt.Sprintf("Unknown method: %s", req.Method))
		}
	}

	if err := scanner.Err(); err != nil {
		ipcplugin.Log("error", fmt.Sprintf("Scanner error: %v", err))
	}
}

func handleInitialize(initStr string, scanner *bufio.Scanner) error {
	var filePath string

	if initStr != "" {
		// File path provided via drag-drop or recent files
		filePath = initStr
		ipcplugin.Log("info", fmt.Sprintf("Using provided file path: %s", filePath))
	} else {
		// Request file selection from host using show_form
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

		ipcplugin.SendShowForm("Select CSV File", schema, uiSchema)

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

	ipcplugin.Log("info", fmt.Sprintf("Loaded CSV with %d columns", len(headers)))

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

	schema := map[string]interface{}{
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
	}
	uiSchema := map[string]interface{}{
		"xColumn": map[string]interface{}{
			"ui:widget": "select",
		},
		"yColumns": map[string]interface{}{
			"ui:widget": "checkboxes",
		},
	}

	ipcplugin.SendShowForm("Select Columns", schema, uiSchema)

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

	ipcplugin.Log("info", fmt.Sprintf("Selected X=%s, Y=%v", selectedX, selectedY))
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

func handleGetSeriesData(seriesID string, preferredStorage string) {
	yData, ok := data[seriesID]
	if !ok {
		ipcplugin.SendError(fmt.Sprintf("series not found: %s", seriesID))
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

	ipcplugin.SendBinaryData(result, storage)
}
