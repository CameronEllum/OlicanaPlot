// Package ipc provides infrastructure for loading and communicating with
// external plugins via subprocess IPC.
package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"olicanaplot/internal/logging"
	"olicanaplot/internal/plugins"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Loader discovers and manages IPC plugins.
type Loader struct {
	pluginsDir string
	logger     logging.Logger
}

// NewLoader creates a new IPC plugin loader.
func NewLoader(pluginsDir string, logger logging.Logger) *Loader {
	return &Loader{
		pluginsDir: pluginsDir,
		logger:     logger,
	}
}

// Discover finds and loads all IPC plugins in the plugins directory.
func (l *Loader) Discover() ([]*Plugin, error) {
	var result []*Plugin

	l.logger.Info("Scanning for IPC plugins", "dir", l.pluginsDir)

	// Check if plugins directory exists
	if _, err := os.Stat(l.pluginsDir); os.IsNotExist(err) {
		l.logger.Warn("IPC plugins directory not found", "dir", l.pluginsDir)
		return nil, nil
	}

	entries, err := os.ReadDir(l.pluginsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugins directory: %w", err)
	}

	execSuffix := ""
	if runtime.GOOS == "windows" {
		execSuffix = ".exe"
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Look for executable with same name as directory
		dirName := entry.Name()
		execName := dirName + execSuffix
		execPath := filepath.Join(l.pluginsDir, dirName, execName)

		if _, err := os.Stat(execPath); err != nil {
			l.logger.Debug("Skipping directory: no executable found", "dir", dirName, "exec", execName)
			continue
		}

		l.logger.Info("Found IPC plugin candidate", "path", execPath)

		plugin, err := NewPlugin(execPath)
		if err != nil {
			l.logger.Error("Failed to load IPC plugin", "dir", dirName, "error", err)
			continue
		}

		result = append(result, plugin)
	}

	l.logger.Info("IPC discovery complete", "count", len(result))
	return result, nil
}

// Plugin wraps an external process as a plugin.
type Plugin struct {
	mu           sync.Mutex
	execPath     string
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	stdout       *bufio.Reader
	name         string
	version      uint32
	filePatterns []plugins.FilePattern
	running      bool
	logger       logging.Logger
	app          *application.App
	commsMu      sync.Mutex // For synchronizing stdin/stdout access
}

// Request represents an IPC request message sent from the host.
type Request struct {
	Method           string                 `json:"method"`
	Args             string                 `json:"args,omitempty"`
	SeriesID         string                 `json:"series_id,omitempty"`
	PreferredStorage string                 `json:"preferred_storage,omitempty"`
	Data             map[string]interface{} `json:"data,omitempty"`
}

// Response represents an IPC response message received from a plugin.
// This structure follows IPC_PROTOCOL.md but uses json.RawMessage for Result
// to allow the host to unmarshal it into different concrete types.
type Response struct {
	Method           string          `json:"method,omitempty"` // For async messages like "log" or "show_form"
	Result           json.RawMessage `json:"result,omitempty"`
	Error            string          `json:"error,omitempty"`
	Type             string          `json:"type,omitempty"`
	Length           int             `json:"length,omitempty"`
	Storage          string          `json:"storage,omitempty"`
	Name             string          `json:"name,omitempty"`
	Version          uint32          `json:"version,omitempty"`
	Title            string          `json:"title,omitempty"`
	Schema           json.RawMessage `json:"schema,omitempty"`
	UISchema         json.RawMessage `json:"uiSchema,omitempty"`
	Data             json.RawMessage `json:"data,omitempty"`
	HandleFormChange bool            `json:"handle_form_change,omitempty"`
}

// PluginMetadata contains everything required for plugin discovery.
type PluginMetadata struct {
	Name         string                `json:"name"`
	FilePatterns []plugins.FilePattern `json:"patterns"`
}

