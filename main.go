package main

import (
	"embed"
	_ "embed"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"olicanaplot/internal/appconfig"
	"olicanaplot/internal/data"
	"olicanaplot/internal/logging"
	"olicanaplot/internal/plugins"
	"olicanaplot/internal/plugins/csv_reader"
	"olicanaplot/internal/plugins/function_generator"
	"olicanaplot/internal/plugins/ipc"
	"olicanaplot/internal/plugins/process_model_generator"
	"olicanaplot/internal/plugins/sine_generator"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	// Register a custom event whose associated data type is string.
	// This is not required, but the binding generator will pick up registered events
	// and provide a strongly typed JS/TS API for them.
	application.RegisterEvent[string]("time")
}

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {
	// Create config service first to get log path
	configService := appconfig.NewConfigService()

	// Clear/Initialize log file
	logPath := configService.GetLogPath()
	os.MkdirAll(filepath.Dir(logPath), 0755)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err == nil {
		// Log to both file and stdout (SafeMultiWriter handles closed stdout in GUI)
		multiWriter := &logging.SafeMultiWriter{Writers: []io.Writer{os.Stdout, logFile}}
		logging.SetOutput(multiWriter)

		// Redirect standard Go logs to our structured logger
		log.SetFlags(0)
		log.SetOutput(logging.NewRedirector(logging.NewLogger("System")))

		defer logFile.Close()
	}

	// Create root logger
	logger := logging.NewLogger("OlicanaPlot")
	logger.Info("Starting OlicanaPlot")

	// Create plugin manager and register plugins
	pluginManager := plugins.NewManager(logger)

	if err := pluginManager.Register(sine_generator.New(), true); err != nil {
		logger.Warn("Failed to register sine plugin", "error", err)
	}
	if err := pluginManager.Register(function_generator.New(configService), true); err != nil {
		logger.Warn("Failed to register funcplot plugin", "error", err)
	}
	if err := pluginManager.Register(process_model_generator.New(), true); err != nil {
		logger.Warn("Failed to register process model plugin", "error", err)
	}
	if err := pluginManager.Register(csv_reader.New(), true); err != nil {
		logger.Warn("Failed to register CSV plugin", "error", err)
	}

	// Load IPC plugins from both built-in and user-configured directories
	builtInDir, _ := filepath.Abs("plugins")
	searchDirs := append([]string{builtInDir}, configService.GetPluginSearchDirs()...)
	loader := ipc.NewLoader(searchDirs, logger)
	ipcPlugins, err := loader.Discover()
	if err != nil {
		logger.Warn("Failed to discover IPC plugins", "error", err)
	}
	for _, p := range ipcPlugins {
		if err := pluginManager.Register(p, false); err != nil {
			logger.Warn("Failed to register IPC plugin", "name", p.Name(), "error", err)
		}
	}

	// Create plugin service for frontend communication
	pluginService := plugins.NewService(pluginManager, configService, logger)

	// Apply saved disabled status
	disabledList := configService.GetDisabledPlugins()
	for _, name := range disabledList {
		pluginManager.SetEnabled(name, false)
	}

	// Set function plotter as the default active plugin
	pluginManager.SetActive("Function Plotter")

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Bind' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.
	app := application.New(application.Options{
		Name:        "OlicanaPlot",
		Description: "OlicanaPlot application",
		Services: []application.Service{
			application.NewService(pluginService),
			application.NewService(configService),
		},
		Assets: application.AssetOptions{
			Handler:    application.AssetFileServerFS(assets),
			Middleware: data.Middleware(pluginManager, logger),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "OlicanaPlot",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	// Provide application context to services for dialog spawning
	pluginService.SetApp(app)
	configService.SetApp(app)

	// Fetch IPC plugin file patterns in the background
	go func() {
		logger.Debug("Refreshing IPC plugin file patterns in background")
		pluginManager.GetAllFilePatterns()
		logger.Debug("IPC plugin file patterns refreshed")
	}()

	// Run the application. This blocks until the application has been exited.
	err = app.Run()

	// Clean up plugins on shutdown
	pluginManager.Close()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}
