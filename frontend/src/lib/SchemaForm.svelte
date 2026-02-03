<script>
    import { onMount } from "svelte";

    let {
        schema = $bindable({}),
        uiSchema = $bindable({}),
        title = "Configuration",
        requestID = "",
        handleFormChange = false,
        onsubmit,
        oncancel,
    } = $props();

    let formData = $state({});
    let loading = $state(false);
    let loadingTimer = null;

    // Initialize formData with defaults from schema
    $effect(() => {
        if (schema && schema.properties) {
            Object.keys(schema.properties).forEach((key) => {
                const prop = schema.properties[key];
                if (formData[key] === undefined) {
                    formData[key] =
                        prop.default !== undefined
                            ? prop.default
                            : prop.type === "integer" || prop.type === "number"
                              ? 0
                              : prop.type === "array"
                                ? []
                                : "";
                }
            });
        }
    });

    // Handle external data updates from plugin
    import { Events } from "@wailsio/runtime";
    onMount(() => {
        if (!requestID) return;

        const unsub = Events.On(`ipc-form-update-${requestID}`, (e) => {
            const update = e.data || e;
            if (update.schema) schema = update.schema;
            if (update.uiSchema) uiSchema = update.uiSchema;
            if (
                update.data &&
                typeof update.data === "object" &&
                !Array.isArray(update.data)
            ) {
                Object.assign(formData, update.data);
            }
            // Stop loading state when update received
            stopLoading();
        });
        return unsub;
    });

    function startLoading() {
        if (loadingTimer) clearTimeout(loadingTimer);
        loadingTimer = setTimeout(() => {
            loading = true;
        }, 250);
    }

    function stopLoading() {
        if (loadingTimer) {
            clearTimeout(loadingTimer);
            loadingTimer = null;
        }
        loading = false;
    }

    // Debounce form changes to notify host
    let changeTimer = null;
    let isFirstRun = true;
    $effect(() => {
        // Track formData changes and also isFirstRun
        JSON.stringify(formData);

        if (changeTimer) clearTimeout(changeTimer);
        changeTimer = setTimeout(() => {
            if (isFirstRun) {
                isFirstRun = false;
                return;
            }
            if (requestID) {
                console.log("Emitting form change:", formData);
                if (handleFormChange) {
                    startLoading();
                }
                Events.Emit(`ipc-form-change-${requestID}`, formData);
            }
        }, 100);
    });

    function handleSubmit() {
        if (onsubmit) onsubmit(formData);
    }

    function handleCancel() {
        if (oncancel) oncancel();
    }

    // Helper for log10 conversion if requested in uiSchema
    function getSliderValue(key, val) {
        const ui = uiSchema[key] || {};
        if (ui["ui:options"]?.scale === "log10") {
            return Math.log10(val || 1);
        }
        return val || 0;
    }

    function setSliderValue(key, val) {
        const ui = uiSchema[key] || {};
        if (ui["ui:options"]?.scale === "log10") {
            formData[key] = Math.pow(10, val);
        } else {
            formData[key] = val;
        }
    }

    function formatValue(key, val) {
        const ui = uiSchema[key] || {};
        if (ui["ui:options"]?.scale === "log10") {
            if (val >= 0.95 * 1000000) return (val / 1000000).toFixed(1) + "M";
            if (val >= 0.95 * 1000) return (val / 1000).toFixed(0) + "K";
            return Math.round(val);
        }
        return val;
    }

    let container = $state(null);
    $effect(() => {
        if (!container || !requestID) return;

        const resizeObserver = new ResizeObserver((entries) => {
            for (let entry of entries) {
                const height = entry.target.getBoundingClientRect().height;
                Events.Emit(`ipc-form-resize-${requestID}`, {
                    width: 500,
                    height: Math.ceil(height),
                });
            }
        });

        resizeObserver.observe(container);
        return () => resizeObserver.disconnect();
    });
</script>