// NewPlugin creates an IPC plugin wrapper and fetches its metadata.
func NewPlugin(execPath string) (*Plugin, error) {
	// Verify exe exists first
	if _, err := os.Stat(execPath); err != nil {
		return nil, fmt.Errorf("plugin executable not found at %s: %w", execPath, err)
	}

	// Generate fallback name from exe basename
	exeName := filepath.Base(execPath)
	// Remove extension
	ext := filepath.Ext(exeName)
	base := strings.TrimSuffix(exeName, ext)
	// Replace both underscores and dashes with spaces
	nameWithSpaces := strings.ReplaceAll(base, "-", " ")
	nameWithSpaces = strings.ReplaceAll(nameWithSpaces, "_", " ")
	displayName := strings.Title(nameWithSpaces)
	// Fix acronyms and special names
	displayName = strings.ReplaceAll(displayName, "Ipc", "IPC")
	displayName = strings.ReplaceAll(displayName, "Cpp", "C++")

	p := &Plugin{
		execPath: execPath,
		name:     displayName, // Default
		version:  1,
	}

	// Fetch metadata via CLI flag
	cmd := exec.Command(execPath, "--metadata")
	configureCommand(cmd, true)
	output, err := cmd.Output()
	if err == nil {
		var meta PluginMetadata
		if json.Unmarshal(output, &meta) == nil {
			if meta.Name != "" {
				p.name = meta.Name
			}
			p.filePatterns = meta.FilePatterns
		}
	}

	return p, nil
}

// start launches the plugin subprocess.
func (p *Plugin) start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return nil
	}

	p.cmd = exec.Command(p.execPath)
	configureCommand(p.cmd, true)

	stdin, err := p.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}
	p.stdin = stdin

	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	p.stdout = bufio.NewReader(stdout)

	// Capture stderr for debugging (goes to host stderr)
	p.cmd.Stderr = os.Stderr

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start plugin: %w", err)
	}

	p.running = true
	return nil
}

// fetchInfo gets the plugin name and version.
func (p *Plugin) fetchInfo() error {
	resp, err := p.sendRequest(Request{Method: "info"})
	if err != nil {
		return err
	}

	p.name = resp.Name
	p.version = resp.Version
	return nil
}

// sendRequest sends a request and reads the response, handling interleaved "log" messages.
func (p *Plugin) sendRequest(req Request) (*Response, error) {
	// If method is not info and not running, try starting
	if req.Method != "info" && !p.running {
		if err := p.start(); err != nil {
			return nil, err
		}
	}

	// Note: mu is locked for the entire send/receive to ensure sync communication
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil, fmt.Errorf("plugin not running")
	}

	return p.sendLockedRequest(req)
}

// sendLockedRequest performs the actual comms while holding necessary locks.
func (p *Plugin) sendLockedRequest(req Request) (*Response, error) {
	p.commsMu.Lock()
	defer p.commsMu.Unlock()

	return p.sendInternal(req)
}

