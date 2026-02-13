<script lang="ts">
    import { onMount } from "svelte";
    import { Events, Window, Dialogs } from "@wailsio/runtime";
    import * as ConfigService from "../../../../bindings/olicanaplot/internal/appconfig/configservice";
    import * as PluginService from "../../../../bindings/olicanaplot/internal/plugins/service";
    import ContextMenu from "../ContextMenu.svelte";

    // Application state for logging, chart, and plugin preferences.
    let logPath = $state("");
    let logLevel = $state("info");
    let chartLibrary = $state("echarts");
    let plugins = $state<any[]>([]);
    let pluginSearchDirs = $state<string[]>([]);
    let showGeneratorsMenu = $state(true);
    let defaultLineWidth = $state(2.0);
    let activeTab = $state("general");
    let isMaximised = $state(false);

    // Context Menu State
    let menuVisible = $state(false);
    let menuX = $state(0);
    let menuY = $state(0);
    let menuItems = $state<
        { label: string; action?: () => void; header?: boolean }[]
    >([]);

    // Initialize configuration settings and window state on component mount.
    onMount(async () => {
        try {
            logPath = await ConfigService.GetLogPath();
            logLevel = await ConfigService.GetLogLevel();
            chartLibrary = await ConfigService.GetChartLibrary();
            plugins = await PluginService.ListPlugins();
            pluginSearchDirs = await ConfigService.GetPluginSearchDirs();
            showGeneratorsMenu = await ConfigService.GetShowGeneratorsMenu();
            defaultLineWidth = await ConfigService.GetDefaultLineWidth();
            isMaximised = await Window.IsMaximised();
        } catch (e) {
            console.error("Failed to get config:", e);
        }
    });

    async function togglePlugin(name: string, enabled: boolean) {
        try {
            await PluginService.SetPluginEnabled(name, enabled);
            plugins = await PluginService.ListPlugins();
        } catch (e) {
            console.error("Failed to toggle plugin:", e);
        }
    }

    function handlePluginContextMenu(event: MouseEvent, plugin: any) {
        event.preventDefault();
        event.stopPropagation();

        menuX = event.clientX;
        menuY = event.clientY;
        menuItems = [
            { label: plugin.name, header: true },
            {
                label: plugin.enabled ? "Disable Plugin" : "Enable Plugin",
                action: () => togglePlugin(plugin.name, !plugin.enabled),
            },
        ];

        if (plugin.path) {
            menuItems.push({
                label: "Show in File Explorer",
                action: () => PluginService.ShowInExplorer(plugin.path),
            });
        }

        if (plugin.patterns && plugin.patterns.length > 0) {
            menuItems.push({ label: "Supported Files:", header: true });
            for (const pattern of plugin.patterns) {
                menuItems.push({
                    label: `• ${pattern.description} (${pattern.patterns.join(", ")})`,
                });
            }
        }

        menuVisible = true;
    }

    // Toggle the window between maximised and restored states.
    async function handleToggleMaximise() {
        if (isMaximised) {
            await Window.UnMaximise();
        } else {
            await Window.Maximise();
        }
        isMaximised = await Window.IsMaximised();
    }

    // Persist all configuration changes to the backend and manage chart library
    // reset confirmation.
    async function handleSave() {
        try {
            PluginService.LogDebug("Options", "handleSave starting", "");
            await ConfigService.SetLogPath(logPath);
            await ConfigService.SetLogLevel(logLevel);
            const oldLibrary = await ConfigService.GetChartLibrary();

            if (oldLibrary !== chartLibrary) {
                const res = await Dialogs.Question({
                    Title: "Change Chart Engine",
                    Message:
                        "Changing the chart engine will erase the current plot and reset to defaults. Continue?",
                    Buttons: [
                        { Label: "Yes", IsDefault: true },
                        { Label: "No", IsCancel: true },
                    ],
                });

                PluginService.LogDebug("Options", "Dialog result: " + res, "");

                // In Wails v3, the result is the label of the button clicked.
                if (res !== "Yes") {
                    chartLibrary = oldLibrary;
                    return;
                }
                await ConfigService.SetChartLibrary(chartLibrary);
                Events.Emit("chartLibraryChanged", chartLibrary);
            } else {
                await ConfigService.SetChartLibrary(chartLibrary);
            }

            await ConfigService.SetShowGeneratorsMenu(showGeneratorsMenu);
            await ConfigService.SetDefaultLineWidth(defaultLineWidth);
            await ConfigService.SetPluginSearchDirs(
                $state.snapshot(pluginSearchDirs),
            );

            PluginService.LogDebug(
                "Options",
                "handleSave complete, closing window",
                "",
            );
            Window.Close();
        } catch (e: any) {
            console.error("Failed to save config:", e);
            alert("Failed to save settings: " + e.message);
        }
    }

    // Command the backend service to open the active log file in the system
    // default editor.
    async function handleOpenLog() {
        try {
            await ConfigService.OpenLogFile();
        } catch (e: any) {
            console.error("Failed to open log file:", e);
            alert("Failed to open log file: " + e.message);
        }
    }

    async function handleAddSearchDir() {
        try {
            const result = await Dialogs.OpenFile({
                Title: "Select Plugin Search Directory",
                CanChooseDirectories: true,
                CanChooseFiles: false,
            });
            if (result && !pluginSearchDirs.includes(result as string)) {
                pluginSearchDirs.push(result as string);
            }
        } catch (e) {
            console.error("Failed to choose directory:", e);
        }
    }

    function handleRemoveSearchDir(dir: string) {
        pluginSearchDirs = pluginSearchDirs.filter((d) => d !== dir);
    }
