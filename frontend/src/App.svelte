<script>
  import * as echarts from "echarts";
  import { onMount, onDestroy } from "svelte";
  import { Events } from "@wailsio/runtime";
  import * as PluginService from "../bindings/olicanaplot/internal/plugins/service";
  import ContextMenu from "./lib/ContextMenu.svelte";
  import MeasurementResult from "./lib/MeasurementResult.svelte";
  import OptionsDialog from "./lib/OptionsDialog.svelte";

  let chartContainer;
  let chartInstance;
  let resizeObserver;
  let loading = $state(true);
  let error = $state(null);
  let dataSource = $state("sine");
  let isDarkMode = $state(false);

  // Context Menu State
  let menuVisible = $state(false);
  let menuX = $state(0);
  let menuY = $state(0);
  let menuItems = $state([]);

  // Measurement State
  let measurementStart = $state(null);
  let measurementResult = $state(null);
  let optionsVisible = $state(false);

  // Store current data to restore chart on theme change
  let currentSeriesData = null;
  let currentTitle = "";
  let currentMode = "single"; // "single" or "multi"
  let allPlugins = $state([]);

  // Activate a plugin generically
  async function activatePlugin(pluginName, sourceLabel) {
    loading = true;
    try {
      // Call the plugin service to activate the plugin
      // The plugin will spawn its own dialog if needed and block until finished
      await PluginService.ActivatePlugin(pluginName);

      // Load data from the now-active and configured plugin
      await loadData(sourceLabel || pluginName);
    } catch (e) {
      console.error("Failed to activate plugin:", e);
      error = e.message;
    }
    loading = false;
  }

  // Open unified file loader
  async function loadFile() {
    loading = true;
    try {
      await PluginService.OpenFile();
      // After OpenFile returns, the plugin has been activated and data loaded
      // We might need to refresh data manually if ActivatePlugin doesn't trigger a frontend reload
      // But in this app, activatePlugin helper does data loading.
      // Let's check how to refresh after OpenFile.
      // Actually, OpenFile calls ActivatePlugin, but we need to call loadData in frontend.

      // We can't easily know which plugin was activated without checking state
      const activePlugin = await PluginService.GetActivePlugin();
      await loadData(activePlugin);
    } catch (e) {
      console.error("Failed to load file:", e);
      if (e.message !== "cancelled") {
        error = e.message;
      }
    }
    loading = false;
  }

  // Show menu for generator plugins
  function showGenerateMenu(event) {
    event.stopPropagation();
    const generators = allPlugins.filter(
      (p) =>
        (!p.patterns || p.patterns.length === 0) &&
        !p.name.includes("Template"),
    );

    // Sort to keep "Sine Wave" at top or use a specific order if desired
    generators.sort((a, b) => {
      if (a.name === "Sine Wave") return -1;
      if (b.name === "Sine Wave") return 1;
      return a.name.localeCompare(b.name);
    });

    menuX = event.clientX;
    menuY = event.clientY;
    menuItems = generators.map((p) => ({
      label: p.name,
      action: () => activatePlugin(p.name),
    }));
    menuVisible = true;
  }

  // Toggle chart theme
  function toggleTheme() {
    isDarkMode = !isDarkMode;
    initChart().then(() => {
      if (currentSeriesData) {
        if (currentMode === "single") {
          updateChartSingleSeries(currentSeriesData, currentTitle);
        } else {
          updateChartMultiSeries(currentSeriesData, currentTitle);
        }
      }
    });
  }

  // Calculate grid right margin based on series names longuest length
  function getGridRight(seriesData) {
    const names = Array.isArray(seriesData)
      ? seriesData.map((s) => s.name)
      : [seriesData.name];
    const maxLen = Math.max(...names.map((n) => (n || "").length), 0);
    // Rough estimate: icon(25px) + gap(10px) + text(len * 8px) + padding
    return Math.max(120, maxLen * 8 + 60);
  }

  // Load data using unified API
  async function loadData(source) {
    loading = true;
    try {
      // Fetch series configuration
      const seriesResponse = await fetch("/api/series_config");
      const seriesConfig = await seriesResponse.json();

      // Fetch data for each series in parallel
      const dataPromises = seriesConfig.map((series) =>
        fetch(`/api/series_data?series=${series.id}`)
          .then((res) => res.arrayBuffer())
          .then((buffer) => ({
            ...series,
            data: new Float64Array(buffer),
          })),
      );

      const seriesData = await Promise.all(dataPromises);

      if (seriesData.length === 1) {
        dataSource = source;
        currentSeriesData = seriesData[0];
        currentTitle = `${source.charAt(0).toUpperCase() + source.slice(1)} Data`;
        currentMode = "single";
        updateChartSingleSeries(currentSeriesData, currentTitle);
      } else {
        dataSource = `${source} (${seriesData.length} series)`;
        currentSeriesData = seriesData;
        currentTitle = `${source.charAt(0).toUpperCase() + source.slice(1)} Data`;
        currentMode = "multi";
        updateChartMultiSeries(currentSeriesData, currentTitle);
      }
    } catch (e) {
      console.error("Failed to fetch data:", e);
      error = e.message;
    }
    loading = false;
  }

  // Update chart with single series
  function updateChartSingleSeries(seriesInfo, title) {
    if (!chartInstance) return;

    const textColor = isDarkMode ? "#ccc" : "#333";
    const bgColor = isDarkMode ? "#2b2b2b" : "#ffffff";

    const option = {
      backgroundColor: bgColor,
      animation: false,
      title: {
        text: title,
        left: "center",
        textStyle: { color: textColor },
      },
      tooltip: { trigger: "axis" },
      toolbox: {
        feature: {
          dataZoom: {},
          restore: {},
          saveAsImage: {},
        },
        right: 20,
        iconStyle: { borderColor: textColor },
      },
      dataZoom: [
        {
          type: "inside",
          xAxisIndex: [0],
          filterMode: "none",
        },
        {
          type: "inside",
          yAxisIndex: [0],
          filterMode: "none",
        },
      ],
      legend: {
        data: [seriesInfo.name],
        orient: "vertical",
        right: 10,
        top: 60,
        textStyle: { color: textColor },
        type: "scroll",
        triggerEvent: true,
      },
      dataset: {
        source: seriesInfo.data,
        dimensions: ["x", "y"],
      },
      xAxis: {
        type: "value",
        name: "Time",
        nameLocation: "center",
        nameGap: 40,
        nameTextStyle: { color: textColor, fontWeight: "bold", fontSize: 16 },
        axisLine: { lineStyle: { color: textColor } },
        splitLine: { lineStyle: { color: isDarkMode ? "#444" : "#e0e0e0" } },
      },
      yAxis: {
        type: "value",
        name: "Value",
        nameLocation: "center",
        nameGap: 55,
        nameRotate: 90,
        nameTextStyle: { color: textColor, fontWeight: "bold", fontSize: 16 },
        axisLine: { lineStyle: { color: textColor } },
        splitLine: { lineStyle: { color: isDarkMode ? "#444" : "#e0e0e0" } },
      },
      series: [
        {
          name: seriesInfo.name,
          type: "line",
          showSymbol: false,
          encode: { x: "x", y: "y" },
          large: true,
          emphasis: { disabled: true },
          color: seriesInfo.color,
          lineStyle: {
            width: 2,
          },
          sampling: "lttb",
        },
      ],
      grid: {
        containLabel: true,
        top: 60,
        bottom: 70,
        left: 80,
        right: getGridRight(seriesInfo),
      },
    };

    chartInstance.setOption(option, { notMerge: true });
  }

  // Update chart with multi series
  function updateChartMultiSeries(seriesDataArray, title) {
    if (!chartInstance) return;

    const textColor = isDarkMode ? "#ccc" : "#333";
    const bgColor = isDarkMode ? "#2b2b2b" : "#ffffff";

    const datasets = seriesDataArray.map((s) => ({
      source: s.data,
      dimensions: ["x", "y"],
    }));

    const series = seriesDataArray.map((s, i) => ({
      name: s.name,
      type: "line",
      showSymbol: false,
      datasetIndex: i,
      encode: { x: "x", y: "y" },
      large: true,
      emphasis: { disabled: true },
      color: s.color,
      lineStyle: { width: 2 },
      sampling: "lttb",
    }));

    const option = {
      backgroundColor: bgColor,
      animation: false,
      title: {
        text: title,
        left: "center",
        textStyle: { color: textColor },
      },
      tooltip: { trigger: "axis" },
      toolbox: {
        feature: {
          dataZoom: {},
          restore: {},
          saveAsImage: {},
        },
        right: 20,
        iconStyle: { borderColor: textColor },
      },
      dataZoom: [
        {
          type: "inside",
          xAxisIndex: [0],
          filterMode: "none",
        },
        {
          type: "inside",
          yAxisIndex: [0],
          filterMode: "none",
        },
      ],
      legend: {
        data: seriesDataArray.map((s) => s.name),
        orient: "vertical",
        right: 10,
        top: 60,
        textStyle: { color: textColor },
        type: "scroll",
        triggerEvent: true,
      },
      dataset: datasets,
      xAxis: {
        type: "value",
        name: "Time",
        nameLocation: "center",
        nameGap: 40,
        nameTextStyle: { color: textColor, fontWeight: "bold", fontSize: 16 },
        axisLine: { lineStyle: { color: textColor } },
        splitLine: { lineStyle: { color: isDarkMode ? "#444" : "#e0e0e0" } },
      },
      yAxis: {
        type: "value",
        name: "Value",
        nameLocation: "center",
        nameGap: 55,
        nameRotate: 90,
        nameTextStyle: { color: textColor, fontWeight: "bold", fontSize: 16 },
        axisLine: { lineStyle: { color: textColor } },
        splitLine: { lineStyle: { color: isDarkMode ? "#444" : "#e0e0e0" } },
      },
      series: series,
      grid: {
        containLabel: true,
        top: 60,
        bottom: 70,
        left: 80,
        right: getGridRight(seriesDataArray),
      },
    };

    chartInstance.setOption(option, { notMerge: true });
  }

  // --- Context Menu & Measurement Logic ---

  function getNearestPoint(pixelPtr) {
    if (!chartInstance || !currentSeriesData) return null;

    const [px, py] = pixelPtr;
    const dataCoord = chartInstance.convertFromPixel("grid", [px, py]);
    const targetX = dataCoord[0];

    // Check all visible series
    let closestPoint = null;
    let minDistanceSq = Infinity;
    const SNAP_RADIUS = 20; // pixels

    const seriesToSearch =
      currentMode === "single" ? [currentSeriesData] : currentSeriesData;

    for (const series of seriesToSearch) {
      if (!series.data || series.data.length === 0) continue;

      // Binary search for closest X (assuming sorted)
      let low = 0;
      let high = series.data.length / 2 - 1;
      let closestIdx = -1;

      while (low <= high) {
        let mid = Math.floor((low + high) / 2);
        let xVal = series.data[mid * 2];

        if (
          closestIdx === -1 ||
          Math.abs(xVal - targetX) <
            Math.abs(series.data[closestIdx * 2] - targetX)
        ) {
          closestIdx = mid;
        }

        if (xVal < targetX) low = mid + 1;
        else if (xVal > targetX) high = mid - 1;
        else break;
      }

      if (closestIdx !== -1) {
        const x = series.data[closestIdx * 2];
        const y = series.data[closestIdx * 2 + 1];
        const pointPixel = chartInstance.convertToPixel("grid", [x, y]);

        const dx = pointPixel[0] - px;
        const dy = pointPixel[1] - py;
        const distSq = dx * dx + dy * dy;

        if (distSq < SNAP_RADIUS * SNAP_RADIUS && distSq < minDistanceSq) {
          minDistanceSq = distSq;
          closestPoint = { x, y };
        }
      }
    }

    return closestPoint;
  }

  function handleContextMenu(e) {
    const event = e.event;
    if (!event || !event.preventDefault) return;

    // Prevent default browser menu
    event.preventDefault();
    event.stopPropagation();

    menuX = event.clientX;
    menuY = event.clientY;
    menuItems = [];

    const offX = e.offsetX;
    const offY = e.offsetY;

    // Try to detect legend item by inspecting the zrender target
    let target = e.target;
    let legendIndex = -1;

    // Traverse up to find the group containing the legend item info
    while (target) {
      if (target.__legendDataIndex !== undefined) {
        legendIndex = target.__legendDataIndex;
        break;
      }
      target = target.parent;
    }

    if (legendIndex !== -1) {
      const option = chartInstance.getOption();
      const legendData = option.legend[0].data;
      const item = legendData[legendIndex];
      const componentName = typeof item === "string" ? item : item.name;

      PluginService.LogDebug(
        "ContextMenu",
        "Legend item detected via zrender",
        componentName,
      );

      menuItems.push({
        label: `Rename "${componentName}"`,
        action: () => renameSeries(componentName),
      });
      menuItems.push({
        label: `Differentiate "${componentName}"`,
        action: () => differentiateSeries(componentName),
      });
    } else {
      // Check if we are in the grid area for measurement
      if (chartInstance.containPixel("grid", [offX, offY])) {
        if (measurementStart === null) {
          menuItems.push({
            label: "Start Measurement Here",
            action: () => {
              const snap = getNearestPoint([offX, offY]);
              measurementStart = snap || {
                x: chartInstance.convertFromPixel("grid", [offX, offY])[0],
                y: chartInstance.convertFromPixel("grid", [offX, offY])[1],
              };
            },
          });
        } else {
          menuItems.push({
            label: "End Measurement Here",
            action: () => {
              const snap = getNearestPoint([offX, offY]);
              const end = snap || {
                x: chartInstance.convertFromPixel("grid", [offX, offY])[0],
                y: chartInstance.convertFromPixel("grid", [offX, offY])[1],
              };

              measurementResult = {
                dx: end.x - measurementStart.x,
                dy: end.y - measurementStart.y,
              };
              measurementStart = null;
            },
          });
          menuItems.push({
            label: "Cancel Measurement",
            action: () => {
              measurementStart = null;
            },
          });
        }
      }
    }

    // Final visibility sync
    menuVisible = menuItems.length > 0;
  }

  function renameSeries(oldName) {
    const newName = prompt(`Enter new name for series "${oldName}":`, oldName);
    if (!newName || newName === oldName) return;

    if (currentMode === "single") {
      if (currentSeriesData.name === oldName) {
        currentSeriesData.name = newName;
      }
    } else {
      const series = currentSeriesData.find((s) => s.name === oldName);
      if (series) {
        series.name = newName;
      }
    }

    if (currentMode === "single") {
      updateChartSingleSeries(currentSeriesData, currentTitle);
    } else {
      updateChartMultiSeries(currentSeriesData, currentTitle);
    }
  }

  function differentiateSeries(seriesName) {
    // Find the source series
    let sourceSeries;
    if (currentMode === "single") {
      if (currentSeriesData.name === seriesName) {
        sourceSeries = currentSeriesData;
      }
    } else {
      sourceSeries = currentSeriesData.find((s) => s.name === seriesName);
    }

    if (!sourceSeries || !sourceSeries.data || sourceSeries.data.length < 4) {
      console.error("Cannot differentiate: series not found or too few points");
      return;
    }

    // Compute discrete derivative: dy/dx = (y[i+1] - y[i]) / (x[i+1] - x[i])
    const sourceData = sourceSeries.data;
    const numPoints = sourceData.length / 2;
    const derivData = new Float64Array((numPoints - 1) * 2);

    for (let i = 0; i < numPoints - 1; i++) {
      const x0 = sourceData[i * 2];
      const y0 = sourceData[i * 2 + 1];
      const x1 = sourceData[(i + 1) * 2];
      const y1 = sourceData[(i + 1) * 2 + 1];

      const dx = x1 - x0;
      const dy = y1 - y0;
      const derivative = dx !== 0 ? dy / dx : 0;

      // Use midpoint for x-coordinate of derivative
      derivData[i * 2] = (x0 + x1) / 2;
      derivData[i * 2 + 1] = derivative;
    }

    // Create new series
    const newSeriesName = `d(${seriesName})/dt`;
    const colorIndex = currentMode === "single" ? 1 : currentSeriesData.length;
    const colors = [
      "#636EFA",
      "#EF553B",
      "#00CC96",
      "#AB63FA",
      "#FFA15A",
      "#19D3F3",
      "#FF6692",
      "#B6E880",
      "#FF97FF",
      "#FECB52",
    ];
    const newSeries = {
      id: `deriv_${Date.now()}`,
      name: newSeriesName,
      color: colors[colorIndex % colors.length],
      data: derivData,
    };

    // Log the added series
    PluginService.LogSeriesAdded(newSeriesName, numPoints - 1);

    // Add to current data
    if (currentMode === "single") {
      // Convert to multi-series mode
      currentSeriesData = [currentSeriesData, newSeries];
      currentMode = "multi";
    } else {
      currentSeriesData.push(newSeries);
    }

    // Refresh chart
    updateChartMultiSeries(currentSeriesData, currentTitle);
  }

  async function initChart() {
    if (chartInstance) {
      chartInstance.dispose();
    }
    if (!chartContainer) return;

    // Use built-in 'dark' theme if requested
    const theme = isDarkMode ? "dark" : null;
    chartInstance = echarts.init(chartContainer, theme);

    // Use ZR contextmenu for all hit detection
    chartInstance.getZr().on("contextmenu", handleContextMenu);

    // Initial load handled by calling loadData directly if not restored
    if (!currentSeriesData) {
      await loadData("sine");
    }

    // Load available plugins
    try {
      allPlugins = await PluginService.ListPlugins();
      console.log("Loaded plugins:", allPlugins);
    } catch (e) {
      console.error("Failed to list plugins:", e);
    }
  }

  onMount(() => {
    if (chartContainer) {
      initChart();

      resizeObserver = new ResizeObserver(() => {
        chartInstance?.resize();
      });
      resizeObserver.observe(chartContainer);
    }
  });

  onDestroy(() => {
    resizeObserver?.disconnect();
    chartInstance?.dispose();
  });