func (p *Plugin) sendInternal(req Request) (*Response, error) {
	// Send request as JSON line
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	reqBytes = append(reqBytes, '\n')

	if p.logger != nil {
		p.logger.Debug("IPC -> PLUGIN", "json", strings.TrimSpace(string(reqBytes)))
	}

	if _, err := p.stdin.Write(reqBytes); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	for {
		// Read response line
		respLine, err := p.stdout.ReadString('\n')
		if err != nil {
			p.running = false
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		if p.logger != nil {
			p.logger.Debug("PLUGIN -> IPC", "json", strings.TrimSpace(respLine))
		}

		var resp Response
		if err := json.Unmarshal([]byte(strings.TrimSpace(respLine)), &resp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		// Handle asynchronous "log" method from plugin
		if resp.Method == "log" {
			if p.logger != nil {
				var logData struct {
					Level   string `json:"level"`
					Message string `json:"message"`
				}
				json.Unmarshal([]byte(respLine), &logData)

				switch strings.ToLower(logData.Level) {
				case "error":
					p.logger.Error(logData.Message, "component", p.name)
				case "warn":
					p.logger.Warn(logData.Message, "component", p.name)
				case "debug":
					p.logger.Debug(logData.Message, "component", p.name)
				default:
					p.logger.Info(logData.Message, "component", p.name)
				}
			}
			continue // Keep waiting for the actual response
		}

		// Handle "show_form" request from plugin
		if resp.Method == "show_form" {
			// We MUST release commsMu while waiting for the form to allow form_change events
			p.commsMu.Unlock()
			err := p.handleShowForm(resp)
			p.commsMu.Lock()
			if err != nil {
				return nil, err
			}
			continue // After handling the form, wait for plugin's final response
		}

		if resp.Error != "" {
			return nil, fmt.Errorf("plugin error: %s", resp.Error)
		}

		return &resp, nil
	}
}

// handleShowForm processes a request from the plugin to show a configuration form.
func (p *Plugin) handleShowForm(formMsg Response) error {
	if p.app == nil {
		return fmt.Errorf("plugin %s requested show_form but no application context available", p.name)
	}

	p.logger.Info("Plugin requested host-controlled form", "title", formMsg.Title)

	// Result channel for the form result
	resultChan := make(chan interface{})
	errChan := make(chan string)
	doneChan := make(chan struct{})

	// Unique string ID for this form request to avoid JS precision issues
	requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())

	// Register temporary event listener for form result
	unsub := p.app.Event.On(fmt.Sprintf("ipc-form-result-%s", requestID), func(e *application.CustomEvent) {
		if e.Data != nil {
			if errStr, ok := e.Data.(string); ok && strings.HasPrefix(errStr, "error:") {
				errChan <- strings.TrimPrefix(errStr, "error:")
			} else {
				resultChan <- e.Data
			}
		}
	})
	defer unsub()

	// Register listener for form changes (dynamic updates) if plugin requested it
	if formMsg.HandleFormChange {
		go func() {
			unsubChange := p.app.Event.On(fmt.Sprintf("ipc-form-change-%s", requestID), func(e *application.CustomEvent) {
				if e.Data == nil {
					return
				}
				data, ok := e.Data.(map[string]interface{})
				if !ok {
					return
				}

				// Send form_change request to plugin using the unified communication lock
				p.logger.Debug("Sending form_change to plugin", "requestID", requestID)
				resp, err := p.sendLockedRequest(Request{
					Method: "form_change",
					Data:   data,
				})
				if err != nil {
					p.logger.Error("Failed to send form_change to plugin", "error", err)
					return
				}

				// Always notify the frontend that the change has been processed to clear the loading state
				var schemaObj, uiSchemaObj, dataObj interface{}
				if len(resp.Schema) > 0 {
					json.Unmarshal(resp.Schema, &schemaObj)
				}
				if len(resp.UISchema) > 0 {
					json.Unmarshal(resp.UISchema, &uiSchemaObj)
				}
				if len(resp.Data) > 0 {
					json.Unmarshal(resp.Data, &dataObj)
				} else if len(resp.Result) > 0 {
					// Fallback to Result for data if it's a JSON object
					// Check first byte to see if it's '{'
					trimmed := strings.TrimSpace(string(resp.Result))
					if strings.HasPrefix(trimmed, "{") {
						json.Unmarshal(resp.Result, &dataObj)
					}
				}

				p.app.Event.Emit(fmt.Sprintf("ipc-form-update-%s", requestID), map[string]interface{}{
					"schema":   schemaObj,
					"uiSchema": uiSchemaObj,
					"data":     dataObj,
				})
			})
			defer unsubChange()
			<-doneChan
		}()
	}

	// Unmarshal schema and uiSchema so they are sent as objects, not raw bytes
	var schemaObj, uiSchemaObj interface{}
	if len(formMsg.Schema) > 0 {
		json.Unmarshal(formMsg.Schema, &schemaObj)
	}
	if len(formMsg.UISchema) > 0 {
		json.Unmarshal(formMsg.UISchema, &uiSchemaObj)
	}

	// Create a new window for the dialog
	dialogWindow := p.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:       formMsg.Title,
		Width:       500,
		Height:      500,
		AlwaysOnTop: true,
		URL:         fmt.Sprintf("/dialog.html?requestID=%s&title=%s&handleFormChange=%v", requestID, formMsg.Title, formMsg.HandleFormChange),
	})

	// Register listener for window resizing
	unsubResize := p.app.Event.On(`ipc-form-resize-`+requestID, func(e *application.CustomEvent) {
		if e.Data == nil {
			return
		}
		data, ok := e.Data.(map[string]interface{})
		if !ok {
			return
		}
		width, _ := data["width"].(float64)
		height, _ := data["height"].(float64)
		if width > 0 && height > 0 {
			// Add buffer for OS title bar (typically 30-48px)
			dialogWindow.SetSize(int(width), int(height)+48)
		}
	})
	defer unsubResize()

	// Register a one-time listener for the dialog to request its initial data
	unsubReady := p.app.Event.On(`ipc-form-ready-`+requestID, func(e *application.CustomEvent) {
		p.app.Event.Emit(`ipc-form-init-`+requestID, map[string]interface{}{
			"schema":           schemaObj,
			"uiSchema":         uiSchemaObj,
			"handleFormChange": formMsg.HandleFormChange,
		})
	})
	defer unsubReady()
	dialogWindow.Center()

	// Wait for result or error
	var finalResult interface{}
	var finalError string

	select {
	case res := <-resultChan:
		finalResult = res
	case err := <-errChan:
		finalError = err
	case <-time.After(5 * time.Minute):
		finalError = "timeout"
	}

	// SIGNAL GOROUTINE TO STOP BEFORE SENDING RESULT
	// This prevents a late form_change from being sent while the plugin is processing the result.
	unsub()
	close(doneChan)
	p.logger.Debug("Form session finished, stopping dynamic listeners", "requestID", requestID)

	// Ensure dialog window is closed if it hasn't been already
	if dialogWindow != nil {
		dialogWindow.Close()
	}

	// Send result back to plugin via stdin, holding commsMu
	p.commsMu.Lock()
	defer p.commsMu.Unlock()

	var response map[string]interface{}
	if finalError != "" {
		response = map[string]interface{}{
			"error": finalError,
		}
	} else {
		response = map[string]interface{}{
			"result": finalResult,
		}
	}

	respBytes, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal form response: %w", err)
	}
	respBytes = append(respBytes, '\n')

	if p.logger != nil {
		p.logger.Debug("IPC -> PLUGIN (form-result)", "json", strings.TrimSpace(string(respBytes)))
	}

	if _, err := p.stdin.Write(respBytes); err != nil {
		return fmt.Errorf("failed to write form response to plugin: %w", err)
	}

	return nil
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return p.name
}

