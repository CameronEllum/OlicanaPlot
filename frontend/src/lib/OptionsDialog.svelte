<script>
    import { onMount } from "svelte";
    import { Events } from "@wailsio/runtime";
    import * as ConfigService from "../../bindings/olicanaplot/internal/appconfig/configservice";

    let { visible, onClose } = $props();
    let logPath = $state("");
    let logLevel = $state("info");
    let chartLibrary = $state("echarts");
    let activeTab = $state("logging"); // "logging" or "chart"

    onMount(async () => {
        try {
            logPath = await ConfigService.GetLogPath();
            logLevel = await ConfigService.GetLogLevel();
            chartLibrary = await ConfigService.GetChartLibrary();
        } catch (e) {
            console.error("Failed to get config:", e);
        }
    });

    async function handleSave() {
        try {
            await ConfigService.SetLogPath(logPath);
            await ConfigService.SetLogLevel(logLevel);
            const oldLibrary = await ConfigService.GetChartLibrary();
            await ConfigService.SetChartLibrary(chartLibrary);

            // Emit event if chart library changed
            if (oldLibrary !== chartLibrary) {
                Events.Emit("chartLibraryChanged", chartLibrary);
            }

            onClose();
        } catch (e) {
            console.error("Failed to save config:", e);
            alert("Failed to save settings: " + e.message);
        }
    }

    async function handleOpenLog() {
        console.log("Attempting to open log file:", logPath);
        try {
            await ConfigService.OpenLogFile();
            console.log("OpenLogFile command sent successfully");
        } catch (e) {
            console.error("Failed to open log file:", e);
            alert("Failed to open log file: " + e.message);
        }
    }
</script>

{#if visible}
    <div
        class="modal-backdrop"
        onclick={onClose}
        onkeydown={(e) =>
            (e.key === "Escape" || e.key === "Enter") && onClose()}
        role="button"
        tabindex="-1"
    >
        <div
            class="modal-content"
            onclick={(e) => e.stopPropagation()}
            onkeydown={(e) => e.stopPropagation()}
            role="dialog"
            aria-modal="true"
            tabindex="0"
        >
            <div class="modal-header">
                <h3>Application Options</h3>
                <button class="icon-close" onclick={onClose} aria-label="Close">
                    <svg
                        viewBox="0 0 24 24"
                        width="24"
                        height="24"
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

            <div class="options-container">
                <div class="sidebar">
                    <button
                        class="tab-btn {activeTab === 'logging'
                            ? 'active'
                            : ''}"
                        onclick={() => (activeTab = "logging")}
                        title="Logging Configuration"
                    >
                        <svg
                            viewBox="0 0 24 24"
                            width="24"
                            height="24"
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
                            <polyline points="10 9 9 9 8 9"></polyline>
                        </svg>
                    </button>
                    <button
                        class="tab-btn {activeTab === 'chart' ? 'active' : ''}"
                        onclick={() => (activeTab = "chart")}
                        title="Chart Engine Configuration"
                    >
                        <svg
                            viewBox="0 0 24 24"
                            width="24"
                            height="24"
                            stroke="currentColor"
                            stroke-width="2"
                            fill="none"
                        >
                            <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"
                            ></polyline>
                        </svg>
                    </button>
                </div>

                <div class="options-body">
                    {#if activeTab === "logging"}
                        <div class="form-group">
                            <label for="logPath">Log File Path</label>
                            <div class="input-group">
                                <input
                                    type="text"
                                    id="logPath"
                                    bind:value={logPath}
                                    placeholder="C:\path\to\olicana.log"
                                />
                            </div>
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

                        <div class="form-group">
                            <button
                                class="btn btn-secondary btn-full"
                                onclick={handleOpenLog}
                            >
                                <svg
                                    viewBox="0 0 24 24"
                                    width="16"
                                    height="16"
                                    stroke="currentColor"
                                    stroke-width="2"
                                    fill="none"
                                    style="margin-right: 8px;"
                                >
                                    <path
                                        d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"
                                    ></path>
                                    <polyline points="15 3 21 3 21 9"
                                    ></polyline>
                                    <line x1="10" y1="14" x2="21" y2="3"></line>
                                </svg>
                                Open Log File in Editor
                            </button>
                        </div>
                    {:else if activeTab === "chart"}
                        <div class="form-group">
                            <label for="chartLibrary">Chart Library</label>
                            <select id="chartLibrary" bind:value={chartLibrary}>
                                <option value="echarts">Apache ECharts</option>
                                <option value="plotly">Plotly.js (WebGL)</option
                                >
                            </select>
                            <p class="help-text">
                                Select the charting engine. Plotly uses WebGL
                                for large datasets.
                            </p>
                        </div>
                    {/if}
                </div>
            </div>

            <div class="modal-footer">
                <button class="btn btn-secondary" onclick={onClose}
                    >Cancel</button
                >
                <button class="btn btn-primary" onclick={handleSave}
                    >Save Changes</button
                >
            </div>
        </div>
    </div>
{/if}

<style>
    .icon-close {
        background: transparent;
        border: none;
        color: var(--text-secondary);
        cursor: pointer;
        padding: 4px;
        border-radius: 8px;
        display: flex;
        transition: all 0.2s;
    }

    .icon-close:hover {
        background: rgba(0, 0, 0, 0.05);
        color: var(--text-primary);
    }

    :global(.dark-mode) .icon-close:hover {
        background: rgba(255, 255, 255, 0.05);
        color: #fff;
    }

    .options-container {
        display: flex;
        gap: 24px;
        min-height: 280px;
    }

    .sidebar {
        display: flex;
        flex-direction: column;
        gap: 8px;
        width: 60px;
        border-right: 1px solid var(--border-color);
        padding-right: 16px;
    }

    .tab-btn {
        width: 44px;
        height: 44px;
        border-radius: 12px;
        border: 1px solid transparent;
        background: transparent;
        color: var(--text-secondary);
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: all 0.2s;
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
    }

    .options-body {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 14px;
    }

    .input-group {
        display: flex;
        gap: 8px;
    }

    .help-text {
        margin: -4px 0 0 0;
        font-size: 0.8rem;
        color: var(--text-secondary);
    }

    .btn-full {
        width: 100%;
    }
</style>
