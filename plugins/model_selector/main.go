package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	"olicanaplot/pkg/ipcplugin"
)

const (
	pluginName    = "Model Selector"
	pluginVersion = 1
)

type pluginState struct {
	modelType  string
	numSeries  int
	order      int
	multiplier int
	p, d, q    int
	noise      float64
	amplitude  float64
	frequency  float64
	seed       int64
}

var state = &pluginState{
	modelType:  "Random Walk",
	numSeries:  3,
	order:      5,
	multiplier: 1,
	noise:      1.0,
	p:          1,
	d:          0,
	q:          1,
	amplitude:  1.0,
	frequency:  0.1,
}

func main() {
	// Metadata support
	if len(os.Args) > 1 && os.Args[1] == "--metadata" {
		meta := map[string]interface{}{
			"name":     pluginName,
			"patterns": []interface{}{},
		}
		json.NewEncoder(os.Stdout).Encode(meta)
		return
	}

	state.seed = time.Now().UnixNano()
	handleIPC()
}

func handleIPC() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		var req ipcplugin.Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			ipcplugin.SendError("invalid json")
			continue
		}

		switch req.Method {
		case "info":
			ipcplugin.SendResponse(ipcplugin.Response{
				Name:    pluginName,
				Version: pluginVersion,
			})

		case "initialize":
			// Request form from host
			schema, uiSchema := getUI(state.modelType)
			ipcplugin.SendResponse(ipcplugin.Response{
				Method:           "show_form",
				Title:            "Model Configuration",
				Schema:           schema,
				UISchema:         uiSchema,
				HandleFormChange: true,
			})

			// Wait for form result or change
			for scanner.Scan() {
				respLine := scanner.Text()
				var formResp map[string]interface{}
				json.Unmarshal([]byte(respLine), &formResp)

				if method, ok := formResp["method"].(string); ok && method == "form_change" {
					if data, ok := formResp["data"].(map[string]interface{}); ok {
						handleFormChange(data)
					}
					continue
				}

				if result, ok := formResp["result"].(map[string]interface{}); ok {
					updateState(result)
					ipcplugin.SendResponse(ipcplugin.Response{Result: "success"})
					break
				}
				if err, ok := formResp["error"].(string); ok {
					ipcplugin.SendError(err)
					break
				}
			}

		case "get_chart_config":
			ipcplugin.SendResponse(ipcplugin.Response{
				Result: ipcplugin.ChartConfig{
					Title:      state.modelType + " Simulation",
					AxisLabels: []string{"Time", "Value"},
				},
			})

		case "get_series_config":
			series := make([]ipcplugin.SeriesConfig, state.numSeries)
			for i := 0; i < state.numSeries; i++ {
				series[i] = ipcplugin.SeriesConfig{
					ID:    fmt.Sprintf("s%d", i),
					Name:  fmt.Sprintf("%s %d", state.modelType, i+1),
					Color: ipcplugin.ChartColors[i%len(ipcplugin.ChartColors)],
				}
			}
			ipcplugin.SendResponse(ipcplugin.Response{
				Result: series,
			})

		case "get_series_data":
			ipcplugin.SendBinaryData(generateData(req.SeriesID))

		default:
			ipcplugin.SendError("unknown method")
		}
	}
}

func handleFormChange(data map[string]interface{}) {
	newModel, _ := data["model"].(string)

	// If model changed, we must send update
	if newModel != "" && newModel != state.modelType {
		state.modelType = newModel
		schema, uiSchema := getUI(newModel)
		ipcplugin.SendFormUpdate(schema, uiSchema, nil)
		return
	}

	// Just send empty update to acknowledge if no model change
	ipcplugin.SendNoUpdate()
}

func updateState(data map[string]interface{}) {
	if v, ok := data["model"].(string); ok {
		state.modelType = v
	}
	if v, ok := data["numSeries"].(float64); ok {
		state.numSeries = int(v)
	}
	if v, ok := data["order"].(float64); ok {
		state.order = int(v)
	}
	if v, ok := data["multiplier"].(float64); ok {
		state.multiplier = int(v)
	}
	if v, ok := data["p"].(float64); ok {
		state.p = int(v)
	}
	if v, ok := data["d"].(float64); ok {
		state.d = int(v)
	}
	if v, ok := data["q"].(float64); ok {
		state.q = int(v)
	}
	if v, ok := data["noise"].(float64); ok {
		state.noise = v
	}
	if v, ok := data["amplitude"].(float64); ok {
		state.amplitude = v
	}
	if v, ok := data["frequency"].(float64); ok {
		state.frequency = v
	}
}

func getUI(model string) (interface{}, interface{}) {
	properties := map[string]interface{}{
		"model": map[string]interface{}{
			"type":    "string",
			"title":   "Model Type",
			"enum":    []string{"Random Walk", "ARIMA", "Sinusoidal"},
			"default": model,
		},
		"numSeries": map[string]interface{}{
			"type":    "integer",
			"title":   "Number of Series",
			"minimum": 1,
			"maximum": 10,
			"default": state.numSeries,
		},
		"order": map[string]interface{}{
			"type":    "integer",
			"title":   "Order (10^n)",
			"minimum": 1,
			"maximum": 6,
			"default": state.order,
		},
		"multiplier": map[string]interface{}{
			"type":    "integer",
			"title":   "Multiplier",
			"minimum": 1,
			"maximum": 9,
			"default": state.multiplier,
		},
	}

	uiSchema := map[string]interface{}{
		"numSeries": map[string]interface{}{
			"ui:widget": "range",
		},
		"order": map[string]interface{}{
			"ui:widget": "range",
		},
		"multiplier": map[string]interface{}{
			"ui:widget": "range",
		},
	}

	switch model {
	case "ARIMA":
		properties["p"] = map[string]interface{}{"type": "integer", "title": "p (AR)", "default": state.p}
		properties["d"] = map[string]interface{}{"type": "integer", "title": "d (I)", "default": state.d}
		properties["q"] = map[string]interface{}{"type": "integer", "title": "q (MA)", "default": state.q}
	case "Sinusoidal":
		properties["amplitude"] = map[string]interface{}{"type": "number", "title": "Amplitude", "default": state.amplitude}
		properties["frequency"] = map[string]interface{}{"type": "number", "title": "Frequency", "default": state.frequency}
	case "Random Walk":
		properties["noise"] = map[string]interface{}{"type": "number", "title": "Noise Level", "default": state.noise}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	return schema, uiSchema
}

func generateData(seriesID string) []float64 {
	numPoints := int(float64(state.multiplier) * math.Pow(10, float64(state.order)))

	// Use series index to jitter the seed
	var seriesIdx int
	fmt.Sscanf(seriesID, "s%d", &seriesIdx)

	rng := rand.New(rand.NewSource(state.seed + int64(seriesIdx*54321)))
	data := make([]float64, (numPoints+1)*2)

	t := 0.0
	y := 0.0
	data[0] = t
	data[1] = y

	for i := 1; i <= numPoints; i++ {
		t += 1.0
		switch state.modelType {
		case "Random Walk":
			y += rng.NormFloat64() * state.noise
		case "ARIMA":
			y = 0.8*y + rng.NormFloat64()
		case "Sinusoidal":
			phase := float64(seriesIdx) * 0.5
			y = state.amplitude*math.Sin(2*math.Pi*state.frequency*t+phase) + rng.NormFloat64()*0.1
		}
		data[i*2] = t
		data[i*2+1] = y
	}
	return data
}
