<script lang="ts">
    import { onMount } from "svelte";
    import { Events, Window } from "@wailsio/runtime";
    import * as ConfigService from "../bindings/olicanaplot/internal/appconfig/configservice";

    // Application state for logging and chart preferences.
    let logPath = $state("");
    let logLevel = $state("info");
    let chartLibrary = $state("echarts");
    let activeTab = $state("logging");
    let isMaximised = $state(false);

    // Initialize configuration settings and window state on component mount.
    onMount(async () => {
        try {
            logPath = await ConfigService.GetLogPath();
            logLevel = await ConfigService.GetLogLevel();
            chartLibrary = await ConfigService.GetChartLibrary();
            isMaximised = await Window.IsMaximised();
        } catch (e) {
            console.error("Failed to get config:", e);
        }
    });

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
            await ConfigService.SetLogPath(logPath);
            await ConfigService.SetLogLevel(logLevel);
            const oldLibrary = await ConfigService.GetChartLibrary();

            if (oldLibrary !== chartLibrary) {
                const confirmed = confirm(
                    "Changing the chart engine will reset the current plot. Continue?",
                );
                if (!confirmed) {
                    chartLibrary = oldLibrary;
                    return;
                }
                await ConfigService.SetChartLibrary(chartLibrary);
                Events.Emit("chartLibraryChanged", chartLibrary);
            } else {
                await ConfigService.SetChartLibrary(chartLibrary);
            }

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
                onclick={() => Window.Close()}
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
                {#if activeTab === "logging"}
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
            <button class="btn btn-secondary" onclick={() => Window.Close()}
                >Cancel</button
            >
            <button class="btn btn-primary" onclick={handleSave}
                >Save Changes</button
            >
        </footer>
    </div>
</div>

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
</style>
