import { Events, Dialogs } from "@wailsio/runtime";
import * as PluginService from "../../../bindings/olicanaplot/internal/plugins/service";
import * as ConfigService from "../../../bindings/olicanaplot/internal/appconfig/configservice";
import type { ChartAdapter, SeriesConfig, ContextMenuEvent } from "../chart/ChartAdapter";
import { EChartsAdapter } from "../chart/EChartsAdapter";
import { PlotlyAdapter } from "../chart/PlotlyAdapter";

export interface Point {
    x: number;
    y: number;
}

export interface AppPlugin {
    name: string;
    patterns: any[];
}

class AppState {
    // Reactive State
    chartContainer = $state<HTMLElement | null>(null);
    chartAdapter = $state<ChartAdapter | null>(null);
    chartLibrary = $state<string>("echarts");
    loading = $state(true);
    error = $state<string | null>(null);
    dataSource = $state("funcplot");
    linkX = $state(true); // Default to true as it was the previous behavior
    linkY = $state(false);
    isDarkMode = $state(false);
    isDefault = $state(true);

    // Data State
    currentSeriesData = $state<SeriesConfig[]>([]);
    currentTitle = $state("");
    xAxisName = $state("Time");
    subplotNames = $state<Record<string, string>>({}); // key is "row,col"
    allPlugins = $state<AppPlugin[]>([]);
    showGeneratorsMenu = $state(true);
    defaultLineWidth = $state(2.0);

    get hasSubplots(): boolean {
        const cells = new Set(
            this.currentSeriesData.map(s => `${s.subplotRow || 0},${s.subplotCol || 0}`)
        );
        return cells.size > 1;
    }

    // UI State - Dialogs
    pluginSelectionVisible = $state(false);
    pluginSelectionCandidates = $state<string[]>([]);
    pendingFilePath = $state("");
    pendingAddMode = $state(false);
    pendingTargetCell = $state({ row: 0, col: 0 });
    addFileChoiceVisible = $state(false);
    private addFileChoiceResolver: ((val: { row: number; col: number } | null) => void) | null = null;

    renameVisible = $state(false);
    renameModalTitle = $state("");
    renameModalLabel = $state("");
    renameModalValue = $state("");
    renameModalCallback = $state<(val: string) => void>(() => { });

    // Context Menu State
    menuVisible = $state(false);
    menuX = $state(0);
    menuY = $state(0);
    menuItems = $state<{ label: string; action?: () => void; header?: boolean }[]>([]);

    // Measurement State
    measurementStart = $state<Point | null>(null);
    measurementResult = $state<{ dx: number; dy: number } | null>(null);

    // Event unsubs
    private unsubs: (() => void)[] = [];
    private initialized = false;
    private initPromise: Promise<void> | null = null;

    constructor() {
        this.initPromise = this.init();
    }

    async init() {
        PluginService.LogDebug("AppState", "init() started", "");
        try {
            this.isDarkMode = (await ConfigService.GetTheme()) === "dark";
            this.chartLibrary = await ConfigService.GetChartLibrary();
            PluginService.LogDebug("AppState", `Config loaded, library: ${this.chartLibrary}`, "");
            this.showGeneratorsMenu = await ConfigService.GetShowGeneratorsMenu();
            this.defaultLineWidth = await ConfigService.GetDefaultLineWidth();

            const plugins = await PluginService.ListPlugins();
            this.allPlugins = plugins || [];
            PluginService.LogDebug("AppState", `Plugins loaded: ${this.allPlugins.length}`, "");

            this.setupEventListeners();
        } catch (e: any) {
            PluginService.LogDebug("AppState", "init() failed", e.toString());
            console.error("Failed to initialize app state:", e);
        }
        this.initialized = true;
        PluginService.LogDebug("AppState", "init() complete", "");
        this.loading = false;
    }

    setupEventListeners() {
        this.unsubs.push(Events.On("chartLibraryChanged", (val: any) => {
            this.chartLibrary = (Array.isArray(val.data) ? val.data[0] : val.data) as string;
            // Clear current data and reset to Sine Wave as requested
            this.currentSeriesData = [];
            this.dataSource = "sine";
            this.initChart(this.chartContainer!);
        }));

        this.unsubs.push(Events.On("showGeneratorsMenuChanged", (val: any) => {
            this.showGeneratorsMenu = (Array.isArray(val.data) ? val.data[0] : val.data) as boolean;
        }));

        this.unsubs.push(Events.On("defaultLineWidthChanged", (val: any) => {
            this.defaultLineWidth = (Array.isArray(val.data) ? val.data[0] : val.data) as number;
            this.updateChart();
        }));
    }

