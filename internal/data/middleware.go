// Package data provides an HTTP middleware for efficient binary data transfer.
package data

import (
	"encoding/json"
	"fmt"
	"net/http"
	"unsafe"

	"olicanaplot/internal/logging"
	"olicanaplot/internal/plugins"
)

// Middleware creates an HTTP middleware that intercepts chart data API requests.
func Middleware(manager *plugins.Manager, logger logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/chart_config":
				handleChartConfig(w, r, manager)
				return

			case "/api/series_config":
				handleSeriesConfig(w, r, manager)
				return

			case "/api/series_data":
				handleSeriesData(w, r, manager, logger)
				return

			case "/api/plugins":
				handlePluginList(w, r, manager)
				return
			}

			// Pass to default asset server
			next.ServeHTTP(w, r)
		})
	}
}

// handleChartConfig handles GET/POST for chart configuration
func handleChartConfig(w http.ResponseWriter, r *http.Request, manager *plugins.Manager) {
	if r.Method == "POST" {
		r.ParseForm()

		// Switch active plugin if requested
		if pluginName := r.FormValue("plugin"); pluginName != "" {
			if err := manager.SetActive(pluginName); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
		return
	}

	// GET - return current plugin config
	plugin := manager.GetActive()
	if plugin == nil {
		http.Error(w, "No active plugin", http.StatusNotFound)
		return
	}

	config, err := plugin.GetChartConfig("")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"activePlugin": manager.ActiveName(),
		"plugins":      manager.List(),
		"title":        config.Title,
		"axisLabels":   config.AxisLabels,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSeriesConfig returns the list of series from the active plugin
func handleSeriesConfig(w http.ResponseWriter, r *http.Request, manager *plugins.Manager) {
	plugin := manager.GetActive()
	if plugin == nil {
		http.Error(w, "No active plugin", http.StatusNotFound)
		return
	}

	series, err := plugin.GetSeriesConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(series)
}

// handleSeriesData returns binary Float64 data for a specific series
func handleSeriesData(w http.ResponseWriter, r *http.Request, manager *plugins.Manager, logger logging.Logger) {
	seriesID := r.URL.Query().Get("series")
	if seriesID == "" {
		http.Error(w, "Missing series parameter", http.StatusBadRequest)
		return
	}

	storage := r.URL.Query().Get("storage") // interleaved or arrays

	plugin := manager.GetActive()
	if plugin == nil {
		http.Error(w, "No active plugin", http.StatusNotFound)
		return
	}

	data, actualStorage, err := plugin.GetSeriesData(seriesID, storage)
	if err != nil {
		logger.Error("Error getting series data", "series", seriesID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set actual storage header so frontend knows if it needs to convert
	w.Header().Set("X-Data-Storage", actualStorage)

	numPoints := len(data) / 2
	logger.Info("Serving series data", "series", seriesID, "points", numPoints)

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)*8))

	// Create a byte slice view of the float64 data without copying
	if len(data) > 0 {
		byteData := unsafe.Slice((*byte)(unsafe.Pointer(&data[0])), len(data)*8)
		w.Write(byteData)
	}
}

// handlePluginList returns the list of available plugins
func handlePluginList(w http.ResponseWriter, r *http.Request, manager *plugins.Manager) {
	response := map[string]interface{}{
		"active":  manager.ActiveName(),
		"plugins": manager.List(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
