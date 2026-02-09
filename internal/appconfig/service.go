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
	mu                 sync.RWMutex
	app                *application.App
	optionsWindow      *application.WebviewWindow
	configPath         string
	logPath            string
	chartLibrary       string
	theme              string
	logLevel           string
	disabledPlugins    []string
	showGeneratorsMenu bool
	defaultLineWidth   float64
}

// configData is the structure we save to disk
type configData struct {
	LogPath            string   `json:"logPath"`
	ChartLibrary       string   `json:"chartLibrary"`
	Theme              string   `json:"theme"`
	LogLevel           string   `json:"logLevel"`
	DisabledPlugins    []string `json:"disabledPlugins"`
	ShowGeneratorsMenu bool     `json:"showGeneratorsMenu"`
	DefaultLineWidth   float64  `json:"defaultLineWidth"`
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
		configPath:         filepath.Join(appDir, "config.json"),
		logPath:            filepath.Join(appDir, "olicana.log"),
		chartLibrary:       "echarts", // Default to ECharts
		theme:              "light",   // Default to light
		logLevel:           "info",    // Default to info
		showGeneratorsMenu: true,      // Default to true
		defaultLineWidth:   2.0,       // Default to 2.0
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
	s.disabledPlugins = cfg.DisabledPlugins
	s.showGeneratorsMenu = cfg.ShowGeneratorsMenu
	if cfg.DefaultLineWidth > 0 {
		s.defaultLineWidth = cfg.DefaultLineWidth
	} else {
		s.defaultLineWidth = 2.0
	}

	// Apply log level
	logging.SetLevel(s.logLevel)
}

func (s *ConfigService) saveConfig() {
	s.mu.RLock()
	cfg := configData{
		LogPath:            s.logPath,
		ChartLibrary:       s.chartLibrary,
		Theme:              s.theme,
		LogLevel:           s.logLevel,
		DisabledPlugins:    s.disabledPlugins,
		ShowGeneratorsMenu: s.showGeneratorsMenu,
		DefaultLineWidth:   s.defaultLineWidth,
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
	app := s.app
	s.mu.Unlock()
	s.saveConfig()

	if app != nil {
		app.Event.Emit("chartLibraryChanged", lib)
	}
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
	app := s.app
	s.mu.Unlock()
	s.saveConfig()

	if app != nil {
		app.Event.Emit("themeChanged", theme)
	}
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

// GetDisabledPlugins returns the list of disabled plugin names.
func (s *ConfigService) GetDisabledPlugins() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.disabledPlugins
}

// SetDisabledPlugins updates the list of disabled plugins.
func (s *ConfigService) SetDisabledPlugins(plugins []string) {
	s.mu.Lock()
	s.disabledPlugins = plugins
	s.mu.Unlock()
	s.saveConfig()
}

// GetShowGeneratorsMenu returns if the generators menu should be shown.
func (s *ConfigService) GetShowGeneratorsMenu() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.showGeneratorsMenu
}

// SetShowGeneratorsMenu updates the show generators menu setting.
func (s *ConfigService) SetShowGeneratorsMenu(show bool) {
	s.mu.Lock()
	s.showGeneratorsMenu = show
	app := s.app
	s.mu.Unlock()
	s.saveConfig()

	if app != nil {
		app.Event.Emit("showGeneratorsMenuChanged", show)
	}
}

// GetDefaultLineWidth returns the default line width for charts.
func (s *ConfigService) GetDefaultLineWidth() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.defaultLineWidth
}

// SetDefaultLineWidth updates the default line width and notifies listeners.
func (s *ConfigService) SetDefaultLineWidth(width float64) {
	s.mu.Lock()
	s.defaultLineWidth = width
	app := s.app
	s.mu.Unlock()
	s.saveConfig()

	if app != nil {
		app.Event.Emit("defaultLineWidthChanged", width)
	}
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

// OpenURL opens the specified URL in the system browser.
func (s *ConfigService) OpenURL(url string) {
	s.mu.RLock()
	app := s.app
	s.mu.RUnlock()

	if app != nil {
		app.Browser.OpenURL(url)
	}
}