    destroy() {
        this.unsubs.forEach(u => u());
    }

    // Chart Lifecycle
    async initChart(container: HTMLElement) {
        PluginService.LogDebug("AppState", "initChart() called", "");
        if (!this.initialized && this.initPromise) {
            await this.initPromise;
        }
        this.chartContainer = container;
        PluginService.LogDebug("AppState", "initChart() container assigned", "");
        if (this.chartAdapter) {
            PluginService.LogDebug("AppState", "initChart() destroying old adapter", "");
            this.chartAdapter.destroy();
        }

        PluginService.LogDebug("AppState", `initChart() library to use: ${this.chartLibrary}`, "");
        if (this.chartLibrary === "plotly") {
            this.chartAdapter = new PlotlyAdapter();
        } else {
            this.chartAdapter = new EChartsAdapter();
        }
        PluginService.LogDebug("AppState", "initChart() adapter instance created", "");

        this.chartAdapter.init(container, this.isDarkMode);
        PluginService.LogDebug("AppState", "initChart() adapter.init sequence basic done", "");

        PluginService.LogDebug("AppState", "initChart() binding context menu", "");
        this.chartAdapter.onContextMenu(this.handleContextMenu.bind(this));
        PluginService.LogDebug("AppState", "initChart() binding legend click", "");
        this.chartAdapter.onLegendClick(this.handleLegendClick.bind(this));
        PluginService.LogDebug("AppState", "initChart() all bindings done", "");

        PluginService.LogDebug("AppState", `initChart sequence check: dataLen=${this.currentSeriesData.length}, dataSource=${this.dataSource}`, "");
        if (this.currentSeriesData.length > 0) {
            PluginService.LogDebug("AppState", "initChart() updating existing data", "");
            this.updateChart();
        } else if (this.dataSource && (this.dataSource.toLowerCase() === "sine" || this.dataSource.toLowerCase() === "funcplot" || this.dataSource === "Function Plotter")) {
            // Restore auto-load of default plot on startup
            PluginService.LogDebug("AppState", "Auto-loading Function Plotter on startup", "");
            this.resetToDefault(true);
        } else {
            PluginService.LogDebug("AppState", `initChart() no auto-load: dataSource=${this.dataSource}`, "");
        }
    }

    // Actions
    async activatePlugin(pluginName: string, initStr = "", sourceLabel = "") {
        this.loading = true;
        try {
            await PluginService.ActivatePlugin(pluginName, initStr);
            await this.loadData(sourceLabel || pluginName);
            await this.fetchPluginConfig();
        } catch (e: any) {
            console.error("Failed to activate plugin:", e);
            this.error = e.message;
        }
        this.loading = false;
    }

    // Create the default plot
    async resetToDefault(skipConfirmation = false) {
        if (!skipConfirmation && !this.isDefault) {
            const res = await Dialogs.Question({
                Title: "Default Plot",
                Message: "Clear current plot and return to default damped sine wave?",
                Buttons: [
                    { Label: "OK", IsDefault: true },
                    { Label: "Cancel", IsCancel: true }
                ]
            });
            if (res !== "OK") return;
        }

        const defaultConfig = JSON.stringify({
            functionName: "Damped Sine",
            expression: "exp(-0.01*x) * sin(x * 0.1)",
            xMin: 0,
            xMax: 500,
            numPoints: 1000
        });

        await this.activatePlugin("Function Plotter", defaultConfig);
        this.differentiateSeries("Damped Sine");
        this.isDefault = true;
    }

    async addDataToChart(pluginName: string, initStr = "", targetCell: { row: number, col: number } = { row: 0, col: 0 }) {
        this.loading = true;
        try {
            await PluginService.ActivatePlugin(pluginName, initStr);

            const seriesResponse = await fetch("/api/series_config");
            const seriesConfig = await seriesResponse.json();
            const storage = this.chartLibrary === "plotly" ? "arrays" : "interleaved";

            const dataPromises = seriesConfig.map(async (series: any) => {
                const res = await fetch(`/api/series_data?series=${series.id}&storage=${storage}`);
                const buffer = await res.arrayBuffer();
                const data = new Float64Array(buffer);
                return { ...series, data };
            });

            const newSeriesData: SeriesConfig[] = await Promise.all(dataPromises);

            const colors = ["#636EFA", "#EF553B", "#00CC96", "#AB63FA", "#FFA15A", "#19D3F3", "#FF6692", "#B6E880", "#FF97FF", "#FECB52"];
            newSeriesData.forEach((s, i) => {
                // Determine color based on existing series in this specific cell
                const countInCell = this.currentSeriesData.filter(ser => (ser.subplotRow || 0) === targetCell.row && (ser.subplotCol || 0) === targetCell.col).length;
                s.color = colors[(countInCell + i) % colors.length];
                s.id = `added_${Date.now()}_${s.id}`;
                s.subplotRow = targetCell.row;
                s.subplotCol = targetCell.col;
            });

            this.currentSeriesData = [...this.currentSeriesData, ...newSeriesData];
            this.isDefault = false;
            await this.fetchPluginConfig(targetCell.row, targetCell.col);
            this.updateChart();
        } catch (e: any) {
            console.error("Failed to add data:", e);
            this.error = e.message;
        }
        this.loading = false;
    }

