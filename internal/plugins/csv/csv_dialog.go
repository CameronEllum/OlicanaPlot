package csv

import (
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ConfigResult holds the configuration result
type ConfigResult struct {
	XColumn  string
	YColumns []string
	Ok       bool
}

// CsvDialog handles the configuration window
type CsvDialog struct {
	window *application.WebviewWindow
	result chan ConfigResult
	app    *application.App
}

// NewCsvDialog creates a new configuration dialog using the standardized SchemaForm
func NewCsvDialog(app *application.App, file string, headers []string) *CsvDialog {
	d := &CsvDialog{
		app:    app,
		result: make(chan ConfigResult, 1),
	}

	requestID := fmt.Sprintf("csv-%p", d)

	// Create JSON Schema for column selection
	var xOptions []map[string]interface{}
	xOptions = append(xOptions, map[string]interface{}{"const": "Index", "title": "Index (0 to N)"})
	for _, h := range headers {
		xOptions = append(xOptions, map[string]interface{}{"const": h, "title": h})
	}

	var yOptions []map[string]interface{}
	for _, h := range headers {
		yOptions = append(yOptions, map[string]interface{}{"const": h, "title": h})
	}

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
		"type":  "object",
		"title": "CSV Column Selection",
		"properties": map[string]interface{}{
			"xColumn": map[string]interface{}{
				"title":   "X Column (Domain)",
				"type":    "string",
				"oneOf":   xOptions,
				"default": defaultX,
			},
			"yColumns": map[string]interface{}{
				"title": "Y Columns (Series)",
				"type":  "array",
				"items": map[string]interface{}{
					"type":  "string",
					"oneOf": yOptions,
				},
				"default": defaultY,
			},
		},
	}

	// Listen for the result from the Svelte dialog
	app.Event.On(fmt.Sprintf("ipc-form-result-%s", requestID), func(e *application.CustomEvent) {
		if e.Data != nil {
			if e.Data == "error:cancelled" {
				d.Cancel()
				return
			}
			if data, ok := e.Data.(map[string]interface{}); ok {
				xCol, _ := data["xColumn"].(string)
				yColsRaw, _ := data["yColumns"].([]interface{})
				yCols := make([]string, 0, len(yColsRaw))
				for _, v := range yColsRaw {
					if s, ok := v.(string); ok {
						yCols = append(yCols, s)
					}
				}
				d.Submit(xCol, yCols)
			}
		}
	})

	// Listen for the ready event to send data
	app.Event.On(fmt.Sprintf("ipc-form-ready-%s", requestID), func(e *application.CustomEvent) {
		app.Event.Emit(fmt.Sprintf("ipc-form-init-%s", requestID), map[string]interface{}{
			"schema": schema,
			"data": map[string]interface{}{
				"xColumn":  defaultX,
				"yColumns": defaultY,
			},
			"handleFormChange": false,
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
		Title:       "CSV Column Selection",
		Width:       500,
		Height:      600,
		AlwaysOnTop: true,
		Frameless:   false, // Use system frame for consistent UX as requested
		URL:         fmt.Sprintf("/dialog.html?requestID=%s", requestID),
	})

	return d
}

// Show displays the dialog and waits for result
func (d *CsvDialog) Show() ConfigResult {
	d.window.Show()
	d.window.Center()
	d.window.Focus()

	// Wait for result
	r := <-d.result
	return r
}

func (d *CsvDialog) Submit(xColumn string, yColumns []string) {
	d.result <- ConfigResult{
		XColumn:  xColumn,
		YColumns: yColumns,
		Ok:       true,
	}
	d.window.Close()
}

func (d *CsvDialog) Cancel() {
	// Only send cancelled result if the channel isn't closed or full
	// But since this channel is buffered (1) and this is a one-shot dialog,
	// we just send.
	select {
	case d.result <- ConfigResult{Ok: false}:
	default:
	}
	d.window.Close()
}
