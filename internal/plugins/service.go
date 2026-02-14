// Package plugins provides the plugin service for frontend communication.
package plugins

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"olicanaplot/internal/appconfig"
	"olicanaplot/internal/logging"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Service provides methods for the frontend to interact with plugins.
type Service struct {
	manager *Manager
	config  *appconfig.ConfigService
	app     interface{}    // Application context for plugins
	logger  logging.Logger // Structured logger
}

// NewService creates a new plugin service.
func NewService(manager *Manager, config *appconfig.ConfigService, logger logging.Logger) *Service {
	return &Service{
		manager: manager,
		config:  config,
		logger:  logger,
	}
}

// SetApp sets the application context for plugins.
func (s *Service) SetApp(app interface{}) {
	s.app = app
}

// ActivatePlugin switches to a plugin and calls its Initialize method.
func (s *Service) ActivatePlugin(name string, initStr string) error {
	s.logger.Info("Activating plugin", "name", name)

	// Close the current active plugin if it's an IPC plugin to ensure fresh start
	active := s.manager.GetActive()
	if active != nil {
		s.logger.Debug("Closing current active plugin before switch", "name", active.Name())
		active.Close()
	}

	if err := s.manager.SetActive(name); err != nil {
		s.logger.Error("Failed to set active plugin", "name", name, "error", err)
		return err
	}

	plugin := s.manager.GetActive()
	if plugin == nil {
		err := fmt.Errorf("plugin not found: %s", name)
		s.logger.Error("Plugin not found", "name", name)
		return err
	}

	// Create a plugin-specific logger
	pluginLogger := logging.NewLogger(name)

	// Call Initialize with the app context and logger
	_, err := plugin.Initialize(s.app, initStr, pluginLogger)
	if err != nil {
		s.logger.Warn("Plugin initialization returned error", "name", name, "error", err)
	}
	return err
}

// PluginMetadata contains basic information about a plugin.
type PluginMetadata struct {
	Name         string        `json:"name"`
	Path         string        `json:"path"`
	FilePatterns []FilePattern `json:"patterns"`
	IsInternal   bool          `json:"is_internal"`
	Enabled      bool          `json:"enabled"`
}

// OpenFileResult contains the result of a file open operation.
type OpenFileResult struct {
	Path       string   `json:"path"`
	Candidates []string `json:"candidates"`
}

// ListPlugins returns metadata for all registered plugins.
func (s *Service) ListPlugins() []PluginMetadata {
	return s.manager.ListMetadata()
}

// GetActivePlugin returns the name of the currently active plugin.
func (s *Service) GetActivePlugin() string {
	return s.manager.ActiveName()
}

// SetPluginEnabled enables or disables a plugin.
func (s *Service) SetPluginEnabled(name string, enabled bool) error {
	s.logger.Info("Setting plugin enabled status", "name", name, "enabled", enabled)
	if err := s.manager.SetEnabled(name, enabled); err != nil {
		return err
	}

	// Persist state
	metadata := s.manager.ListMetadata()
	disabled := []string{}
	for _, m := range metadata {
		if !m.Enabled {
			disabled = append(disabled, m.Name)
		}
	}
	s.config.SetDisabledPlugins(disabled)

	// Notify frontend
	if app, ok := s.app.(*application.App); ok {
		app.Event.Emit("pluginsChanged")
	}

	return nil
}

// LogSeriesAdded logs when a new series is added (e.g., from the frontend).
func (s *Service) LogSeriesAdded(name string, points int) {
	s.logger.Info("Series added", "name", name, "points", points)
}

// LogDebug logs a debug message from the frontend.
func (s *Service) LogDebug(component string, message string, details string) {
	s.logger.Debug(message, "component", component, "details", details)
}

// ShowInExplorer opens the file explorer with the specified path selected or opened.
func (s *Service) ShowInExplorer(path string) error {
	s.logger.Info("Showing in explorer", "path", path)
	if path == "" {
		return fmt.Errorf("empty path")
	}

	app, ok := s.app.(*application.App)
	if !ok {
		return fmt.Errorf("invalid application context")
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// /select,path opens explorer and selects the file
		cmd = exec.Command("explorer", "/select,", filepath.Clean(path))
	case "darwin":
		cmd = exec.Command("open", "-R", path)
	default: // linux
		cmd = exec.Command("xdg-open", filepath.Dir(path))
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start explorer: %w", err)
	}

	_ = app // Keeping reference just in case
	return nil
}

// GetFilePatterns returns all file patterns supported by all plugins.
func (s *Service) GetFilePatterns() []FilePatternWithPlugin {
	return s.manager.GetAllFilePatterns()
}

// OpenFile opens a file dialog and returns candidates for the selected file.
func (s *Service) OpenFile() (*OpenFileResult, error) {
	s.logger.Info("Opening file dialog")
	app, ok := s.app.(*application.App)
	if !ok {
		return nil, fmt.Errorf("invalid application context")
	}

	patterns := s.GetFilePatterns()
	dialog := app.Dialog.OpenFile().SetTitle("Load Data File")

	// Map to track which plugins belong to which extension
	extMap := make(map[string][]string)

	// Group patterns by description to collapse duplicates in the UI
	groupedPatterns := make(map[string]map[string]bool)

	for _, fp := range patterns {
		if groupedPatterns[fp.Description] == nil {
			groupedPatterns[fp.Description] = make(map[string]bool)
		}
		for _, p := range fp.Patterns {
			groupedPatterns[fp.Description][p] = true

			// Map extension to plugin
			ext := strings.ToLower(filepath.Ext(p))
			if ext != "" {
				// Avoid duplicates in the candidate list
				exists := false
				for _, name := range extMap[ext] {
					if name == fp.PluginName {
						exists = true
						break
					}
				}
				if !exists {
					extMap[ext] = append(extMap[ext], fp.PluginName)
				}
			}
		}
	}

	// First, add "All Supported Files" if we have multiple, so it's the default
	if len(patterns) > 1 {
		allPatternsMap := make(map[string]bool)
		for _, fp := range patterns {
			for _, p := range fp.Patterns {
				allPatternsMap[p] = true
			}
		}
		var allPatterns []string
		for p := range allPatternsMap {
			allPatterns = append(allPatterns, p)
		}
		dialog.AddFilter("All Supported Files", strings.Join(allPatterns, ";"))
	}

	// Add grouped filters to the dialog
	for desc, patsMap := range groupedPatterns {
		var pats []string
		for p := range patsMap {
			pats = append(pats, p)
		}
		dialog.AddFilter(desc, strings.Join(pats, ";"))
	}

	dialog.AddFilter("All Files", "*.*")

	path, err := dialog.PromptForSingleSelection()
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}

	s.logger.Info("File selected for loading", "path", path)

	// Determine matching plugins
	ext := strings.ToLower(filepath.Ext(path))
	candidates := extMap[ext]

	return &OpenFileResult{
		Path:       path,
		Candidates: candidates,
	}, nil
}

// GetChartConfig returns the chart configuration for the active plugin.
func (s *Service) GetChartConfig() (*ChartConfig, error) {
	active := s.manager.GetActive()
	if active == nil {
		return nil, fmt.Errorf("no active plugin")
	}
	return active.GetChartConfig("")
}