</script>

<div class="window-container">
    <header class="titlebar">
        <div class="title">Application Options</div>
        <div class="controls">
            <button
                class="control-btn"
                onclick={() => Window.Minimise()}
                title="Minimise"
            >
                <svg
                    viewBox="0 0 24 24"
                    width="14"
                    height="14"
                    stroke="currentColor"
                    stroke-width="2"
                    fill="none"
                    ><line x1="5" y1="12" x2="19" y2="12"></line></svg
                >
            </button>
            <button
                class="control-btn"
                onclick={handleToggleMaximise}
                title={isMaximised ? "Restore" : "Maximise"}
            >
                {#if isMaximised}
                    <svg
                        viewBox="0 0 24 24"
                        width="14"
                        height="14"
                        stroke="currentColor"
                        stroke-width="2"
                        fill="none"
                        ><rect x="8" y="4" width="12" height="12"
                        ></rect><polyline points="4 8 4 20 16 20"
                        ></polyline></svg
                    >
                {:else}
                    <svg
                        viewBox="0 0 24 24"
                        width="14"
                        height="14"
                        stroke="currentColor"
                        stroke-width="2"
                        fill="none"
                        ><rect x="3" y="3" width="18" height="18" rx="2" ry="2"
                        ></rect></svg
                    >
                {/if}
            </button>
            <button
                class="control-btn close-btn"
                onclick={() => {
                    if (Window && typeof Window.Close === "function") {
                        Window.Close();
                    } else {
                        (window as any).wails?.Window?.Close();
                    }
                }}
                title="Close"
            >
                <svg
                    viewBox="0 0 24 24"
                    width="14"
                    height="14"
                    stroke="currentColor"
                    stroke-width="2"
                    fill="none"
                    ><line x1="18" y1="6" x2="6" y2="18"></line><line
                        x1="6"
                        y1="6"
                        x2="18"
                        y2="18"
                    ></line></svg
                >
            </button>
        </div>
    </header>

    <div class="content-wrapper">
        <div class="options-layout">
            <aside class="sidebar">
                <button
                    class="tab-btn {activeTab === 'general' ? 'active' : ''}"
                    onclick={() => (activeTab = "general")}
                    title="General Application Settings"
                >
                    <svg
                        viewBox="0 0 24 24"
                        width="20"
                        height="20"
                        stroke="currentColor"
                        stroke-width="2"
                        fill="none"
                    >
                        <circle cx="12" cy="12" r="3"></circle>
                        <path
                            d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"
                        ></path>
                    </svg>
                    <span>General</span>
                </button>
                <button
                    class="tab-btn {activeTab === 'plotting' ? 'active' : ''}"
                    onclick={() => (activeTab = "plotting")}
                    title="Plotting Defaults"
                >
                    <svg
                        viewBox="0 0 24 24"
                        width="20"
                        height="20"
                        stroke="currentColor"
                        stroke-width="2"
                        fill="none"
                    >
                        <path d="M3 3v18h18" />
                        <path d="M19 9l-5 5-4-4-3 3" />
                    </svg>
                    <span>Plotting</span>
                </button>
                <button
                    class="tab-btn {activeTab === 'plugins' ? 'active' : ''}"
                    onclick={() => (activeTab = "plugins")}
                    title="Plugin Configuration"
                >
                    <svg
                        viewBox="0 0 24 24"
                        width="20"
                        height="20"
                        stroke="currentColor"
                        stroke-width="2"
                        fill="none"
                    >
                        <path
                            d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"
                        ></path>
                        <polyline points="3.27 6.96 12 12.01 20.73 6.96"
                        ></polyline>
                        <line x1="12" y1="22.08" x2="12" y2="12"></line>
                    </svg>
                    <span>Plugins</span>
                </button>
                <button
                    class="tab-btn {activeTab === 'logging' ? 'active' : ''}"
                    onclick={() => (activeTab = "logging")}
                    title="Logging Configuration"
                >
                    <svg
                        viewBox="0 0 24 24"
                        width="20"
                        height="20"
                        stroke="currentColor"
                        stroke-width="2"
                        fill="none"
                    >
                        <path
                            d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"
                        ></path>
                        <polyline points="14 2 14 8 20 8"></polyline>
                        <line x1="16" y1="13" x2="8" y2="13"></line>
                        <line x1="16" y1="17" x2="8" y2="17"></line>
                    </svg>
                    <span>Logging</span>
                </button>
                <button
                    class="tab-btn {activeTab === 'chart' ? 'active' : ''}"
                    onclick={() => (activeTab = "chart")}
                    title="Chart Engine Configuration"
                >
                    <svg
                        viewBox="0 0 24 24"
                        width="20"
                        height="20"
                        stroke="currentColor"
                        stroke-width="2"
                        fill="none"
                    >
                        <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"
                        ></polyline>
                    </svg>
                    <span>Chart</span>
                </button>
            </aside>

            <main class="options-body">
                {#if activeTab === "general"}
                    <div class="form-group">
                        <label class="checkbox-item">
                            <input
                                type="checkbox"
                                bind:checked={showGeneratorsMenu}
                            />
                            <div class="checkbox-info">
                                <span class="title">Show Generators Menu</span>
                                <p class="help-text">
                                    Display the "Generators" button in the main
                                    application toolbar.
                                </p>
                            </div>
                        </label>
                    </div>
                {:else if activeTab === "plotting"}
                    <section class="form-section">
                        <h3>Defaults</h3>
                        <div class="form-group">
                            <label for="defaultLineWidth">Line Width</label>
                            <input
                                type="number"
                                id="defaultLineWidth"
                                bind:value={defaultLineWidth}
                                step="0.1"
                                min="0.1"
                                max="10"
                            />
                            <p class="help-text">
                                Default line width for all chart series.
                            </p>
                        </div>
                    </section>
                {:else if activeTab === "plugins"}
                    <section class="plugin-section">
                        <div class="section-header">
                            <h3>Plugin Search Directories</h3>
                            <button
                                class="btn btn-secondary btn-sm"
                                onclick={handleAddSearchDir}
                            >
                                <svg
                                    viewBox="0 0 24 24"
                                    width="14"
                                    height="14"
                                    stroke="currentColor"
                                    stroke-width="2"
                                    fill="none"
                                >
                                    <line x1="12" y1="5" x2="12" y2="19"></line>
                                    <line x1="5" y1="12" x2="19" y2="12"></line>
                                </svg>
                                Add Directory
                            </button>
                        </div>
                        <div class="dir-list">
                            <div class="dir-item built-in">
                                <span class="dir-path">Built-in Plugins</span>
                                <span class="badge">Read-only</span>
                            </div>
                            {#each pluginSearchDirs as dir}
                                <div class="dir-item">
                                    <span class="dir-path" title={dir}
                                        >{dir}</span
                                    >
                                    <button
                                        class="icon-btn remove-btn"
                                        onclick={() =>
                                            handleRemoveSearchDir(dir)}
                                        title="Remove directory"
                                    >
                                        <svg
                                            viewBox="0 0 24 24"
                                            width="14"
                                            height="14"
                                            stroke="currentColor"
                                            stroke-width="2"
                                            fill="none"
                                        >
                                            <line x1="18" y1="6" x2="6" y2="18"
                                            ></line>
                                            <line x1="6" y1="6" x2="18" y2="18"
                                            ></line>
                                        </svg>
                                    </button>
                                </div>
                            {/each}
                        </div>
                        <p class="help-text mt-8">
                            Note: Changes to search directories require an
                            application restart to take effect.
                        </p>
                    </section>

                    <section class="plugin-section">
                        <h3>External Plugins</h3>
                        <div class="plugin-list">
                            {#each plugins.filter((p: any) => !p.is_internal) as plugin}
                                <div
                                    class="plugin-item"
                                    oncontextmenu={(e) =>
                                        handlePluginContextMenu(e, plugin)}
                                    role="listitem"
                                >
                                    <input
                                        type="checkbox"
                                        id="plugin-ext-{plugin.name}"
                                        checked={plugin.enabled}
                                        onchange={(e) =>
                                            togglePlugin(
                                                plugin.name,
                                                (e.target as HTMLInputElement)
                                                    .checked,
                                            )}
                                    />
                                    <label
                                        for="plugin-ext-{plugin.name}"
                                        class="plugin-info"
                                    >
                                        <span class="plugin-name"
                                            >{plugin.name}</span
                                        >
                                        <span class="plugin-meta">
                                            {#if plugin.patterns && plugin.patterns.length > 0}
                                                SUPPORT: {plugin.patterns
                                                    .map(
                                                        (p: any) =>
                                                            `${p.description} (${p.patterns.join(", ")})`,
                                                    )
                                                    .join(", ")}
                                            {:else}
                                                GENERATOR
                                            {/if}
                                        </span>
                                        {#if plugin.path}
                                            <span class="plugin-meta path"
                                                >{plugin.path.length > 50
                                                    ? "..." +
                                                      plugin.path.slice(-47)
                                                    : plugin.path}</span
                                            >
                                        {/if}
                                    </label>
                                </div>
                            {/each}
                            {#if plugins.filter((p: any) => !p.is_internal).length === 0}
                                <p class="help-text">
                                    No external plugins detected.
                                </p>
                            {/if}
                        </div>
                    </section>

                    <section class="plugin-section">
                        <h3>Internal Plugins</h3>
                        <div class="plugin-list">
                            {#each plugins.filter((p: any) => p.is_internal) as plugin}
                                <div
                                    class="plugin-item"
                                    oncontextmenu={(e) =>
                                        handlePluginContextMenu(e, plugin)}
                                    role="listitem"
                                >
                                    <input
                                        type="checkbox"
                                        id="plugin-int-{plugin.name}"
                                        checked={plugin.enabled}
                                        onchange={(e) =>
                                            togglePlugin(
                                                plugin.name,
                                                (e.target as HTMLInputElement)
                                                    .checked,
                                            )}
                                    />
                                    <label
                                        for="plugin-int-{plugin.name}"
                                        class="plugin-info"
                                    >
                                        <span class="plugin-name"
                                            >{plugin.name}</span
                                        >
                                        <span class="plugin-meta">
                                            {#if plugin.patterns && plugin.patterns.length > 0}
                                                SUPPORT: {plugin.patterns
                                                    .map(
                                                        (p: any) =>
                                                            `${p.description} (${p.patterns.join(", ")})`,
                                                    )
                                                    .join(", ")}
                                            {:else}
                                                GENERATOR
                                            {/if}
                                        </span>
                                    </label>
                                </div>
                            {/each}
                        </div>
                    </section>
                {:else if activeTab === "logging"}
                    <div class="form-group">
                        <label for="logPath">Log File Path</label>
                        <input type="text" id="logPath" bind:value={logPath} />
                        <p class="help-text">
                            The structured log data will be written to this
                            file.
                        </p>
                    </div>

                    <div class="form-group">
                        <label for="logLevel">Logging Level</label>
                        <select id="logLevel" bind:value={logLevel}>
                            <option value="debug">Debug</option>
                            <option value="info">Info</option>
                            <option value="warn">Warning</option>
                            <option value="error">Error</option>
                        </select>
                        <p class="help-text">
                            Controls the verbosity of application logs.
                        </p>
                    </div>

                    <button class="btn btn-secondary" onclick={handleOpenLog}>
                        Open Log File in Editor
                    </button>
                {:else if activeTab === "chart"}
                    <div class="form-group">
                        <label for="chartLibrary">Chart Library</label>
                        <select id="chartLibrary" bind:value={chartLibrary}>
                            <option value="echarts">Apache ECharts</option>
                            <option value="plotly">Plotly.js (WebGL)</option>
                        </select>
                        <p class="help-text">
                            Select the charting engine. Plotly uses WebGL for
                            large datasets.
                        </p>
                    </div>
                {/if}
            </main>
        </div>

        <footer class="window-footer">
            <button
                class="btn btn-secondary"
                onclick={() => {
                    if (Window && typeof Window.Close === "function") {
                        Window.Close();
                    } else {
                        (window as any).wails?.Window?.Close();
                    }
                }}>Cancel</button
            >
            <button class="btn btn-primary" onclick={handleSave}
                >Save Changes</button
            >
        </footer>
    </div>
</div>

<ContextMenu
    x={menuX}
    y={menuY}
    visible={menuVisible}
    items={menuItems}
    onClose={() => (menuVisible = false)}
/>

<style>
    :global(body) {
        margin: 0;
        padding: 0;
        overflow: hidden;
        background: var(--bg-primary);
        color: var(--text-primary);
        font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
            Helvetica, Arial, sans-serif;
        --wails-resize: all;
    }

    .window-container {
        display: flex;
        flex-direction: column;
        height: 100vh;
        background: var(--bg-primary);
    }

    .titlebar {
        --wails-draggable: drag;
        display: flex;
        justify-content: space-between;
        align-items: center;
        height: 32px;
        background: var(--bg-secondary);
        padding: 0 8px 0 16px;
        border-bottom: 1px solid var(--border-color);
        user-select: none;
    }

    .title {
        font-size: 12px;
        font-weight: 500;
        color: var(--text-secondary);
    }

    .controls {
        display: flex;
        gap: 4px;
    }

    .control-btn {
        --wails-draggable: no-drag;
        display: flex;
        align-items: center;
        justify-content: center;
        width: 28px;
        height: 24px;
        border: none;
        background: transparent;
        color: var(--text-secondary);
        cursor: pointer;
        border-radius: 4px;
        transition: all 0.2s;
    }

    .control-btn:hover {
        background: rgba(0, 0, 0, 0.05);
        color: var(--text-primary);
    }

    :global(.dark-mode) .control-btn:hover {
        background: rgba(255, 255, 255, 0.1);
    }

    .close-btn:hover {
        background: #e81123 !important;
        color: white !important;
    }

    .content-wrapper {
        flex: 1;
        display: flex;
        flex-direction: column;
        padding: 24px;
        overflow: hidden;
    }

    .options-layout {
        display: flex;
        flex: 1;
        gap: 32px;
        overflow: hidden;
    }

    .sidebar {
        display: flex;
        flex-direction: column;
        gap: 8px;
        width: 140px;
    }

    .tab-btn {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 10px 16px;
        border: 1px solid transparent;
        border-radius: 8px;
        background: transparent;
        color: var(--text-secondary);
        cursor: pointer;
        transition: all 0.2s;
        font-size: 14px;
        text-align: left;
    }

    .tab-btn:hover {
        background: rgba(0, 0, 0, 0.05);
        color: var(--text-primary);
    }

    :global(.dark-mode) .tab-btn:hover {
        background: rgba(255, 255, 255, 0.05);
    }

    .tab-btn.active {
        background: rgba(99, 102, 241, 0.1);
        color: var(--accent);
        border-color: var(--accent);
        font-weight: 500;
    }

    .options-body {
        flex: 1;
        overflow-y: auto;
        padding-right: 8px;
    }

    .form-group {
        margin-bottom: 24px;
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    label {
        font-size: 13px;
        font-weight: 500;
        color: var(--text-primary);
    }

    input,
    select {
        padding: 8px 12px;
        border: 1px solid var(--border-color);
        border-radius: 6px;
        background: var(--bg-tertiary);
        color: var(--text-primary);
        font-size: 14px;
    }

    input:focus,
    select:focus {
        outline: none;
        border-color: var(--accent);
        box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2);
    }

    .help-text {
        font-size: 12px;
        color: var(--text-secondary);
        margin: 0;
    }

    .window-footer {
        display: flex;
        justify-content: flex-end;
        gap: 12px;
        margin-top: 24px;
        padding-top: 24px;
        border-top: 1px solid var(--border-color);
    }

    .btn {
        padding: 8px 16px;
        border-radius: 6px;
        font-size: 14px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
        border: 1px solid transparent;
    }

    .btn-primary {
        background: var(--accent);
        color: white;
    }

    .btn-primary:hover {
        filter: brightness(1.1);
    }

    .btn-secondary {
        background: var(--bg-secondary);
        border-color: var(--border-color);
        color: var(--text-primary);
    }

    .btn-secondary:hover {
        background: var(--bg-tertiary);
    }

    .plugin-section {
        margin-bottom: 32px;
    }

    .plugin-section h3 {
        font-size: 14px;
        font-weight: 600;
        color: var(--text-secondary);
        margin-bottom: 16px;
        text-transform: uppercase;
        letter-spacing: 0.05em;
    }

    .plugin-list {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .plugin-item {
        display: flex;
        align-items: flex-start;
        gap: 16px;
        padding: 12px 16px;
        background: var(--bg-secondary);
        border: 1px solid var(--border-color);
        border-radius: 8px;
        cursor: pointer;
        transition: all 0.2s;
    }

    .plugin-item:hover {
        border-color: var(--accent);
        background: var(--bg-tertiary);
    }

    .plugin-item input[type="checkbox"],
    .checkbox-item input[type="checkbox"] {
        margin-top: 4px;
        width: 16px;
        height: 16px;
        cursor: pointer;
    }

    .checkbox-item {
        margin-top: 4px; /* Local adjustment for options layout */
    }

    .plugin-info {
        display: flex;
        flex-direction: column;
        gap: 4px;
    }

    .plugin-name {
        font-size: 14px;
        font-weight: 500;
        color: var(--text-primary);
    }

    .plugin-meta {
        font-size: 12px;
        color: var(--text-secondary);
    }

    .plugin-meta.path {
        font-family: var(--font-mono, monospace);
        opacity: 0.7;
        font-size: 11px;
        margin-top: 4px;
        word-break: break-all;
    }

    .section-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 16px;
    }

    .section-header h3 {
        margin-bottom: 0 !important;
    }

    .dir-list {
        display: flex;
        flex-direction: column;
        gap: 8px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border-color);
        border-radius: 8px;
        padding: 8px;
    }

    .dir-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 12px;
        padding: 8px 12px;
        background: var(--bg-secondary);
        border: 1px solid var(--border-color);
        border-radius: 6px;
        font-size: 13px;
    }

    .dir-path {
        flex: 1;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        color: var(--text-primary);
    }

    .icon-btn {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 24px;
        height: 24px;
        border: none;
        background: transparent;
        color: var(--text-secondary);
        cursor: pointer;
        border-radius: 4px;
        transition: all 0.2s;
    }

    .icon-btn:hover {
        background: rgba(0, 0, 0, 0.05);
        color: var(--text-primary);
    }

    .remove-btn:hover {
        background: rgba(232, 17, 35, 0.1) !important;
        color: #e81123 !important;
    }

    .badge {
        font-size: 10px;
        padding: 2px 6px;
        background: var(--bg-primary);
        color: var(--text-secondary);
        border-radius: 4px;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        font-weight: 600;
        border: 1px solid var(--border-color);
    }

    .btn-sm {
        padding: 4px 10px;
        font-size: 12px;
        display: flex;
        align-items: center;
        gap: 6px;
    }

    .mt-8 {
        margin-top: 8px;
    }
</style>
