package function_generator

import (
	"encoding/json"
	"fmt"
	"olicanaplot/internal/appconfig"
	"olicanaplot/internal/funceval"
	"olicanaplot/internal/logging"
	"olicanaplot/internal/plugins"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const pluginName = "Function Plotter"

// Plugin implements the function plotter generator.
type Plugin struct {
	mu           sync.RWMutex
	config       *appconfig.ConfigService
	logger       logging.Logger
	expression   string
	functionName string
	xMin         float64
	xMax         float64
	numPoints    int
}

type ConfigResult struct {
	PresetFunction string  `json:"presetFunction"`
	FunctionName   string  `json:"functionName"`
	Expression     string  `json:"expression"`
	XMin           float64 `json:"xMin"`
	XMax           float64 `json:"xMax"`
	NumPoints      int     `json:"numPoints"`
	Cancelled      bool    `json:"-"`
}

// Built-in presets
var builtinPresets = []appconfig.FunctionPreset{
	{Name: "Damped Sine", Expression: "exp(-0.01*x) * sin(x * 0.1)", XMin: 0, XMax: 500, NumPoints: 1000},
	{Name: "Beats", Expression: "sin(x * 0.1) + sin(x * 0.11)", XMin: 0, XMax: 1000, NumPoints: 2000},
	{Name: "Sine", Expression: "sin(x * 0.1)", XMin: 0, XMax: 360, NumPoints: 361},
	{Name: "Cosine", Expression: "cos(x * 0.1)", XMin: 0, XMax: 360, NumPoints: 361},
	{Name: "Chirp", Expression: "sin(x * x * 0.0001)", XMin: 0, XMax: 1000, NumPoints: 2000},
}

// New creates a new function plotter plugin.
func New(config *appconfig.ConfigService) *Plugin {
	return &Plugin{
		config:       config,
		expression:   builtinPresets[0].Expression,
		functionName: builtinPresets[0].Name,
		xMin:         builtinPresets[0].XMin,
		xMax:         builtinPresets[0].XMax,
		numPoints:    builtinPresets[0].NumPoints,
	}
}

func (p *Plugin) Name() string {
	return pluginName
}

func (p *Plugin) Version() uint32 {
	return plugins.PluginAPIVersion
}

func (p *Plugin) Path() string {
	return ""
}

func (p *Plugin) GetFilePatterns() []plugins.FilePattern {
	return nil
}

func (p *Plugin) Initialize(ctx interface{}, initStr string, logger logging.Logger) (string, error) {
	p.logger = logger
	logger.Debug("Initializing function plotter")

	if initStr != "" {
		var cfg ConfigResult
		if err := json.Unmarshal([]byte(initStr), &cfg); err == nil {
			p.applyConfig(cfg)
			return "{}", nil
		}
	}

	app, ok := ctx.(*application.App)
	if !ok || app == nil {
		return "{}", nil
	}

	result := p.showConfigDialog(app)
	if result.Cancelled {
		return "{}", fmt.Errorf("cancelled")
	}

	p.applyConfig(result)

	// Save as user preset if a name is provided and it's not a direct built-in match
	if result.FunctionName != "" {
		isBuiltin := false
		for _, b := range builtinPresets {
			if b.Name == result.FunctionName && b.Expression == result.Expression {
				isBuiltin = true
				break
			}
		}
		if !isBuiltin {
			p.config.AddFunctionPreset(appconfig.FunctionPreset{
				Name:       result.FunctionName,
				Expression: result.Expression,
				XMin:       result.XMin,
				XMax:       result.XMax,
				NumPoints:  result.NumPoints,
			})
		}
	}

	return "{}", nil
}

func (p *Plugin) applyConfig(cfg ConfigResult) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.expression = cfg.Expression
	p.functionName = cfg.FunctionName
	p.xMin = cfg.XMin
	p.xMax = cfg.XMax
	p.numPoints = cfg.NumPoints
}

