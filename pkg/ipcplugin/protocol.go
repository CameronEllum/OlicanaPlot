package ipcplugin

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

// ChartConfig holds chart display configuration.
type ChartConfig struct {
	Title      string   `json:"title"`
	AxisLabels []string `json:"axis_labels"`
}

// SeriesConfig describes a data series metadata.
type SeriesConfig struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// ChartColors provides the standard Plotly-inspired color palette.
var ChartColors = []string{
	"#636EFA", "#EF553B", "#00CC96", "#AB63FA", "#FFA15A",
	"#19D3F3", "#FF6692", "#B6E880", "#FF97FF", "#FECB52",
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

// SendShowForm requests the host to show a form.
func SendShowForm(title string, schema, uiSchema interface{}) {
	resp := Response{
		Method:   "show_form",
		Title:    title,
		Schema:   schema,
		UISchema: uiSchema,
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