</script>

<div class="app-container" class:dark-mode={isDarkMode}>
  <header class="main-header">
    <div
      class="logo"
      onclick={() => activatePlugin("Sine Wave", "Sine")}
      role="button"
      tabindex="0"
      onkeydown={(e) =>
        e.key === "Enter" && activatePlugin("Sine Wave", "Sine")}
      style="cursor: pointer;"
    >
      <svg
        viewBox="0 0 24 24"
        width="24"
        height="24"
        stroke="currentColor"
        stroke-width="2"
        fill="none"><path d="M3 3v18h18" /><path d="M18 9l-5 5-2-2-4 4" /></svg
      >
      <span>OlicanaPlot</span>
    </div>
    <!-- Hidden file input -->

    <nav class="menu-bar">
      <button onclick={toggleTheme} title="Toggle Dark Mode">
        {#if isDarkMode}
          <!-- Sun icon -->
          <svg
            viewBox="0 0 24 24"
            width="16"
            height="16"
            stroke="currentColor"
            stroke-width="2"
            fill="none"
            stroke-linecap="round"
            stroke-linejoin="round"
            ><circle cx="12" cy="12" r="5"></circle><line
              x1="12"
              y1="1"
              x2="12"
              y2="3"
            ></line><line x1="12" y1="21" x2="12" y2="23"></line><line
              x1="4.22"
              y1="4.22"
              x2="5.64"
              y2="5.64"
            ></line><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"
            ></line><line x1="1" y1="12" x2="3" y2="12"></line><line
              x1="21"
              y1="12"
              x2="23"
              y2="12"
            ></line><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line><line
              x1="18.36"
              y1="5.64"
              x2="19.78"
              y2="4.22"
            ></line></svg
          >
        {:else}
          <!-- Moon icon -->
          <svg
            viewBox="0 0 24 24"
            width="16"
            height="16"
            stroke="currentColor"
            stroke-width="2"
            fill="none"
            stroke-linecap="round"
            stroke-linejoin="round"
            ><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"
            ></path></svg
          >
        {/if}
      </button>
      <button onclick={loadFile}>
        <svg
          viewBox="0 0 24 24"
          width="16"
          height="16"
          stroke="currentColor"
          stroke-width="2"
          fill="none"
          ><path
            d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"
          /><polyline points="13 2 13 9 20 9" /></svg
        >
        Load File
      </button>
      <button onclick={(e) => showGenerateMenu(e)}>
        <svg
          viewBox="0 0 24 24"
          width="16"
          height="16"
          stroke="currentColor"
          stroke-width="2"
          fill="none"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12" /></svg
        >
        Generate
      </button>
      <button
        onclick={() => (optionsVisible = true)}
        title="Application Options"
      >
        <svg
          viewBox="0 0 24 24"
          width="16"
          height="16"
          stroke="currentColor"
          stroke-width="2"
          fill="none"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <circle cx="12" cy="12" r="3"></circle>
          <path
            d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"
          ></path>
        </svg>
        Options
      </button>
    </nav>
  </header>

  <main class="content-area">
    <div class="chart-container" bind:this={chartContainer}></div>
  </main>

  <footer class="status-bar">
    <span>{loading ? "Loading..." : "Ready"}</span>
    <span>Data: {dataSource}</span>
  </footer>

  <ContextMenu
    x={menuX}
    y={menuY}
    visible={menuVisible}
    items={menuItems}
    onClose={() => {
      menuVisible = false;
    }}
  />

  <MeasurementResult
    visible={measurementResult !== null}
    deltaX={measurementResult?.dx || 0}
    deltaY={measurementResult?.dy || 0}
    onClose={() => {
      measurementResult = null;
    }}
  />

  <OptionsDialog
    visible={optionsVisible}
    onClose={() => {
      optionsVisible = false;
    }}
  />
</div>

<style>
  :global(html),
  :global(body),
  :global(#app) {
    margin: 0;
    padding: 0;
    width: 100%;
    height: 100%;
    overflow: hidden;
    background-color: #f2f2f2;
    color: #2a3f5f;
    font-family:
      "Open Sans",
      -apple-system,
      BlinkMacSystemFont,
      "Segoe UI",
      Roboto,
      Helvetica,
      Arial,
      sans-serif;
  }

  .app-container {
    height: 100vh;
    display: flex;
    flex-direction: column;
  }

  .main-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 20px;
    height: 50px;
    background-color: #ffffff;
    border-bottom: 1px solid #d8d8d8;
    flex-shrink: 0;
  }

  .logo {
    display: flex;
    align-items: center;
    gap: 10px;
    font-weight: 700;
    font-size: 1.1em;
    color: #2a3f5f;
  }

  .menu-bar {
    display: flex;
    gap: 10px;
  }

  .menu-bar button {
    background: transparent;
    border: 1px solid transparent;
    color: #506784;
    padding: 6px 12px;
    border-radius: 6px;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 0.9em;
    transition: all 0.2s;
  }

  .menu-bar button:hover {
    background: #e2e2e2;
    color: #2a3f5f;
    border-color: #d8d8d8;
  }

  .content-area {
    flex: 1;
    background-color: #ffffff;
    padding: 20px;
    overflow: hidden;
    position: relative;
    min-height: 0; /* Important for flex shrinking */
  }

  .chart-container {
    width: 100%;
    height: 100%;
  }

  .status-bar {
    height: 24px;
    background-color: #ffffff;
    border-top: 1px solid #d8d8d8;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 15px;
    font-size: 0.75em;
    color: #506784;
    flex-shrink: 0;
  }

  /* Dark Mode Styles */
  .app-container.dark-mode {
    background-color: #1a1a1a;
    color: #e0e0e0;
  }

  .app-container.dark-mode .main-header,
  .app-container.dark-mode .status-bar {
    background-color: #2d2d2d;
    border-color: #444;
    color: #e0e0e0;
  }

  .app-container.dark-mode .logo {
    color: #e0e0e0;
  }

  .app-container.dark-mode .content-area {
    background-color: #1a1a1a; /* Match chart dark bg */
  }

  .app-container.dark-mode .menu-bar button {
    color: #a0a0a0;
  }

  .app-container.dark-mode .menu-bar button:hover {
    background-color: #3d3d3d;
    color: #ffffff;
    border-color: #555;
  }
</style>