func (p *Plugin) showConfigDialog(app *application.App) ConfigResult {
	requestID := fmt.Sprintf("function_generator-%p", p)
	resultChan := make(chan ConfigResult, 1)
	var window *application.WebviewWindow

	// Prepare presets for dropdown
	userPresets := p.config.GetFunctionPresets()
	var enum []string
	presetMap := make(map[string]appconfig.FunctionPreset)

	for _, b := range builtinPresets {
		label := fmt.Sprintf("%s: %s", b.Name, b.Expression)
		enum = append(enum, label)
		presetMap[label] = b
	}
	for _, u := range userPresets {
		label := fmt.Sprintf("★ %s: %s", u.Name, u.Expression)
		enum = append(enum, label)
		presetMap[label] = u
	}

	schema := map[string]interface{}{
		"type":  "object",
		"title": "Function Plotter Configuration",
		"properties": map[string]interface{}{
			"presetFunction": map[string]interface{}{
				"title":   "Presets",
				"type":    "string",
				"enum":    enum,
				"default": enum[0],
			},
			"functionName": map[string]interface{}{
				"title":   "Function Name",
				"type":    "string",
				"default": builtinPresets[0].Name,
			},
			"expression": map[string]interface{}{
				"title":   "Function Expression y = f(x)",
				"type":    "string",
				"default": builtinPresets[0].Expression,
			},
			"xMin": map[string]interface{}{
				"title":   "X Min",
				"type":    "number",
				"default": builtinPresets[0].XMin,
			},
			"xMax": map[string]interface{}{
				"title":   "X Max",
				"type":    "number",
				"default": builtinPresets[0].XMax,
			},
			"numPoints": map[string]interface{}{
				"title":   "Number of Points",
				"type":    "integer",
				"minimum": 2,
				"maximum": 1000000,
				"default": builtinPresets[0].NumPoints,
			},
		},
	}

	uiSchema := map[string]interface{}{
		"ui:order": []string{"presetFunction", "functionName", "expression", "xMin", "xMax", "numPoints"},
	}

	// Handle form change for presets
	lastPreset := enum[0]
	app.Event.On(fmt.Sprintf("ipc-form-change-%s", requestID), func(e *application.CustomEvent) {
		if data, ok := e.Data.(map[string]interface{}); ok {
			if presetLabel, ok := data["presetFunction"].(string); ok {
				if presetLabel != lastPreset {
					lastPreset = presetLabel
					if preset, ok := presetMap[presetLabel]; ok {
						app.Event.Emit(fmt.Sprintf("ipc-form-update-%s", requestID), map[string]interface{}{
							"data": map[string]interface{}{
								"functionName": preset.Name,
								"expression":   preset.Expression,
								"xMin":         preset.XMin,
								"xMax":         preset.XMax,
								"numPoints":    preset.NumPoints,
							},
						})
					}
				}
			}
		}
	})

	app.Event.On(fmt.Sprintf("ipc-form-result-%s", requestID), func(e *application.CustomEvent) {
		if e.Data == "error:cancelled" {
			resultChan <- ConfigResult{Cancelled: true}
			return
		}
		if data, ok := e.Data.(map[string]interface{}); ok {
			res := ConfigResult{
				FunctionName: data["functionName"].(string),
				Expression:   data["expression"].(string),
				XMin:         data["xMin"].(float64),
				XMax:         data["xMax"].(float64),
				NumPoints:    int(data["numPoints"].(float64)),
			}
			resultChan <- res
		}
	})

	app.Event.On(fmt.Sprintf("ipc-form-ready-%s", requestID), func(e *application.CustomEvent) {
		app.Event.Emit(fmt.Sprintf("ipc-form-init-%s", requestID), map[string]interface{}{
			"schema":             schema,
			"uiSchema":           uiSchema,
			"handle_form_change": true,
		})
	})

	// Add resize listener
	app.Event.On(fmt.Sprintf("ipc-form-resize-%s", requestID), func(e *application.CustomEvent) {
		if data, ok := e.Data.(map[string]interface{}); ok {
			width, _ := data["width"].(float64)
			height, _ := data["height"].(float64)
			if width > 0 && height > 0 {
				window.SetSize(int(width), int(height)+48)
			}
		}
	})

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:        requestID,
		Title:       "Function Plotter",
		Width:       500,
		Height:      600,
		AlwaysOnTop: true,
		URL:         fmt.Sprintf("/dialog.html?requestID=%s", requestID),
	})

	window.Show()
	window.Center()
	window.Focus()

	res := <-resultChan
	window.Close()
	return res
}

func (p *Plugin) GetChartConfig(args string) (*plugins.ChartConfig, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return &plugins.ChartConfig{
		Title:      p.functionName,
		AxisLabels: []string{"X", "Y"},
	}, nil
}

func (p *Plugin) GetSeriesConfig() ([]plugins.SeriesConfig, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return []plugins.SeriesConfig{
		{
			ID:   "func_0",
			Name: p.functionName,
		},
	}, nil
}

func (p *Plugin) GetSeriesData(seriesID string, preferredStorage string) ([]float64, string, error) {
	p.mu.RLock()
	exprStr := p.expression
	xMin := p.xMin
	xMax := p.xMax
	numPoints := p.numPoints
	p.mu.RUnlock()

	eval, err := funceval.Compile(exprStr)
	if err != nil {
		return nil, "", err
	}

	result := make([]float64, numPoints*2)
	isArrays := preferredStorage == "arrays"
	storage := "interleaved"
	if isArrays {
		storage = "arrays"
	}

	dx := (xMax - xMin) / float64(numPoints-1)
	for i := 0; i < numPoints; i++ {
		x := xMin + float64(i)*dx
		y, err := eval.Eval(x)
		if err != nil {
			y = 0 // Or handle error
		}

		if isArrays {
			result[i] = x
			result[numPoints+i] = y
		} else {
			result[i*2] = x
			result[i*2+1] = y
		}
	}

	return result, storage, nil
}

func (p *Plugin) Close() error {
	return nil
}
