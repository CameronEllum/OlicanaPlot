package appconfig

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ConfigService handles application configuration and settings.
type ConfigService struct {
	logPath      string
	chartLibrary string
}

// NewConfigService creates a new config service with default values.
func NewConfigService() *ConfigService {
	// Default log path: %APPDATA%/OlicanaPlot/logs/olicana.log
	configDir, err := os.UserConfigDir()
	var defaultLogPath string
	if err == nil {
		defaultLogPath = filepath.Join(configDir, "OlicanaPlot", "olicana.log")
	} else {
		defaultLogPath = "olicana.log"
	}

	return &ConfigService{
		logPath:      defaultLogPath,
		chartLibrary: "echarts", // Default to ECharts
	}
}

// GetLogPath returns the current log path.
func (s *ConfigService) GetLogPath() string {
	return s.logPath
}

// SetLogPath updates the log path.
func (s *ConfigService) SetLogPath(path string) {
	s.logPath = path
}

// GetChartLibrary returns the current chart library ("echarts" or "plotly").
func (s *ConfigService) GetChartLibrary() string {
	return s.chartLibrary
}

// SetChartLibrary sets the chart library preference.
func (s *ConfigService) SetChartLibrary(lib string) {
	s.chartLibrary = lib
}

// OpenLogFile opens the current log file in the OS default text editor.
func (s *ConfigService) OpenLogFile() error {
	path := filepath.Clean(s.logPath)

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
