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

// NewCsvDialog creates a new configuration dialog
func NewCsvDialog(app *application.App, file string, headers []string) *CsvDialog {
	d := &CsvDialog{
		app:    app,
		result: make(chan ConfigResult, 1),
	}

	d.window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:       "CSV Column Selection",
		Width:       700,
		Height:      750,
		AlwaysOnTop: true,
		Frameless:   true,
		Hidden:      true,
		URL:         fmt.Sprintf("/csv-dialog.html?file=%s&headers=%s", file, encodeHeaders(headers)),
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