    async loadFile() {
        this.loading = true;
        try {
            const result = await PluginService.OpenFile();
            if (!result) { this.loading = false; return; }
            const { path, candidates } = result as { path: string; candidates: string[] };

            if (candidates?.length === 1) {
                await this.activatePlugin(candidates[0], path);
            } else if (candidates?.length > 1) {
                this.pluginSelectionCandidates = candidates;
                this.pendingFilePath = path;
                this.pluginSelectionVisible = true;
            } else {
                this.error = "No specific plugin found to handle this file extension.";
            }
        } catch (e: any) {
            if (e.message !== "cancelled") this.error = e.message;
        }
        this.loading = false;
    }

    async addFile(event: MouseEvent) {
        let targetCell: { row: number, col: number } | null = null;
        if (event.ctrlKey) {
            const maxRow = Math.max(0, ...this.currentSeriesData.map(s => s.subplotRow || 0));
            targetCell = { row: maxRow + 1, col: 0 };
        } else if (event.altKey) {
            targetCell = { row: 0, col: 0 };
        } else {
            targetCell = await this.showAddFileChoice();
        }

        if (targetCell === null) return;

        this.loading = true;
        try {
            const result = await PluginService.OpenFile();
            if (!result) { this.loading = false; return; }
            const { path, candidates } = result as { path: string; candidates: string[] };

            if (candidates?.length === 1) {
                await this.addDataToChart(candidates[0], path, targetCell);
            } else if (candidates?.length > 1) {
                this.pluginSelectionCandidates = candidates;
                this.pendingFilePath = path;
                this.pendingAddMode = true;
                this.pendingTargetCell = targetCell; // Add this state field below
                this.pluginSelectionVisible = true;
            } else {
                this.error = "No specific plugin found to handle this file extension.";
            }
        } catch (e: any) {
            if (e.message !== "cancelled") this.error = e.message;
        }
        this.loading = false;
    }

    private showAddFileChoice(): Promise<{ row: number, col: number } | null> {
        return new Promise((resolve) => {
            this.addFileChoiceResolver = resolve;
            this.addFileChoiceVisible = true;
        });
    }

    handleAddFileChoice(cell: { row: number, col: number } | null) {
        this.addFileChoiceVisible = false;
        if (this.addFileChoiceResolver) {
            this.addFileChoiceResolver(cell);
            this.addFileChoiceResolver = null;
        }
    }


    async handlePluginSelection(pluginName: string) {
        this.pluginSelectionVisible = false;
        if (this.pendingAddMode) {
            await this.addDataToChart(pluginName, this.pendingFilePath, this.pendingTargetCell);
            this.pendingAddMode = false;
            this.pendingTargetCell = { row: 0, col: 0 };
        } else {
            await this.activatePlugin(pluginName, this.pendingFilePath);
        }
        this.pluginSelectionCandidates = [];
        this.pendingFilePath = "";
    }

    async loadData(source: string) {
        if (!this.initialized && this.initPromise) {
            await this.initPromise;
        }
        this.loading = true;
        try {
            const seriesResponse = await fetch("/api/series_config");
            const seriesConfig = await seriesResponse.json();
            const storage = this.chartLibrary === "plotly" ? "arrays" : "interleaved";

            const dataPromises = seriesConfig.map(async (series: any) => {
                const res = await fetch(`/api/series_data?series=${series.id}&storage=${storage}`);
                const buffer = await res.arrayBuffer();
                const data = new Float64Array(buffer);
                return { ...series, data };
            });

            const seriesData: SeriesConfig[] = await Promise.all(dataPromises);
            seriesData.forEach((s) => {
                s.subplotRow = 0;
                s.subplotCol = 0;
            });

            this.currentSeriesData = seriesData;
            this.dataSource = source;
            this.isDefault = false; // Reset to false whenever any data is loaded

            await this.fetchPluginConfig();
            this.updateChart();
        } catch (e: any) {
            console.error("Failed to fetch data:", e);
            this.error = e.message;
        }
        this.loading = false;
    }

