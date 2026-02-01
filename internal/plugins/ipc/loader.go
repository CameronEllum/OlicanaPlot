// Package ipc provides infrastructure for loading and communicating with
// external plugins via subprocess IPC.
package ipc

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"olicanaplot/internal/logging"
	"olicanaplot/internal/plugins"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Loader discovers and manages IPC plugins.
type Loader struct {
	pluginsDir string
}

// NewLoader creates a new IPC plugin loader.
func NewLoader(pluginsDir string) *Loader {
	return &Loader{pluginsDir: pluginsDir}
}

// Discover finds and loads all IPC plugins in the plugins directory.
func (l *Loader) Discover() ([]*Plugin, error) {
	var result []*Plugin

	log.Printf("Scanning for IPC plugins in: %s", l.pluginsDir)

	// Check if plugins directory exists
	if _, err := os.Stat(l.pluginsDir); os.IsNotExist(err) {
		log.Printf("IPC plugins directory not found: %s", l.pluginsDir)
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
			log.Printf("  - Skipped directory %s: no executable %s found", dirName, execName)
			continue
		}

		log.Printf("  + Found IPC plugin candidate: %s", execPath)

		plugin, err := NewPlugin(execPath)
		if err != nil {
			log.Printf("Failed to load IPC plugin %s: %v", dirName, err)
			continue
		}

		result = append(result, plugin)
	}

	log.Printf("IPC discovery complete. Found %d plugin(s).", len(result))
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
}

// Request represents an IPC request message sent from the host.
type Request struct {
	Method   string `json:"method"`
	Args     string `json:"args,omitempty"`
	SeriesID string `json:"series_id,omitempty"`
}

// Response represents an IPC response message received from a plugin.
// This structure follows IPC_PROTOCOL.md but uses json.RawMessage for Result
// to allow the host to unmarshal it into different concrete types.
type Response struct {
	Method   string          `json:"method,omitempty"` // For async messages like "log" or "show_form"
	Result   json.RawMessage `json:"result,omitempty"`
	Error    string          `json:"error,omitempty"`
	Type     string          `json:"type,omitempty"`
	Length   int             `json:"length,omitempty"`
	Name     string          `json:"name,omitempty"`
	Version  uint32          `json:"version,omitempty"`
	Title    string          `json:"title,omitempty"`
	Schema   json.RawMessage `json:"schema,omitempty"`
	UISchema json.RawMessage `json:"uiSchema,omitempty"`
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
	configureCommand(p.cmd, false)

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

	// Send request as JSON line
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	reqBytes = append(reqBytes, '\n')

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
			if err := p.handleShowForm(resp); err != nil {
				return nil, err
			}
			continue // After handling the form and sending result back to plugin, wait for plugin's final response
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

	// Unmarshal schema and uiSchema so they are sent as objects, not raw bytes
	var schemaObj, uiSchemaObj interface{}
	if len(formMsg.Schema) > 0 {
		json.Unmarshal(formMsg.Schema, &schemaObj)
	}
	if len(formMsg.UISchema) > 0 {
		json.Unmarshal(formMsg.UISchema, &uiSchemaObj)
	}

	// Emit event to frontend to show the form
	p.app.Event.Emit("ipc-show-form", map[string]interface{}{
		"requestID": requestID,
		"title":     formMsg.Title,
		"schema":    schemaObj,
		"uiSchema":  uiSchemaObj,
	})

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

	// Send result back to plugin via stdin
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

// GetSeriesData returns series data. For binary responses, reads raw bytes.
func (p *Plugin) GetSeriesData(seriesID string) ([]float64, error) {
	// Re-check running status - sendRequest handles it too but GetSeriesData is custom
	if !p.running {
		if err := p.start(); err != nil {
			return nil, err
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	req := Request{
		Method:   "get_series_data",
		SeriesID: seriesID,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	reqBytes = append(reqBytes, '\n')

	if _, err := p.stdin.Write(reqBytes); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	for {
		// Read header line
		respLine, err := p.stdout.ReadString('\n')
		if err != nil {
			p.running = false
			return nil, fmt.Errorf("failed to read response header: %w", err)
		}

		var resp Response
		if err := json.Unmarshal([]byte(strings.TrimSpace(respLine)), &resp); err != nil {
			return nil, fmt.Errorf("failed to parse response header: %w", err)
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
			return nil, fmt.Errorf("plugin error: %s", resp.Error)
		}

		if resp.Type != "binary" {
			return nil, fmt.Errorf("expected binary response, got: %s", resp.Type)
		}

		// Read binary data (resp.Length bytes)
		binaryData := make([]byte, resp.Length)
		if _, err := io.ReadFull(p.stdout, binaryData); err != nil {
			return nil, fmt.Errorf("failed to read binary data: %w", err)
		}

		// Convert bytes to float64 slice
		return bytesToFloats(binaryData), nil
	}
}

// bytesToFloats converts little-endian bytes to float64 slice.
func bytesToFloats(data []byte) []float64 {
	count := len(data) / 8
	result := make([]float64, count)
	for i := 0; i < count; i++ {
		bits := binary.LittleEndian.Uint64(data[i*8:])
		result[i] = math.Float64frombits(bits)
	}
	return result
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
