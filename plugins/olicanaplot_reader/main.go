package main

import (
	"bufio"
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"

	sdk "olicanaplot/sdk/go"

	"gopkg.in/yaml.v3"
)

const (
	pluginName    = "OlicanaPlot Reader"
	pluginVersion = 1
)

// YAML structures
type FileConfig struct {
	Version   int              `yaml:"version"`
	Chart     ChartSection     `yaml:"chart"`
	Layout    LayoutSection    `yaml:"layout"`
	Behaviour BehaviourSection `yaml:"behaviour"`
	Axes      []AxisEntry      `yaml:"axes"`
}

type ChartSection struct {
	Title     string   `yaml:"title"`
	LineWidth *float64 `yaml:"line_width"`
}

type LayoutSection struct {
	Rows int `yaml:"rows"`
	Cols int `yaml:"cols"`
}

type BehaviourSection struct {
	LinkX *bool `yaml:"link_x"`
	LinkY *bool `yaml:"link_y"`
}

type AxisEntry struct {
	Title   string        `yaml:"title"`
	Subplot []int         `yaml:"subplot"`
	XAxes   []AxisDetail  `yaml:"x_axes"`
	YAxes   []AxisDetail  `yaml:"y_axes"`
	Series  []SeriesEntry `yaml:"series"`
}

type AxisDetail struct {
	Title    string   `yaml:"title"`
	Position string   `yaml:"position"`
	Unit     string   `yaml:"unit"`
	Type     string   `yaml:"type"`
	Min      *float64 `yaml:"min"`
	Max      *float64 `yaml:"max"`
}

type SeriesEntry struct {
	Title     string   `yaml:"title"`
	Column    int      `yaml:"column"`
	YAxis     string   `yaml:"y_axis"`
	Color     string   `yaml:"color"`
	LineType  string   `yaml:"line_type"`
	LineWidth *float64 `yaml:"line_width"`
	Visible   *bool    `yaml:"visible"`
}

type CsvBlock struct {
	Data [][]float64 // [column][row]
}

type Plugin struct {
	mu         sync.Mutex
	fileConfig *FileConfig
	csvBlocks  []CsvBlock
}

func (p *Plugin) loadFile(path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var reader io.Reader = file
	if strings.HasSuffix(strings.ToLower(path), ".olicaplotz") {
		gz, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer gz.Close()
		reader = gz
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	// Split on form feed (\f)
	parts := strings.Split(string(content), "\f")
	if len(parts) == 0 {
		return fmt.Errorf("empty file")
	}

	// Parse YAML
	var config FileConfig
	if err := yaml.Unmarshal([]byte(parts[0]), &config); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}
	p.fileConfig = &config

	// Parse CSV blocks
	p.csvBlocks = nil
	for i := 1; i < len(parts); i++ {
		block, err := p.parseCsvBlock(parts[i])
		if err != nil {
			return fmt.Errorf("failed to parse CSV block %d: %w", i-1, err)
		}
		p.csvBlocks = append(p.csvBlocks, block)
	}

	return nil
}

func (p *Plugin) parseCsvBlock(content string) (CsvBlock, error) {
	reader := csv.NewReader(strings.NewReader(strings.TrimSpace(content)))
	reader.FieldsPerRecord = -1 // Allow variable fields if needed, but we expect consistency

	records, err := reader.ReadAll()
	if err != nil {
		return CsvBlock{}, err
	}

	if len(records) == 0 {
		return CsvBlock{}, nil
	}

	rows := len(records)
	cols := len(records[0])

	data := make([][]float64, cols)
	for c := 0; c < cols; c++ {
		data[c] = make([]float64, rows)
		for r := 0; r < rows; r++ {
			valStr := ""
			if c < len(records[r]) {
				valStr = strings.TrimSpace(records[r][c])
			}

			val, err := strconv.ParseFloat(valStr, 64)
			if err != nil || valStr == "" {
				data[c][r] = math.NaN()
			} else {
				data[c][r] = val
			}
		}
	}

	return CsvBlock{Data: data}, nil
}

func main() {
	// Check for --metadata flag
	for _, arg := range os.Args {
		if arg == "--metadata" {
			meta := map[string]interface{}{
				"name": pluginName,
				"patterns": []interface{}{
					map[string]interface{}{
						"description": "OlicanaPlot Files",
						"patterns":    []string{"*.olicanaplot", "*.olicaplotz"},
					},
				},
			}
			bytes, _ := json.Marshal(meta)
			fmt.Println(string(bytes))
			os.Exit(0)
		}
	}

	p := &Plugin{}
	handleIPC(p)
}

