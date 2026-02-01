// Package plugins provides the plugin service for frontend communication.
package plugins

import (
	"fmt"
	"path/filepath"
	"strings"

	"olicanaplot/internal/logging"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Service provides methods for the frontend to interact with plugins.
type Service struct {
	manager *Manager
	app     interface{}    // Application context for plugins
	logger  logging.Logger // Structured logger
}

// NewService creates a new plugin service.
func NewService(manager *Manager, logger logging.Logger) *Service {
	return &Service{
		manager: manager,
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
	FilePatterns []FilePattern `json:"patterns"`
}

// ListPlugins returns metadata for all registered plugins.
func (s *Service) ListPlugins() []PluginMetadata {
	return s.manager.ListMetadata()
}

// GetActivePlugin returns the name of the currently active plugin.
func (s *Service) GetActivePlugin() string {
	return s.manager.ActiveName()
}

// LogSeriesAdded logs when a new series is added (e.g., from the frontend).
func (s *Service) LogSeriesAdded(name string, points int) {
	s.logger.Info("Series added", "name", name, "points", points)
}

// LogDebug logs a debug message from the frontend.
func (s *Service) LogDebug(component string, message string, details string) {
	s.logger.Debug(message, "component", component, "details", details)
}

// GetFilePatterns returns all file patterns supported by all plugins.
func (s *Service) GetFilePatterns() []FilePatternWithPlugin {
	return s.manager.GetAllFilePatterns()
}

// OpenFile opens a file dialog with filters from all plugins and activates the appropriate one.
func (s *Service) OpenFile() error {
	s.logger.Info("Opening file dialog")
	app, ok := s.app.(*application.App)
	if !ok {
		return fmt.Errorf("invalid application context")
	}

	patterns := s.GetFilePatterns()
	dialog := app.Dialog.OpenFile().SetTitle("Load Data File")

	// Map to track which extensions belong to which plugin
	extMap := make(map[string]string)

	for _, fp := range patterns {
		dialog.AddFilter(fp.Description, strings.Join(fp.Patterns, ";"))
		for _, p := range fp.Patterns {
			// Extract extension (e.g., *.csv -> .csv)
			ext := filepath.Ext(p)
			if ext != "" {
				extMap[strings.ToLower(ext)] = fp.PluginName
			}
		}
	}

	// Add all supported files filter if we have multiple
	if len(patterns) > 1 {
		var allPatterns []string
		for _, fp := range patterns {
			allPatterns = append(allPatterns, fp.Patterns...)
		}
		dialog.AddFilter("All Supported Files", strings.Join(allPatterns, ";"))
	}
	dialog.AddFilter("All Files", "*.*")

	path, err := dialog.PromptForSingleSelection()
	if err != nil {
		return err
	}
	if path == "" {
		return nil
	}

	s.logger.Info("File selected for loading", "path", path)

	// Determine matching plugin
	ext := strings.ToLower(filepath.Ext(path))
	pluginName, ok := extMap[ext]
	if !ok {
		// Fallback to CSV if unknown but selected anyway?
		// Or show error.
		return fmt.Errorf("no plugin found to handle file extension: %s", ext)
	}

	s.logger.Info("Routing file to plugin", "plugin", pluginName, "ext", ext)
	return s.ActivatePlugin(pluginName, path)
}