<div class="schema-form-container" bind:this={container}>
    <div class="form-content">
        {#if schema && schema.properties}
            {#each Object.keys(schema.properties) as key}
                {@const prop = schema.properties[key]}
                {@const ui = uiSchema[key] || {}}

                <div class="form-group">
                    <label for={key}>{prop.title || key}</label>

                    {#if prop.enum || prop.oneOf}
                        <select id={key} bind:value={formData[key]}>
                            {#if prop.oneOf}
                                {#each prop.oneOf as option}
                                    <option
                                        value={option.const !== undefined
                                            ? option.const
                                            : option}
                                    >
                                        {option.title || option}
                                    </option>
                                {/each}
                            {:else}
                                {#each prop.enum as option}
                                    <option value={option}>{option}</option>
                                {/each}
                            {/if}
                        </select>
                    {:else if prop.type === "array" && (prop.items?.enum || prop.items?.oneOf)}
                        <div class="checkbox-group">
                            {#each prop.items.oneOf || prop.items.enum.map( (v) => ({ const: v, title: v }), ) as option}
                                {@const val =
                                    option.const !== undefined
                                        ? option.const
                                        : option}
                                <label class="checkbox-label">
                                    <input
                                        type="checkbox"
                                        checked={(formData[key] || []).includes(
                                            val,
                                        )}
                                        onchange={(e) => {
                                            const current = formData[key] || [];
                                            if (e.target.checked) {
                                                formData[key] = [
                                                    ...current,
                                                    val,
                                                ];
                                            } else {
                                                formData[key] = current.filter(
                                                    (v) => v !== val,
                                                );
                                            }
                                        }}
                                    />
                                    {option.title || val}
                                </label>
                            {/each}
                        </div>
                    {:else if prop.type === "integer" || prop.type === "number"}
                        {#if ui["ui:widget"] === "range"}
                            <div class="slider-container">
                                <input
                                    type="range"
                                    id={key}
                                    min={ui["ui:options"]?.scale === "log10"
                                        ? Math.log10(prop.minimum || 1)
                                        : prop.minimum || 0}
                                    max={ui["ui:options"]?.scale === "log10"
                                        ? Math.log10(prop.maximum || 100)
                                        : prop.maximum || 100}
                                    step={ui["ui:options"]?.scale === "log10"
                                        ? 0.1
                                        : prop.step || 1}
                                    value={getSliderValue(key, formData[key])}
                                    oninput={(e) =>
                                        setSliderValue(
                                            key,
                                            parseFloat(e.target.value),
                                        )}
                                />
                                <span class="value-display"
                                    >{formatValue(key, formData[key])}</span
                                >
                            </div>
                        {:else}
                            <input
                                type="number"
                                id={key}
                                bind:value={formData[key]}
                                min={prop.minimum}
                                max={prop.maximum}
                                step={prop.step}
                            />
                        {/if}
                    {:else}
                        <input
                            type="text"
                            id={key}
                            bind:value={formData[key]}
                            placeholder={prop.default}
                        />
                    {/if}

                    {#if prop.description}
                        <p class="description">{prop.description}</p>
                    {/if}
                </div>
            {/each}
        {/if}
    </div>

    {#if formData.order !== undefined && formData.multiplier !== undefined}
        <div class="preview-stats">
            <span class="preview-label">Points per Series:</span>
            <span class="preview-value">
                {Math.round(
                    formData.multiplier * Math.pow(10, formData.order),
                ).toLocaleString()}
            </span>
        </div>
    {/if}

    <div class="actions">
        <button class="cancel-btn" onclick={handleCancel}>Cancel</button>
        <button class="submit-btn" onclick={handleSubmit}>OK</button>
    </div>

    {#if loading}
        <div class="form-loading-overlay">
            <div class="spinner"></div>
        </div>
    {/if}
</div>

<style>
    .schema-form-container {
        padding: 24px;
        width: 100%;
        background: #fff;
        color: #2a3f5f;
    }

    :global(.dark-mode) .schema-form-container {
        background: rgba(45, 45, 45, 0.95);
        color: #e0e0e0;
    }

    .form-content {
        display: flex;
        flex-direction: column;
        gap: 16px;
        margin-bottom: 24px;
    }

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    label {
        font-weight: 600;
        font-size: 0.9em;
    }

    input[type="text"],
    input[type="number"],
    select {
        padding: 8px 12px;
        border: 1px solid #d8d8d8;
        border-radius: 6px;
        background: #fff;
        font-size: 1em;
    }

    :global(.dark-mode) input[type="text"],
    :global(.dark-mode) input[type="number"],
    :global(.dark-mode) select {
        background: #3d3d3d;
        border-color: #555;
        color: #fff;
    }

    .slider-container {
        display: flex;
        align-items: center;
        gap: 12px;
    }

    input[type="range"] {
        flex: 1;
        cursor: pointer;
    }

    .value-display {
        min-width: 50px;
        font-weight: 700;
        color: #4a90e2;
    }

    .description {
        font-size: 0.8em;
        color: #666;
        margin: 4px 0 0 0;
    }

    :global(.dark-mode) .description {
        color: #aaa;
    }

    .checkbox-group {
        display: flex;
        flex-direction: column;
        gap: 8px;
        max-height: 200px;
        overflow-y: auto;
        padding: 10px;
        background: rgba(0, 0, 0, 0.03);
        border-radius: 6px;
        border: 1px solid #eee;
    }

    :global(.dark-mode) .checkbox-group {
        background: rgba(0, 0, 0, 0.2);
        border-color: #444;
    }

    .checkbox-label {
        display: flex;
        align-items: center;
        gap: 8px;
        font-weight: normal;
        cursor: pointer;
        padding: 4px 0;
    }

    .checkbox-label input {
        cursor: pointer;
    }

    .preview-stats {
        background: rgba(74, 144, 226, 0.1);
        border-radius: 8px;
        padding: 12px;
        margin-bottom: 24px;
        display: flex;
        justify-content: space-between;
        align-items: center;
        border: 1px solid rgba(74, 144, 226, 0.2);
    }

    .preview-label {
        font-size: 0.85em;
        font-weight: 600;
    }

    .preview-value {
        font-weight: 700;
        color: #4a90e2;
        font-size: 1.1em;
    }

    :global(.dark-mode) .preview-stats {
        background: rgba(74, 144, 226, 0.05);
    }

    .actions {
        display: flex;
        justify-content: center;
        gap: 12px;
    }

    button {
        padding: 10px 24px;
        border-radius: 8px;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s;
        border: none;
    }

    .submit-btn {
        background: #4a90e2;
        color: white;
    }

    .submit-btn:hover {
        background: #357abd;
        transform: translateY(-1px);
    }

    .cancel-btn {
        background: #ddd;
        color: #333;
    }

    .cancel-btn:hover {
        background: #ccc;
    }

    :global(.dark-mode) .cancel-btn {
        background: #555;
        color: #eee;
    }

    .form-loading-overlay {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(255, 255, 255, 0.4);
        backdrop-filter: blur(2px);
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 12px;
        z-index: 10;
    }

    :global(.dark-mode) .form-loading-overlay {
        background: rgba(0, 0, 0, 0.4);
    }

    .spinner {
        width: 30px;
        height: 30px;
        border: 3px solid rgba(74, 144, 226, 0.3);
        border-top: 3px solid #4a90e2;
        border-radius: 50%;
        animation: spin 1s linear infinite;
    }

    @keyframes spin {
        0% {
            transform: rotate(0deg);
        }
        100% {
            transform: rotate(360deg);
        }
    }
</style>