func handleIPC(p *Plugin) {
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		var req sdk.Request
		if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &req); err != nil {
			sdk.SendError("failed to parse request")
			continue
		}

		switch req.Method {
		case "info":
			sdk.SendResponse(sdk.Response{
				Name:    pluginName,
				Version: uint32(pluginVersion),
			})

		case "initialize":
			if req.Args == "" {
				sdk.SendError("no file path provided")
				continue
			}
			if err := p.loadFile(req.Args); err != nil {
				sdk.SendError(fmt.Sprintf("failed to load file: %v", err))
				continue
			}
			sdk.SendResponse(sdk.Response{Result: "loaded"})

		case "get_chart_config":
			if p.fileConfig == nil {
				sdk.SendError("no file loaded")
				continue
			}

			axes := make([]sdk.AxisGroupConfig, len(p.fileConfig.Axes))
			for i, entry := range p.fileConfig.Axes {
				axes[i] = sdk.AxisGroupConfig{
					Title:   entry.Title,
					Subplot: entry.Subplot,
				}
				for _, x := range entry.XAxes {
					axes[i].XAxes = append(axes[i].XAxes, sdk.AxisConfig{
						Title:    x.Title,
						Position: x.Position,
						Unit:     x.Unit,
						Type:     x.Type,
						Min:      x.Min,
						Max:      x.Max,
					})
				}
				for _, y := range entry.YAxes {
					axes[i].YAxes = append(axes[i].YAxes, sdk.AxisConfig{
						Title:    y.Title,
						Position: y.Position,
						Unit:     y.Unit,
						Type:     y.Type,
						Min:      y.Min,
						Max:      y.Max,
					})
				}
			}

			config := sdk.ChartConfig{
				Title:     p.fileConfig.Chart.Title,
				LineWidth: p.fileConfig.Chart.LineWidth,
				Axes:      axes,
				LinkX:     p.fileConfig.Behaviour.LinkX,
				LinkY:     p.fileConfig.Behaviour.LinkY,
				Rows:      p.fileConfig.Layout.Rows,
				Cols:      p.fileConfig.Layout.Cols,
			}
			sdk.SendResponse(sdk.Response{Result: config})

		case "get_series_config":
			if p.fileConfig == nil {
				sdk.SendError("no file loaded")
				continue
			}

			var seriesConfigs []sdk.SeriesConfig
			for i, entry := range p.fileConfig.Axes {
				for _, s := range entry.Series {
					color := s.Color

					name := s.Title
					if name == "" {
						name = fmt.Sprintf("Axis %d Col %d", i, s.Column)
					}

					id := fmt.Sprintf("axis%d:col%d", i, s.Column)

					seriesConfigs = append(seriesConfigs, sdk.SeriesConfig{
						ID:        id,
						Name:      name,
						Color:     color,
						Subplot:   entry.Subplot,
						LineType:  s.LineType,
						LineWidth: s.LineWidth,
						Visible:   s.Visible,
						YAxis:     s.YAxis,
					})
				}
			}
			sdk.SendResponse(sdk.Response{Result: seriesConfigs})

		case "get_series_data":
			// Parse ID: axis%d:col%d
			var axisIdx, colIdx int
			_, err := fmt.Sscanf(req.SeriesID, "axis%d:col%d", &axisIdx, &colIdx)
			if err != nil {
				sdk.SendError("invalid series ID")
				continue
			}

			if axisIdx < 0 || axisIdx >= len(p.csvBlocks) {
				sdk.SendError("axis index out of range")
				continue
			}

			block := p.csvBlocks[axisIdx]
			if colIdx < 0 || colIdx >= len(block.Data) {
				sdk.SendError("column index out of range")
				continue
			}

			xData := block.Data[0]
			yData := block.Data[colIdx]

			if req.PreferredStorage == "arrays" {
				data := append(xData, yData...)
				sdk.SendBinaryData(data, "arrays")
			} else {
				// Interleaved
				data := make([]float64, 2*len(xData))
				for i := 0; i < len(xData); i++ {
					data[2*i] = xData[i]
					data[2*i+1] = yData[i]
				}
				sdk.SendBinaryData(data, "interleaved")
			}

		default:
			sdk.SendError("unknown method: " + req.Method)
		}
	}
}
