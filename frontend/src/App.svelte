<script>
  import { onMount, onDestroy } from "svelte";
  import { Events } from "@wailsio/runtime";
  import * as PluginService from "../bindings/olicanaplot/internal/plugins/service";
  import * as ConfigService from "../bindings/olicanaplot/internal/appconfig/configservice";
  import ContextMenu from "./lib/ContextMenu.svelte";
  import MeasurementResult from "./lib/MeasurementResult.svelte";
  import OptionsDialog from "./lib/OptionsDialog.svelte";
  import { EChartsAdapter } from "./lib/chart/EChartsAdapter.js";
  import { PlotlyAdapter } from "./lib/chart/PlotlyAdapter.js";

  let chartContainer;
  let chartAdapter = null;
  let chartLibrary = $state("echarts");
  let resizeObserver;
  let loading = $state(true);
  let error = $state(null);
  let dataSource = $state("sine");
  let isDarkMode = $state(false);

  $effect(() => {
    if (isDarkMode) {
      document.documentElement.classList.add("dark-mode");
    } else {
      document.documentElement.classList.remove("dark-mode");
    }
  });

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
  let currentSeriesData = $state([]);
  let currentTitle = "";
  let allPlugins = $state([]);

  // Plugin Selection State (for ambiguous file matches)
  let pluginSelectionVisible = $state(false);
  let pluginSelectionCandidates = $state([]);
  let pendingFilePath = $state("");

  // Activate a plugin generically
  async function activatePlugin(pluginName, initStr = "", sourceLabel = "") {
    loading = true;
    try {
      // Call the plugin service to activate the plugin
      // The plugin will spawn its own dialog if needed and block until finished
      await PluginService.ActivatePlugin(pluginName, initStr);

      // Load data from the now-active and configured plugin
      await loadData(sourceLabel || pluginName);
    } catch (e) {
      console.error("Failed to activate plugin:", e);
      error = e.message;
    }
    loading = false;
  }

  // Add data to existing chart from a plugin (asSubplots = CTRL held)
  async function addDataToChart(pluginName, initStr = "", asSubplots = false) {
    loading = true;
    try {
      await PluginService.ActivatePlugin(pluginName, initStr);

      // Fetch the new series config and data
      const seriesResponse = await fetch("/api/series_config");
      const seriesConfig = await seriesResponse.json();
      const storage = chartLibrary === "plotly" ? "arrays" : "interleaved";

      const dataPromises = seriesConfig.map(async (series) => {
        const res = await fetch(
          `/api/series_data?series=${series.id}&storage=${storage}`,
        );
        const buffer = await res.arrayBuffer();
        const data = new Float64Array(buffer);
        return { ...series, data };
      });

      const newSeriesData = await Promise.all(dataPromises);

      if (asSubplots) {
        // Add as subplots - assign a new subplot index
        const maxSubplot = Math.max(
          0,
          ...currentSeriesData.map((s) => s.subplotIndex || 0),
        );
        const nextSubplotIndex = maxSubplot + 1;
        console.log(`Assigning new subplot index: ${nextSubplotIndex}`);
        newSeriesData.forEach((s) => {
          s.subplotIndex = nextSubplotIndex;
          s.id = `subplot_${Date.now()}_${s.id}`;
        });
      } else {
        // Add as new series - assign new colors from the color palette
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
        newSeriesData.forEach((s, i) => {
          const colorIndex =
            currentSeriesData.reduce(
              (count, ser) =>
                (ser.subplotIndex || 0) === 0 ? count + 1 : count,
              0,
            ) + i;
          s.color = colors[colorIndex % colors.length];
          s.id = `added_${Date.now()}_${s.id}`;
          s.subplotIndex = 0; // Explicitly part of the main subplot
        });
      }

      // Append to existing data
      currentSeriesData = [...currentSeriesData, ...newSeriesData];

      const subplotLabel = asSubplots
        ? "Subplot " +
          Math.max(0, ...currentSeriesData.map((s) => s.subplotIndex || 0))
        : "Series";
      dataSource = `${dataSource} + [${subplotLabel}] ${pluginName}`;

      if (asSubplots) {
        currentTitle = `Multi-Subplot Analysis`;
      } else {
        currentTitle = `${currentTitle} + ${pluginName}`;
      }

      updateChart();

      PluginService.LogDebug(
        "AddData",
        `Added ${newSeriesData.length} series (asSubplots=${asSubplots})`,
        "",
      );
    } catch (e) {
      console.error("Failed to add data:", e);
      error = e.message;
    }
    loading = false;
  }

  // Open unified file loader
  async function loadFile() {
    loading = true;
    try {
      const result = await PluginService.OpenFile();
      if (!result) {
        loading = false;
        return;
      }

      const { path, candidates } = result;

      if (candidates && candidates.length === 1) {
        await activatePlugin(candidates[0], path);
      } else if (candidates && candidates.length > 1) {
        pluginSelectionCandidates = candidates;
        pendingFilePath = path;
        pluginSelectionVisible = true;
      } else {
        // No matching plugin found (maybe selected via "All Files")
        // Try a generic CSV fallback or show error
        error = "No specific plugin found to handle this file extension.";
      }
    } catch (e) {
      console.error("Failed to load file:", e);
      if (e.message !== "cancelled") {
        error = e.message;
      }
    }
    loading = false;
  }

  // Track pending add mode for plugin selection
  let pendingAddMode = $state(false);
  let pendingAsSubplots = $state(false);

  // Add file to existing chart (CTRL = subplots, otherwise series)
  async function addFile(event) {
    const asSubplots = event.ctrlKey;
    loading = true;
    try {
      const result = await PluginService.OpenFile();
      if (!result) {
        loading = false;
        return;
      }

      const { path, candidates } = result;

      if (candidates && candidates.length === 1) {
        await addDataToChart(candidates[0], path, asSubplots);
      } else if (candidates && candidates.length > 1) {
        pluginSelectionCandidates = candidates;
        pendingFilePath = path;
        pendingAddMode = true;
        pendingAsSubplots = asSubplots;
        pluginSelectionVisible = true;
      } else {
        error = "No specific plugin found to handle this file extension.";
      }
    } catch (e) {
      console.error("Failed to add file:", e);
      if (e.message !== "cancelled") {
        error = e.message;
      }
    }
    loading = false;
  }

  async function handlePluginSelection(pluginName) {
    pluginSelectionVisible = false;
    if (pendingAddMode) {
      await addDataToChart(pluginName, pendingFilePath, pendingAsSubplots);
      pendingAddMode = false;
      pendingAsSubplots = false;
    } else {
      await activatePlugin(pluginName, pendingFilePath);
    }
    pluginSelectionCandidates = [];
    pendingFilePath = "";
  }

  // Show menu for generator plugins (CTRL = add to existing chart)
  function showGenerateMenu(event) {
    event.stopPropagation();
    const isAddMode = event.ctrlKey;
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
    menuItems = generators.map((p) => {
      const modeLabel = isAddMode ? "Add as Subplot" : "Replace with";
      return {
        label: `${modeLabel} ${p.name}`,
        action: isAddMode
          ? () => {
              console.log(`Adding subplot data from ${p.name}`);
              addDataToChart(p.name, "", true); // Set asSubplots to true when CTRL is held
            }
          : () => {
              console.log(`Activating plugin ${p.name}`);
              activatePlugin(p.name, "", "");
            },
      };
    });
    menuVisible = true;
  }

  // Toggle chart theme
  async function toggleTheme() {
    isDarkMode = !isDarkMode;
    const newTheme = isDarkMode ? "dark" : "light";
    await ConfigService.SetTheme(newTheme);

    initChart().then(() => {
      if (currentSeriesData && currentSeriesData.length > 0) {
        updateChart();
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

  async function loadData(source) {
    loading = true;
    try {
      // Fetch series configuration
      const seriesResponse = await fetch("/api/series_config");
      const seriesConfig = await seriesResponse.json();

      // Determine requested storage format
      const storage = chartLibrary === "plotly" ? "arrays" : "interleaved";

      // Fetch data for each series in parallel
      const dataPromises = seriesConfig.map(async (series) => {
        const res = await fetch(
          `/api/series_data?series=${series.id}&storage=${storage}`,
        );
        const buffer = await res.arrayBuffer();
        const data = new Float64Array(buffer);

        return {
          ...series,
          data: data,
        };
      });

      const seriesData = await Promise.all(dataPromises);

      // Initialize subplotIndex for all base series
      seriesData.forEach((s) => (s.subplotIndex = 0));

      currentSeriesData = seriesData;
      currentTitle = `${source.charAt(0).toUpperCase() + source.slice(1)} Data`;
      dataSource = source;
      updateChart();
    } catch (e) {
      console.error("Failed to fetch data:", e);
      error = e.message;
    }
    loading = false;
  }

  // Unified chart update function using adapter
  function updateChart() {
    if (!chartAdapter || !currentSeriesData) return;
    chartAdapter.setData(
      currentSeriesData,
      currentTitle,
      isDarkMode,
      getGridRight,
    );
  }

  // --- Context Menu & Measurement Logic ---

  function getNearestPoint(pixelPtr) {
    if (!chartAdapter || !currentSeriesData) return null;

    const [px, py] = pixelPtr;
    const dataCoord = chartAdapter.getDataAtPixel(px, py);
    if (!dataCoord) return null;

    const targetX = dataCoord.x;

    // Check all visible series
    let closestPoint = null;
    let minDistanceSq = Infinity;
    const SNAP_RADIUS = 20; // pixels

    const seriesToSearch = currentSeriesData;

    for (const series of seriesToSearch) {
      if (!series.data || series.data.length === 0) continue;

      // Binary search for closest X (assuming sorted)
      let low = 0;
      let high = series.data.length / 2 - 1;
      let closestIdx = -1;

      const isArrays = chartLibrary === "plotly";
      const numPoints = series.data.length / 2;

      while (low <= high) {
        let mid = Math.floor((low + high) / 2);
        let xVal = isArrays ? series.data[mid] : series.data[mid * 2];

        if (
          closestIdx === -1 ||
          Math.abs(xVal - targetX) <
            Math.abs(
              (isArrays
                ? series.data[closestIdx]
                : series.data[closestIdx * 2]) - targetX,
            )
        ) {
          closestIdx = mid;
        }

        if (xVal < targetX) low = mid + 1;
        else if (xVal > targetX) high = mid - 1;
        else break;
      }

      if (closestIdx !== -1) {
        const x = isArrays
          ? series.data[closestIdx]
          : series.data[closestIdx * 2];
        const y = isArrays
          ? series.data[numPoints + closestIdx]
          : series.data[closestIdx * 2 + 1];
        const pointPixel = chartAdapter.getPixelFromData(x, y);

        if (pointPixel) {
          const dx = pointPixel.x - px;
          const dy = pointPixel.y - py;
          const distSq = dx * dx + dy * dy;

          if (distSq < SNAP_RADIUS * SNAP_RADIUS && distSq < minDistanceSq) {
            minDistanceSq = distSq;
            closestPoint = { x, y };
          }
        }
      }
    }

    return closestPoint;
  }

  function handleContextMenu(e) {
    // Handle both ECharts zrender events and native DOM events
    const event = e.event || e;
    if (!event || !event.preventDefault) return;

    // Prevent default browser menu
    event.preventDefault();
    event.stopPropagation();

    menuX = event.clientX;
    menuY = event.clientY;
    menuItems = [];

    // Get offset relative to container
    const rect = chartContainer.getBoundingClientRect();
    const offX = event.clientX - rect.left;
    const offY = event.clientY - rect.top;

    // For ECharts adapter, try to detect legend item
    let legendIndex = -1;
    if (e.target) {
      let target = e.target;
      while (target) {
        if (target.__legendDataIndex !== undefined) {
          legendIndex = target.__legendDataIndex;
          break;
        }
        target = target.parent;
      }
    }

    if (e.plotlyLegendName || (legendIndex !== -1 && chartAdapter.instance)) {
      let componentName;
      if (e.plotlyLegendName) {
        componentName = e.plotlyLegendName;
      } else {
        // ECharts-specific legend handling
        const option = chartAdapter.instance.getOption();
        const legendData = option?.legend?.[0]?.data || [];
        const item = legendData[legendIndex];
        componentName = typeof item === "string" ? item : item?.name;
      }

      if (componentName) {
        PluginService.LogDebug(
          "ContextMenu",
          "Legend item detected",
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
      }
    } else {
      // Check if we are in the grid area for measurement
      const dataPoint = chartAdapter.getDataAtPixel(offX, offY);
      if (dataPoint) {
        if (measurementStart === null) {
          menuItems.push({
            label: "Start Measurement Here",
            action: () => {
              const snap = getNearestPoint([offX, offY]);
              measurementStart = snap || dataPoint;
            },
          });
        } else {
          menuItems.push({
            label: "End Measurement Here",
            action: () => {
              const snap = getNearestPoint([offX, offY]);
              const end = snap || dataPoint;

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

    updateChart();
  }

  function differentiateSeries(seriesName) {
    // Find the source series
    let sourceSeries;
    sourceSeries = currentSeriesData.find((s) => s.name === seriesName);

    if (!sourceSeries || !sourceSeries.data || sourceSeries.data.length < 4) {
      console.error("Cannot differentiate: series not found or too few points");
      return;
    }

    // Compute discrete derivative: dy/dx = (y[i+1] - y[i]) / (x[i+1] - x[i])
    // Data in memory always matches current engine because it's re-fetched or cleared on engine change
    const engineStorage = chartLibrary === "plotly" ? "arrays" : "interleaved";
    const currentStorage = engineStorage;
    const sourceData = sourceSeries.data;
    const numPoints = sourceData.length / 2;
    const derivData = new Float64Array((numPoints - 1) * 2);

    const isArrays = engineStorage === "arrays";
    for (let i = 0; i < numPoints - 1; i++) {
      let x0, y0, x1, y1;
      if (currentStorage === "arrays") {
        x0 = sourceData[i];
        y0 = sourceData[numPoints + i];
        x1 = sourceData[i + 1];
        y1 = sourceData[numPoints + i + 1];
      } else {
        x0 = sourceData[i * 2];
        y0 = sourceData[i * 2 + 1];
        x1 = sourceData[(i + 1) * 2];
        y1 = sourceData[(i + 1) * 2 + 1];
      }

      const dx = x1 - x0;
      const dy = y1 - y0;
      const derivative = dx !== 0 ? dy / dx : 0;
      const xMid = (x0 + x1) / 2;

      if (isArrays) {
        derivData[i] = xMid;
        derivData[numPoints - 1 + i] = derivative;
      } else {
        derivData[i * 2] = xMid;
        derivData[i * 2 + 1] = derivative;
      }
    }

    const finalDerivData = derivData;

    // Create new series
    const newSeriesName = `d(${seriesName})/dt`;
    const colorIndex = currentSeriesData.length;
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
      data: finalDerivData,
    };

    // Log the added series
    PluginService.LogSeriesAdded(newSeriesName, numPoints - 1);

    // Add to current data
    currentSeriesData.push(newSeries);

    // Refresh chart
    updateChart();
  }

  async function initChart() {
    // Destroy existing adapter
    if (chartAdapter) {
      chartAdapter.destroy();
      chartAdapter = null;
    }
    if (!chartContainer) return;

    // Load chart library preference
    try {
      chartLibrary = await ConfigService.GetChartLibrary();
    } catch (e) {
      console.warn("Failed to get chart library preference:", e);
      chartLibrary = "echarts";
    }

    // Create appropriate adapter
    if (chartLibrary === "plotly") {
      chartAdapter = new PlotlyAdapter();
    } else {
      chartAdapter = new EChartsAdapter();
    }

    chartAdapter.init(chartContainer, isDarkMode);
    chartAdapter.onContextMenu(handleContextMenu);

    // Initial load handled by calling loadData directly if not restored
    if (currentSeriesData.length === 0) {
      await loadData("sine");
    } else {
      // Restore existing data
      updateChart();
    }

    // Load available plugins
    try {
      allPlugins = await PluginService.ListPlugins();
      console.log("Loaded plugins:", allPlugins);
    } catch (e) {
      console.error("Failed to list plugins:", e);
    }
  }

  // Handle chart library change from options dialog
  async function handleChartLibraryChange(newLibrary) {
    if (chartLibrary !== newLibrary) {
      if (currentSeriesData.length > 0) {
        const confirmed = confirm(
          "Changing the chart engine will reset the current plot. Any added subplots or series will be lost. Continue?",
        );
        if (!confirmed) {
          // Note: The config is already saved in ConfigService by the dialog.
          // We could revert it here, but a simple reset is usually the intended behavior for this constraint.
          return;
        }
      }

      chartLibrary = newLibrary;
      currentSeriesData = []; // Reset plot data
      dataSource = "none";

      await initChart();
    }
  }

  onMount(() => {
    if (chartContainer) {
      initChart();

      resizeObserver = new ResizeObserver(() => {
        chartAdapter?.resize();
      });
      resizeObserver.observe(chartContainer);
    }

    // Load initial theme
    ConfigService.GetTheme().then((theme) => {
      isDarkMode = theme === "dark";
    });

    // Listen for chart library changes
    Events.On("chartLibraryChanged", handleChartLibraryChange);
  });

  onDestroy(() => {
    resizeObserver?.disconnect();
    chartAdapter?.destroy();
    Events.Off("chartLibraryChanged", handleChartLibraryChange);
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
      <button
        onclick={(e) => addFile(e)}
        title="Add data to current chart (Ctrl = add as subplots)"
      >
        <svg
          viewBox="0 0 24 24"
          width="16"
          height="16"
          stroke="currentColor"
          stroke-width="2"
          fill="none"
          ><path
            d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"
          /><polyline points="13 2 13 9 20 9" /><line
            x1="12"
            y1="18"
            x2="12"
            y2="12"
          /><line x1="9" y1="15" x2="15" y2="15" /></svg
        >
        Add File
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

  <!-- Plugin Selection Modal -->
  {#if pluginSelectionVisible}
    <div
      class="modal-backdrop"
      onclick={() => (pluginSelectionVisible = false)}
      onkeydown={(e) => {
        if (e.key === "Escape" || e.key === "Enter")
          pluginSelectionVisible = false;
      }}
      role="button"
      tabindex="-1"
      aria-label="Close selection modal"
    >
      <div
        class="modal-content"
        onclick={(e) => e.stopPropagation()}
        onkeydown={(e) => e.stopPropagation()}
        role="dialog"
        tabindex="-1"
      >
        <h3 class="text-gradient">Select Plugin</h3>
        <p>
          Multiple plugins can handle this file. Which one would you like to
          use?
        </p>
        <div class="candidate-list">
          {#each pluginSelectionCandidates as plugin}
            <button
              class="candidate-item"
              onclick={() => handlePluginSelection(plugin)}
            >
              <span class="plugin-name">{plugin}</span>
              <svg
                viewBox="0 0 24 24"
                width="16"
                height="16"
                stroke="currentColor"
                stroke-width="2"
                fill="none"
              >
                <path d="M5 12h14M12 5l7 7-7 7" />
              </svg>
            </button>
          {/each}
        </div>
        <div class="modal-footer">
          <button
            class="btn btn-secondary"
            onclick={() => (pluginSelectionVisible = false)}>Cancel</button
          >
        </div>
      </div>
    </div>
  {/if}
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
    background-color: var(--bg-primary);
    color: var(--text-primary);
    font-family: "Inter", sans-serif;
  }

  .app-container {
    height: 100vh;
    display: flex;
    flex-direction: column;
    background-color: var(--bg-primary);
  }

  .main-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 24px;
    height: 60px;
    background: var(--bg-glass);
    border-bottom: 1px solid var(--border-color);
    backdrop-filter: blur(10px);
    z-index: 100;
  }

  .logo {
    display: flex;
    align-items: center;
    gap: 12px;
    font-weight: 800;
    font-size: 1.2rem;
    letter-spacing: -0.02em;
    background: var(--header-gradient);
    -webkit-background-clip: text;
    background-clip: text;
    -webkit-text-fill-color: transparent;
  }

  .menu-bar {
    display: flex;
    gap: 8px;
  }

  .menu-bar button {
    background: rgba(0, 0, 0, 0.03);
    border: 1px solid var(--border-color);
    color: var(--text-secondary);
    padding: 8px 16px;
    border-radius: 10px;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 0.9rem;
    font-weight: 500;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  }

  :global(.dark-mode) .menu-bar button {
    background: rgba(255, 255, 255, 0.05);
  }

  .menu-bar button:hover {
    background: var(--accent);
    color: white;
    border-color: var(--accent);
    transform: translateY(-1px);
    box-shadow: 0 4px 12px var(--accent-glow);
  }

  .content-area {
    flex: 1;
    background-color: var(--bg-primary);
    padding: 0;
    overflow: hidden;
    position: relative;
    min-height: 0;
  }

  .chart-container {
    width: 100%;
    height: 100%;
  }

  .status-bar {
    height: 32px;
    background: var(--bg-secondary);
    border-top: 1px solid var(--border-color);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 20px;
    font-size: 0.8rem;
    color: var(--text-secondary);
    font-weight: 500;
  }

  /* Candidate List Styles */
  .candidate-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 24px;
  }

  .candidate-item {
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid var(--border-color);
    border-radius: 12px;
    padding: 16px;
    color: var(--text-primary);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: space-between;
    transition: all 0.2s;
    text-align: left;
    width: 100%;
    font-family: inherit;
  }

  .candidate-item:hover {
    background: rgba(99, 110, 250, 0.15);
    border-color: var(--accent);
    transform: translateX(4px);
  }

  .candidate-item .plugin-name {
    font-weight: 600;
  }
</style>
