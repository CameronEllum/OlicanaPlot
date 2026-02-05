<script>
    import { onMount } from "svelte";
    import { Events } from "@wailsio/runtime";
    import * as ConfigService from "../../bindings/olicanaplot/internal/appconfig/configservice";

    let { visible, onClose } = $props();
    let logPath = $state("");
    let chartLibrary = $state("echarts");

    onMount(async () => {
        try {
            logPath = await ConfigService.GetLogPath();
            chartLibrary = await ConfigService.GetChartLibrary();
        } catch (e) {
            console.error("Failed to get config:", e);
        }
    });

    async function handleSave() {
        try {
            await ConfigService.SetLogPath(logPath);
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

            <div class="options-body">
                <div class="form-group">
                    <label for="chartLibrary">Chart Library</label>
                    <select id="chartLibrary" bind:value={chartLibrary}>
                        <option value="echarts">Apache ECharts</option>
                        <option value="plotly">Plotly.js (WebGL)</option>
                    </select>
                    <p class="help-text">
                        Select the charting engine. Plotly uses WebGL for large
                        datasets.
                    </p>
                </div>

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
                        The structured log data will be written to this file.
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
                            <polyline points="15 3 21 3 21 9"></polyline>
                            <line x1="10" y1="14" x2="21" y2="3"></line>
                        </svg>
                        Open Log File in Editor
                    </button>
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
        background: rgba(255, 255, 255, 0.05);
        color: #fff;
    }

    .options-body {
        display: flex;
        flex-direction: column;
        gap: 24px;
    }

    .input-group {
        display: flex;
        gap: 8px;
    }

    .help-text {
        margin: 0;
        font-size: 0.8rem;
        color: var(--text-secondary);
    }

    .btn-full {
        width: 100%;
    }
</style>