    async fetchPluginConfig(row = 0, col = 0) {
        try {
            const config = await PluginService.GetChartConfig();
            if (config) {
                if (row === 0 && col === 0 && config.title) this.currentTitle = config.title;
                if (config.axis_labels && config.axis_labels.length >= 2) {
                    if (row === 0) this.xAxisName = config.axis_labels[0];
                    // Update the subplot's Y label
                    this.subplotNames[`${row},${col}`] = config.axis_labels[1];
                }
            }
        } catch (e) {
            console.error("Failed to fetch plugin config:", e);
        }
    }


    updateChart() {
        if (!this.chartAdapter || !this.currentSeriesData) return;
        this.chartAdapter.setData(
            this.currentSeriesData,
            this.currentTitle,
            this.isDarkMode,
            this.getGridRight.bind(this),
            this.defaultLineWidth,
            this.xAxisName,
            this.subplotNames,
            this.linkX,
            this.linkY
        );
    }

    toggleLinkX() {
        this.linkX = !this.linkX;
        this.updateChart();
    }

    toggleLinkY() {
        this.linkY = !this.linkY;
        this.updateChart();
    }

    getGridRight(seriesData: SeriesConfig[]) {
        const names = Array.isArray(seriesData)
            ? seriesData.map((s) => s.name)
            : [(seriesData as any).name];
        const maxLen = Math.max(...names.map((n) => (n || "").length), 0);
        return Math.max(120, maxLen * 8 + 60);
    }

    async toggleTheme() {
        this.isDarkMode = !this.isDarkMode;
        const newTheme = this.isDarkMode ? "dark" : "light";
        await ConfigService.SetTheme(newTheme);
        if (this.chartAdapter) {
            this.chartAdapter.init(this.chartContainer!, this.isDarkMode);
            this.updateChart();
        }
    }

    // Context Menu Handlers
    handleContextMenu(e: ContextMenuEvent) {
        const rawEvent = e.rawEvent;
        if (!rawEvent) return;

        // Prevent default browser context menu
        if (typeof rawEvent.preventDefault === "function") {
            rawEvent.preventDefault();
        }
        // Stop propagation to prevent window-level listeners from closing the menu immediately
        if (typeof rawEvent.stopPropagation === "function") {
            rawEvent.stopPropagation();
        }

        this.menuX = rawEvent.clientX;
        this.menuY = rawEvent.clientY;
        this.menuItems = [];

        if (e.type === "legend" && e.seriesName) {
            this.menuItems.push({ label: e.seriesName, header: true });
            this.menuItems.push({ label: "Rename", action: () => this.renameSeries(e.seriesName!) });
            this.menuItems.push({ label: "Differentiate", action: () => this.differentiateSeries(e.seriesName!) });
        } else if (e.type === "title") {
            this.menuItems.push({ label: "Rename Plot Title", action: () => this.renameTitle() });
        } else if (e.type === "grid" && e.dataPoint) {
            const dataPoint = e.dataPoint;
            if (this.measurementStart === null) {
                this.menuItems.push({
                    label: "Start Measurement Here",
                    action: () => {
                        const rect = this.chartContainer!.getBoundingClientRect();
                        const snap = this.getNearestPoint([e.rawEvent.clientX - rect.left, e.rawEvent.clientY - rect.top]);
                        this.measurementStart = snap || dataPoint;
                    }
                });
            } else {
                this.menuItems.push({
                    label: "End Measurement Here",
                    action: () => {
                        const rect = this.chartContainer!.getBoundingClientRect();
                        const snap = this.getNearestPoint([e.rawEvent.clientX - rect.left, e.rawEvent.clientY - rect.top]);
                        const end = snap || dataPoint;
                        this.measurementResult = { dx: end.x - this.measurementStart!.x, dy: end.y - this.measurementStart!.y };
                        this.measurementStart = null;
                    }
                });
                this.menuItems.push({ label: "Cancel Measurement", action: () => { this.measurementStart = null; } });
            }
        } else if (e.type === "xAxis" || e.type === "yAxis") {
            this.menuItems.push({ label: `${e.type === "xAxis" ? "X" : "Y"} Axis: ${e.axisLabel}`, header: true });
            this.menuItems.push({
                label: "Rename",
                action: () => {
                    this.openRenameDialog(`Rename ${e.type === "xAxis" ? "X" : "Y"} Axis`, "New Name", e.axisLabel || "", (val) => {
                        if (e.type === "xAxis") this.xAxisName = val;
                        else this.subplotNames[`${e.row},${e.col}`] = val;
                        this.updateChart();
                    });
                }
            });
        } else {
            // General Plot Options for "other" or empty types
            this.menuItems.push({ label: "Plot Options", header: true });
            this.menuItems.push({ label: "Toggle Theme", action: () => this.toggleTheme() });
            if (this.currentSeriesData.length > 1) {
                this.menuItems.push({
                    label: "Clear All",
                    action: () => {
                        this.currentSeriesData = [];
                        this.dataSource = "none";
                        this.updateChart();
                    }
                });
            }
        }

        this.menuVisible = this.menuItems.length > 0;
    }

