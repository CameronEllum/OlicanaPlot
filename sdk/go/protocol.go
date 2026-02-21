package sdk

import (
	"encoding/json"
	"os"
	"unsafe"
)

// Request represents an IPC request from the host.
type Request struct {
	Method           string                 `json:"method"`
	Args             string                 `json:"args,omitempty"`
	SeriesID         string                 `json:"series_id,omitempty"`
	PreferredStorage string                 `json:"preferred_storage,omitempty"` // interleaved or arrays
	Data             map[string]interface{} `json:"data,omitempty"`              // For form_change
}

// Response represents an IPC response to the host.
type Response struct {
	Method           string                 `json:"method,omitempty"` // For async messages like "log" or "show_form"
	Result           interface{}            `json:"result,omitempty"`
	Error            string                 `json:"error,omitempty"`
	Type             string                 `json:"type,omitempty"`
	Length           int                    `json:"length,omitempty"`
	Storage          string                 `json:"storage,omitempty"` // interleaved or arrays
	Name             string                 `json:"name,omitempty"`
	Version          uint32                 `json:"version,omitempty"`
	Title            string                 `json:"title,omitempty"`    // For show_form
	Schema           interface{}            `json:"schema,omitempty"`   // For form updates
	UISchema         interface{}            `json:"uiSchema,omitempty"` // For form updates
	Data             map[string]interface{} `json:"data,omitempty"`     // For form updates
	HandleFormChange bool                   `json:"handle_form_change,omitempty"`
}

// IMPORTANT: The following structs are intentionally duplicated from internal/plugins
// instead of using type aliases. This is a workaround for a Wails 3 binding
// generation bug where cross-package type aliases result in broken JavaScript
// imports (e.g., undefined "$0" references).
//
// Plugins use this package, while the host uses internal/plugins.
// If you modify these, you MUST also modify the corresponding structs in
// internal/plugins/plugin.go and ensure internal/plugins/sync_test.go passes.

// ChartConfig holds chart display configuration.
type ChartConfig struct {
	Title string            `json:"title"`
	Grid  *GridConfig       `json:"grid,omitempty"`
	Axes  []AxisGroupConfig `json:"axes,omitempty"`
	LinkX *bool             `json:"link_x,omitempty"`
	LinkY *bool             `json:"link_y,omitempty"`
}

// GridConfig describes the subplot grid layout.
type GridConfig struct {
	Rows int `json:"rows"`
	Cols int `json:"cols"`
}

// AxisConfig describes an axis within a subplot.
type AxisConfig struct {
	Title    string   `json:"title,omitempty"`
	Position string   `json:"position,omitempty"` // "bottom", "top", "left", "right"
	Unit     string   `json:"unit,omitempty"`
	Type     string   `json:"type,omitempty"` // "linear", "log", "date"
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
}

// SubPlot describes a cell in the chart grid.
type SubPlot struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

// AxisGroupConfig describes all axes and series for one subplot cell.
type AxisGroupConfig struct {
	Title   string       `json:"title,omitempty"`
	Subplot *SubPlot     `json:"subplot,omitempty"`
	XAxes   []AxisConfig `json:"x_axes,omitempty"`
	YAxes   []AxisConfig `json:"y_axes,omitempty"`
}

// SeriesConfig describes a data series metadata.
type SeriesConfig struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Color      string   `json:"color,omitempty"`
	Subplot    *SubPlot `json:"subplot,omitempty"`
	LineType   string   `json:"line_type,omitempty"` // "solid", "dashed", "dotted"
	LineWidth  *float64 `json:"line_width,omitempty"`
	MarkerType string   `json:"marker_type,omitempty"` // "none", "circle", "square", "triangle", "diamond", "cross", "x"
	MarkerSize *float64 `json:"marker_size,omitempty"`
	MarkerFill string   `json:"marker_fill,omitempty"` // "empty" or "solid"
	Unit       string   `json:"unit,omitempty"`
	Visible    *bool    `json:"visible"`
	YAxis      string   `json:"y_axis,omitempty"` // references Y axis title
}

// FilePattern describes a file type supported by a plugin.
type FilePattern struct {
	Description string   `json:"description"`
	Patterns    []string `json:"patterns"`
}

// SendResponse sends a JSON response to stdout.
func SendResponse(resp Response) {
	respJSON, _ := json.Marshal(resp)
	os.Stdout.Write(respJSON)
	os.Stdout.Write([]byte("\n"))
	os.Stdout.Sync()
}

// SendError sends an error response to stdout.
func SendError(msg string) {
	SendResponse(Response{Error: msg})
}

// SendBinaryData sends binary float64 data following a JSON header.
func SendBinaryData(data []float64, storage string) {
	binaryData := floatsToBytes(data)
	headerJSON, _ := json.Marshal(Response{
		Type:    "binary",
		Length:  len(binaryData),
		Storage: storage,
	})

	os.Stdout.Write(headerJSON)
	os.Stdout.Write([]byte("\n"))
	os.Stdout.Sync()

	os.Stdout.Write(binaryData)
	os.Stdout.Sync()
}

// Log sends an asynchronous log message to the host.
func Log(level, message string) {
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

// SendFormUpdate sends an updated form configuration.
func SendFormUpdate(schema, uiSchema interface{}, data map[string]interface{}) {
	resp := Response{
		Schema:   schema,
		UISchema: uiSchema,
		Data:     data,
	}
	SendResponse(resp)
}

// SendShowForm requests the host to show a form with initial data.
func SendShowForm(title string, schema, uiSchema interface{}, data map[string]interface{}) {
	resp := Response{
		Method:   "show_form",
		Title:    title,
		Schema:   schema,
		UISchema: uiSchema,
		Data:     data,
	}
	SendResponse(resp)
}

// SendNoUpdate indicates no UI change is needed.
func SendNoUpdate() {
	os.Stdout.Write([]byte("{}\n"))
	os.Stdout.Sync()
}

// FloatsToBytes converts a float64 slice to little-endian bytes without copying.
func floatsToBytes(data []float64) []byte {
	if len(data) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&data[0])), len(data)*8)
}
