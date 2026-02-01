package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"olicanaplot/pkg/ipcplugin"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	pluginName    = "Template IPC"
	pluginVersion = 1
)

var (
	app        *application.App
	mainWindow application.Window
)

func main() {
	// Check for --file-patterns flag
	for _, arg := range os.Args {
		if arg == "--file-patterns" {
			fmt.Println("[]")
			os.Exit(0)
		}
	}

	app = application.New(application.Options{
		Name: pluginName,
	})

	// Run IPC handler in a goroutine
	go handleIPC()

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run application: %v\n", err)
		os.Exit(1)
	}
}

func handleIPC() {
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			app.Quit()
			return
		}

		var req ipcplugin.Request
		if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &req); err != nil {
			ipcplugin.SendError("failed to parse request")
			continue
		}

		switch req.Method {
		case "info":
			ipcplugin.SendResponse(ipcplugin.Response{
				Name:    pluginName,
				Version: pluginVersion,
			})

		case "initialize":
			// Show configuration UI here
			if mainWindow == nil {
				mainWindow = app.Window.NewWithOptions(application.WebviewWindowOptions{
					Title:     pluginName,
					Width:     400,
					Height:    300,
					Frameless: true,
				})
			}
			mainWindow.Show()
			ipcplugin.SendResponse(ipcplugin.Response{Result: "initialized"})

		case "get_chart_config":
			ipcplugin.SendResponse(ipcplugin.Response{
				Result: ipcplugin.ChartConfig{
					Title:      "Template Data",
					AxisLabels: []string{"Time", "Value"},
				},
			})

		case "get_series_config":
			ipcplugin.SendResponse(ipcplugin.Response{
				Result: []ipcplugin.SeriesConfig{
					{ID: "series1", Name: "Series 1", Color: ipcplugin.ChartColors[0]},
				},
			})

		case "get_series_data":
			data := []float64{0, 0, 1, 1, 2, 0, 3, 1}
			ipcplugin.SendBinaryData(data)

		default:
			ipcplugin.SendError("unknown method: " + req.Method)
		}
	}
}