    handleLegendClick(seriesName: string, event: any) {
        if (event.ctrlKey) {
            this.renameSeries(seriesName);
        }
    }

    handleLogoContextMenu(event: MouseEvent) {
        event.preventDefault();
        event.stopPropagation();
        this.showMenu(event.clientX, event.clientY, [
            { label: "OlicanaPlot", header: true },
            { label: "Go to homepage", action: () => ConfigService.OpenURL("https://github.com/CameronEllum/OlicanaPlot") }
        ]);
    }

    showGenerateMenu(event: MouseEvent) {
        event.stopPropagation();
        const isAddMode = event.ctrlKey;
        const generators = this.allPlugins.filter(p => (!p.patterns || p.patterns.length === 0) && !p.name.includes("Template"));
        generators.sort((a, b) => a.name === "Sine Wave" ? -1 : b.name === "Sine Wave" ? 1 : a.name.localeCompare(b.name));

        this.showMenu(event.clientX, event.clientY, generators.map(p => ({
            label: `${isAddMode ? "Add" : "Replace with"} ${p.name}`,
            action: async () => {
                if (isAddMode) {
                    const targetCell = await this.showAddFileChoice();
                    if (targetCell) {
                        await this.addDataToChart(p.name, "", targetCell);
                    }
                } else {
                    await this.activatePlugin(p.name, "", "");
                }
            }
        })));
    }

    // Rename Logic
    renameSeries(oldName: string) {
        this.openRenameDialog("Rename Series", "New Name", oldName, (newName) => {
            this.currentSeriesData = this.currentSeriesData.map(s => s.name === oldName ? { ...s, name: newName } : s);
            this.updateChart();
        });
    }

    renameTitle() {
        this.openRenameDialog("Rename Plot", "New Title", this.currentTitle, (newTitle) => {
            this.currentTitle = newTitle;
            this.updateChart();
        });
    }

    differentiateSeries(seriesName: string) {
        const series = this.currentSeriesData.find(s => s.name === seriesName);
        if (!series || !series.data) return;

        const numPoints = series.data.length / 2;
        const newData = new Float64Array(series.data.length);
        const isArrays = this.chartLibrary === "plotly";

        for (let i = 0; i < numPoints; i++) {
            if (isArrays) {
                newData[i] = series.data[i];
                newData[numPoints + i] = (i === 0) ? NaN : (series.data[numPoints + i] - series.data[numPoints + i - 1]) / (series.data[i] - series.data[i - 1]);
            } else {
                newData[i * 2] = series.data[i * 2];
                newData[i * 2 + 1] = (i === 0) ? NaN : (series.data[i * 2 + 1] - series.data[(i - 1) * 2 + 1]) / (series.data[i * 2] - series.data[(i - 1) * 2]);
            }
        }

        const newSeriesName = `d(${series.name})/d(${this.xAxisName})`;
        const newSeries: SeriesConfig = {
            ...series,
            id: `diff_${Date.now()}_${series.id}`,
            name: newSeriesName,
            data: newData,
            color: "#ff0000"
        };

        this.currentSeriesData = [...this.currentSeriesData, newSeries];
        this.isDefault = false;
        this.updateChart();
    }

    getNearestPoint(pixelPtr: [number, number]): Point | null {
        if (!this.chartAdapter || !this.currentSeriesData) return null;
        return (this.chartAdapter as any).getNearestPoint?.(pixelPtr) || null; // This logic might need moving too but let's see
    }

    openRenameDialog(title: string, label: string, value: string, callback: (val: string) => void) {
        this.renameModalTitle = title;
        this.renameModalLabel = label;
        this.renameModalValue = value;
        this.renameModalCallback = callback;
        this.renameVisible = true;
    }

    showMenu(x: number, y: number, items: { label: string; action?: () => void; header?: boolean }[]) {
        this.menuX = x;
        this.menuY = y;
        this.menuItems = items;
        this.menuVisible = true;
    }
}

export const appState = new AppState();