// Version returns the API version.
func (p *Plugin) Version() uint32 {
	return p.version
}

// Path returns the executable path.
func (p *Plugin) Path() string {
	return p.execPath
}

// GetFilePatterns returns the list of file patterns supported by the plugin.
func (p *Plugin) GetFilePatterns() []plugins.FilePattern {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.filePatterns
}

// Initialize executes plugin initialization.
func (p *Plugin) Initialize(ctx interface{}, initStr string, logger logging.Logger) (string, error) {
	p.logger = logger
	if app, ok := ctx.(*application.App); ok {
		p.app = app
	}

	if !p.running {
		if err := p.start(); err != nil {
			return "", err
		}
	}

	logger.Debug("Sending initialize request to IPC plugin")
	resp, err := p.sendRequest(Request{
		Method: "initialize",
		Args:   initStr,
	})
	if err != nil {
		logger.Error("IPC plugin initialization failed", "error", err)
		return "", err
	}
	logger.Info("IPC plugin initialized successfully")
	return string(resp.Result), nil
}

// GetChartConfig returns chart configuration.
func (p *Plugin) GetChartConfig(args string) (*plugins.ChartConfig, error) {
	resp, err := p.sendRequest(Request{
		Method: "get_chart_config",
		Args:   args,
	})
	if err != nil {
		return nil, err
	}

	var config plugins.ChartConfig
	if err := json.Unmarshal(resp.Result, &config); err != nil {
		return nil, fmt.Errorf("failed to parse chart config: %w", err)
	}
	return &config, nil
}

