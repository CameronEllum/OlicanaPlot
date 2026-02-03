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
                <div class="option-item">
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

                <div class="option-item">
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

                <div class="option-item">
                    <button
                        class="btn-secondary btn-full"
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
                <button class="btn-secondary" onclick={onClose}>Cancel</button>
                <button class="btn-primary" onclick={handleSave}
                    >Save Changes</button
                >
            </div>
        </div>
    </div>
{/if}

<style>
    .modal-backdrop {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.6);
        display: flex;
        justify-content: center;
        align-items: center;
        z-index: 3000;
        backdrop-filter: blur(4px);
    }

    .modal-content {
        background: white;
        padding: 0;
        border-radius: 12px;
        min-width: 500px;
        max-width: 90vw;
        box-shadow: 0 10px 40px rgba(0, 0, 0, 0.3);
        display: flex;
        flex-direction: column;
        overflow: hidden;
        border: 1px solid rgba(0, 0, 0, 0.1);
    }

    :global(.dark-mode) .modal-content {
        background: #1e1e1e;
        color: #eee;
        border-color: #444;
    }

    .modal-header {
        padding: 20px 24px;
        background: #f8f9fa;
        border-bottom: 1px solid #eee;
        display: flex;
        justify-content: space-between;
        align-items: center;
    }

    :global(.dark-mode) .modal-header {
        background: #252525;
        border-bottom-color: #333;
    }

    h3 {
        margin: 0;
        font-size: 1.25rem;
        color: #2a3f5f;
        font-weight: 600;
    }

    :global(.dark-mode) h3 {
        color: #fff;
    }

    .icon-close {
        background: transparent;
        border: none;
        color: #888;
        cursor: pointer;
        padding: 4px;
        border-radius: 4px;
        display: flex;
        transition: all 0.2s;
    }

    .icon-close:hover {
        background: #e9ecef;
        color: #333;
    }

    :global(.dark-mode) .icon-close:hover {
        background: #333;
        color: #fff;
    }

    .options-body {
        padding: 24px;
        display: flex;
        flex-direction: column;
        gap: 20px;
    }

    .option-item {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    label {
        font-weight: 600;
        color: #506784;
        font-size: 0.95rem;
    }

    :global(.dark-mode) label {
        color: #a0a0a0;
    }

    .input-group {
        display: flex;
        gap: 8px;
    }

    input[type="text"] {
        flex: 1;
        padding: 10px 14px;
        border: 1px solid #ddd;
        border-radius: 6px;
        font-size: 0.95rem;
        font-family: "JetBrains Mono", "Courier New", monospace;
        background: white;
        color: #2a3f5f;
        transition: border-color 0.2s;
    }

    input[type="text"]:focus {
        outline: none;
        border-color: #4db8ff;
        box-shadow: 0 0 0 3px rgba(77, 184, 255, 0.1);
    }

    :global(.dark-mode) input[type="text"] {
        background: #121212;
        border-color: #444;
        color: #eee;
    }

    select {
        padding: 10px 14px;
        border: 1px solid #ddd;
        border-radius: 6px;
        font-size: 0.95rem;
        background: white;
        color: #2a3f5f;
        cursor: pointer;
        transition: border-color 0.2s;
    }

    select:focus {
        outline: none;
        border-color: #4db8ff;
        box-shadow: 0 0 0 3px rgba(77, 184, 255, 0.1);
    }

    :global(.dark-mode) select {
        background: #121212;
        border-color: #444;
        color: #eee;
    }

    .help-text {
        margin: 0;
        font-size: 0.85rem;
        color: #888;
    }

    .modal-footer {
        padding: 16px 24px;
        background: #f8f9fa;
        border-top: 1px solid #eee;
        display: flex;
        justify-content: flex-end;
        gap: 12px;
    }

    :global(.dark-mode) .modal-footer {
        background: #252525;
        border-top-color: #333;
    }

    .btn-primary,
    .btn-secondary {
        padding: 10px 20px;
        border-radius: 6px;
        font-weight: 600;
        cursor: pointer;
        font-size: 0.95rem;
        transition: all 0.2s;
    }

    .btn-primary {
        background: #007bff;
        color: white;
        border: 1px solid #0069d9;
    }

    .btn-primary:hover {
        background: #0069d9;
    }

    .btn-secondary {
        background: white;
        color: #506784;
        border: 1px solid #ddd;
    }

    .btn-secondary:hover {
        background: #f8f9fa;
        border-color: #ccc;
    }

    :global(.dark-mode) .btn-secondary {
        background: #333;
        color: #eee;
        border-color: #444;
    }

    :global(.dark-mode) .btn-secondary:hover {
        background: #444;
        border-color: #555;
    }

    .btn-full {
        width: 100%;
        display: flex;
        justify-content: center;
        align-items: center;
    }
</style>
