// Package plugins provides the plugin manager for OlicanaPlot.
package plugins

import (
	"fmt"
	"sync"

	"olicanaplot/internal/logging"
)

// Manager handles registration and lookup of plugins.
type Manager struct {
	mu           sync.RWMutex
	plugins      map[string]Plugin
	activePlugin string         // Currently active plugin name
	logger       logging.Logger // Structured logger
}

// NewManager creates a new plugin manager.
func NewManager(logger logging.Logger) *Manager {
	return &Manager{
		plugins: make(map[string]Plugin),
		logger:  logger,
	}
}

// Register adds a plugin to the manager.
// Returns an error if a plugin with the same name already exists.
func (m *Manager) Register(p Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := p.Name()
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin already registered: %s", name)
	}

	// Verify API version compatibility
	if p.Version() != PluginAPIVersion {
		return fmt.Errorf("plugin %s has incompatible API version: got %d, want %d",
			name, p.Version(), PluginAPIVersion)
	}

	m.plugins[name] = p
	m.logger.Info("Registered plugin", "name", name, "version", p.Version())

	// Set as active if it's the first plugin
	if m.activePlugin == "" {
		m.activePlugin = name
	}

	return nil
}

// Get returns a plugin by name, or nil if not found.
func (m *Manager) Get(name string) Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.plugins[name]
}

// GetActive returns the currently active plugin.
func (m *Manager) GetActive() Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.plugins[m.activePlugin]
}

// SetActive sets the active plugin by name.
func (m *Manager) SetActive(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.plugins[name]; !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}
	m.activePlugin = name
	return nil
}

// ActiveName returns the name of the active plugin.
func (m *Manager) ActiveName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activePlugin
}

// ListMetadata returns metadata for all registered plugins.
func (m *Manager) ListMetadata() []PluginMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]PluginMetadata, 0, len(m.plugins))
	for name, p := range m.plugins {
		result = append(result, PluginMetadata{
			Name:         name,
			FilePatterns: p.GetFilePatterns(),
		})
	}
	return result
}

// List returns the names of all registered plugins.
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.plugins))
	for name := range m.plugins {
		names = append(names, name)
	}
	return names
}

// FilePatternWithPlugin extends FilePattern with the plugin name.
type FilePatternWithPlugin struct {
	FilePattern
	PluginName string `json:"plugin"`
}

// GetAllFilePatterns returns all file patterns supported by all plugins.
func (m *Manager) GetAllFilePatterns() []FilePatternWithPlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allPatterns []FilePatternWithPlugin
	for name, p := range m.plugins {
		patterns := p.GetFilePatterns()
		for _, fp := range patterns {
			allPatterns = append(allPatterns, FilePatternWithPlugin{
				FilePattern: fp,
				PluginName:  name,
			})
		}
	}
	return allPatterns
}

// Close shuts down all plugins.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for name, p := range m.plugins {
		if err := p.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("error closing plugin %s: %w", name, err)
		}
	}
	return firstErr
}