// GetChartConfig returns chart configuration. (Note: duplicate comment in previous file, fixed below)
// GetSeriesConfig returns series configuration.
func (p *Plugin) GetSeriesConfig() ([]plugins.SeriesConfig, error) {
	resp, err := p.sendRequest(Request{
		Method: "get_series_config",
	})
	if err != nil {
		return nil, err
	}

	var series []plugins.SeriesConfig
	if err := json.Unmarshal(resp.Result, &series); err != nil {
		return nil, fmt.Errorf("failed to parse series config: %w", err)
	}
	return series, nil
}

// GetSeriesData returns binary float64 data for the specified series ID.
func (p *Plugin) GetSeriesData(seriesID string, preferredStorage string) ([]float64, string, error) {
	// Re-check running status - sendRequest handles it too but GetSeriesData is custom
	if !p.running {
		if err := p.start(); err != nil {
			return nil, "", err
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.commsMu.Lock()
	defer p.commsMu.Unlock()

	req := Request{
		Method:           "get_series_data",
		SeriesID:         seriesID,
		PreferredStorage: preferredStorage,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal request: %w", err)
	}
	reqBytes = append(reqBytes, '\n')

	if p.logger != nil {
		p.logger.Debug("IPC -> PLUGIN", "json", strings.TrimSpace(string(reqBytes)))
	}

	if _, err := p.stdin.Write(reqBytes); err != nil {
		return nil, "", fmt.Errorf("failed to write request: %w", err)
	}

	for {
		// Read header line
		respLine, err := p.stdout.ReadString('\n')
		if err != nil {
			p.running = false
			return nil, "", fmt.Errorf("failed to read response header: %w", err)
		}

		if p.logger != nil {
			p.logger.Debug("PLUGIN -> IPC", "json", strings.TrimSpace(respLine))
		}

		var resp Response
		if err := json.Unmarshal([]byte(strings.TrimSpace(respLine)), &resp); err != nil {
			return nil, "", fmt.Errorf("failed to parse response header: %w", err)
		}

		// Handle intermediate "log" messages
		if resp.Method == "log" {
			if p.logger != nil {
				var logData struct {
					Level   string `json:"level"`
					Message string `json:"message"`
				}
				json.Unmarshal([]byte(respLine), &logData)
				switch strings.ToLower(logData.Level) {
				case "error":
					p.logger.Error(logData.Message, "component", p.name)
				case "warn":
					p.logger.Warn(logData.Message, "component", p.name)
				case "debug":
					p.logger.Debug(logData.Message, "component", p.name)
				default:
					p.logger.Info(logData.Message, "component", p.name)
				}
			}
			continue
		}

		if resp.Error != "" {
			return nil, "", fmt.Errorf("plugin error: %s", resp.Error)
		}

		if resp.Type != "binary" {
			return nil, "", fmt.Errorf("expected binary response, got: %s", resp.Type)
		}

		// Read binary data (resp.Length bytes)
		binaryData := make([]byte, resp.Length)
		if _, err := io.ReadFull(p.stdout, binaryData); err != nil {
			return nil, "", fmt.Errorf("failed to read binary data: %w", err)
		}

		// Convert bytes to float64 slice
		return bytesToFloats(binaryData), resp.Storage, nil
	}
}

// bytesToFloats converts little-endian bytes to float64 slice without copying.
func bytesToFloats(data []byte) []float64 {
	if len(data) == 0 {
		return nil
	}
	return unsafe.Slice((*float64)(unsafe.Pointer(&data[0])), len(data)/8)
}

// Close stops the plugin process.
func (p *Plugin) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	if p.stdin != nil {
		p.stdin.Close()
	}

	if p.cmd != nil && p.cmd.Process != nil {
		// Give it a moment to exit normally on stdin close before killing
		done := make(chan error, 1)
		go func() {
			done <- p.cmd.Wait()
		}()

		select {
		case <-done:
			// exited cleanly
		case <-time.After(500 * time.Millisecond):
			p.cmd.Process.Kill()
		}
	}

	p.running = false
	return nil
}
