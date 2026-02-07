package appconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"olicanaplot/internal/logging"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ConfigService handles application configuration and settings.
type ConfigService struct {
	mu            sync.RWMutex
	app           *application.App
	optionsWindow *application.WebviewWindow
	configPath    string
	logPath       string
	chartLibrary  string
	theme         string
	logLevel      string
}

// configData is the structure we save to disk
type configData struct {
	LogPath      string `json:"logPath"`
	ChartLibrary string `json:"chartLibrary"`
	Theme        string `json:"theme"`
	LogLevel     string `json:"logLevel"`
}

// NewConfigService creates a new config service with default values.
func NewConfigService() *ConfigService {
	configDir, err := os.UserConfigDir()
	var appDir string
	if err == nil {
		appDir = filepath.Join(configDir, "OlicanaPlot")
	} else {
		appDir = "."
	}

	// Ensure app directory exists
	os.MkdirAll(appDir, 0755)

	s := &ConfigService{
		configPath:   filepath.Join(appDir, "config.json"),
		logPath:      filepath.Join(appDir, "olicana.log"),
		chartLibrary: "echarts", // Default to ECharts
		theme:        "light",   // Default to light
		logLevel:     "info",    // Default to info
	}

	s.loadConfig()
	return s
}

func (s *ConfigService) SetApp(app *application.App) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.app = app
}

func (s *ConfigService) OpenOptions() {
	s.mu.Lock()
	app := s.app
	s.mu.Unlock()

	if app == nil {
		return
	}

	// Create new frameless options window
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "options",
		Title:     "Options",
		Width:     800,
		Height:    600,
		Frameless: true,
		URL:       "/options.html",
	})
}

func (s *ConfigService) loadConfig() {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return // File might not exist yet, use defaults
	}

	var cfg configData
	if err := json.Unmarshal(data, &cfg); err != nil {
		return
	}

	if cfg.LogPath != "" {
		s.logPath = cfg.LogPath
	}
	if cfg.ChartLibrary != "" {
		s.chartLibrary = cfg.ChartLibrary
	}
	if cfg.Theme != "" {
		s.theme = cfg.Theme
	}
	if cfg.LogLevel != "" {
		s.logLevel = cfg.LogLevel
	}

	// Apply log level
	logging.SetLevel(s.logLevel)
}

func (s *ConfigService) saveConfig() {
	s.mu.RLock()
	cfg := configData{
		LogPath:      s.logPath,
		ChartLibrary: s.chartLibrary,
		Theme:        s.theme,
		LogLevel:     s.logLevel,
	}
	s.mu.RUnlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return
	}

	os.WriteFile(s.configPath, data, 0644)
}

// GetLogPath returns the current log path.
func (s *ConfigService) GetLogPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.logPath
}

// SetLogPath updates the log path.
func (s *ConfigService) SetLogPath(path string) {
	s.mu.Lock()
	s.logPath = path
	s.mu.Unlock()
	s.saveConfig()
}

// GetChartLibrary returns the current chart library ("echarts" or "plotly").
func (s *ConfigService) GetChartLibrary() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.chartLibrary
}

// SetChartLibrary sets the chart library preference.
func (s *ConfigService) SetChartLibrary(lib string) {
	s.mu.Lock()
	s.chartLibrary = lib
	s.mu.Unlock()
	s.saveConfig()
}

// GetTheme returns the current theme ("light" or "dark").
func (s *ConfigService) GetTheme() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.theme
}

// SetTheme sets the application theme.
func (s *ConfigService) SetTheme(theme string) {
	s.mu.Lock()
	s.theme = theme
	s.mu.Unlock()
	s.saveConfig()
}

// GetLogLevel returns the current log level.
func (s *ConfigService) GetLogLevel() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.logLevel
}

// SetLogLevel sets the application log level.
func (s *ConfigService) SetLogLevel(level string) {
	s.mu.Lock()
	s.logLevel = level
	s.mu.Unlock()

	logging.SetLevel(level)
	s.saveConfig()
}

// OpenLogFile opens the current log file in the OS default text editor.
func (s *ConfigService) OpenLogFile() error {
	s.mu.RLock()
	path := filepath.Clean(s.logPath)
	s.mu.RUnlock()

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("log file does not exist: %s", path)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// Use rundll32 to open the file with its default association
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default: // linux and others
		cmd = exec.Command("xdg-open", path)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}
	return nil
}
