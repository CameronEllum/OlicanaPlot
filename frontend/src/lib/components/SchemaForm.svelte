<script lang="ts">
    import MapPicker from "./MapPicker.svelte";
    import { onMount } from "svelte";
    import { Events } from "@wailsio/runtime";

    // Define structural interfaces for JSON schema and UI schema mapping.
    interface SchemaProperty {
        type?: string;
        title?: string;
        description?: string;
        default?: any;
        minimum?: number;
        maximum?: number;
        step?: number;
        enum?: any[];
        oneOf?: any[];
        format?: string;
        items?: {
            type?: string;
            title?: string;
            enum?: any[];
            oneOf?: any[];
        };
    }

    interface Schema {
        properties?: Record<string, SchemaProperty>;
    }

    interface UiSchemaOptions {
        scale?: "log10";
    }

    interface UiSchema {
        "ui:order"?: string[];
        [key: string]: any;
    }

    // Receive schema, initial data, and callback handlers via props.
    let {
        schema = $bindable({}),
        uiSchema = $bindable({}),
        initialData = {},
        title = "Configuration",
        requestID = "",
        handleFormChange = false,
        onsubmit,
        oncancel,
    }: {
        schema?: Schema;
        uiSchema?: UiSchema;
        initialData?: any;
        title?: string;
        requestID?: string;
        handleFormChange?: boolean;
        onsubmit?: (data: any) => void;
        oncancel?: () => void;
    } = $props();

    let formData = $state<any>({});
    let loading = $state(false);
    let loadingTimer: number | null = null;

    // Synchronize form data with initial values and schema defaults upon component
    // mounting.
    onMount(() => {
        const newData = { ...initialData };
        if (schema && schema.properties) {
            Object.keys(schema.properties).forEach((key) => {
                const prop = schema.properties![key];
                if (newData[key] === undefined) {
                    newData[key] =
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
        formData = newData;
    });

    // Register an event listener for remote form updates from the host.
    onMount(() => {
        if (!requestID) return;

        const unsub = Events.On(`ipc-form-update-${requestID}`, (e) => {
            const update = (e.data || e) as {
                schema?: Schema;
                uiSchema?: UiSchema;
                data?: any;
            };
            if (update.schema) schema = update.schema;
            if (update.uiSchema) uiSchema = update.uiSchema;
            if (
                update.data &&
                typeof update.data === "object" &&
                !Array.isArray(update.data)
            ) {
                Object.assign(formData, update.data);
            }
            stopLoading();
        });
        return unsub;
    });

    // Activate the loading spinner after a short delay.
    function startLoading() {
        if (loadingTimer) clearTimeout(loadingTimer);
        loadingTimer = setTimeout(() => {
            loading = true;
        }, 250);
    }

    // Cancel any pending loading timer and hide the spinner.
    function stopLoading() {
        if (loadingTimer) {
            clearTimeout(loadingTimer);
            loadingTimer = null;
        }
        loading = false;
    }

    // Capture form state changes and emit update events to the host with
    // debouncing.
    let changeTimer: number | null = null;
    let isFirstRun = true;
    $effect(() => {
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

    // Invoke the submission callback with the current form data.
    function handleSubmit() {
        if (onsubmit) onsubmit(formData);
    }

    // Signal the cancellation handler.
    function handleCancel() {
        if (oncancel) oncancel();
    }

    // Map a literal value to its corresponding position on a logarithmic or linear
    // slider.
    function getSliderValue(key: string, val: number) {
        const ui = (uiSchema || {})[key] || {};
        if (ui["ui:options"]?.scale === "log10") {
            return Math.log10(val || 1);
        }
        return val || 0;
    }

    // Convert a slider position back to its literal numerical value based on the
    // configured scale.
    function setSliderValue(key: string, val: number) {
        const ui = (uiSchema || {})[key] || {};
        if (ui["ui:options"]?.scale === "log10") {
            formData[key] = Math.pow(10, val);
        } else {
            formData[key] = val;
        }
    }

    // Format a numerical value for display, applying unit suffixes for large
    // logarithmic values.
    function formatValue(key: string, val: number) {
        const ui = (uiSchema || {})[key] || {};
        if (ui["ui:options"]?.scale === "log10") {
            if (val >= 0.95 * 1000000) return (val / 1000000).toFixed(1) + "M";
            if (val >= 0.95 * 1000) return (val / 1000).toFixed(0) + "K";
            return Math.round(val);
        }
        return val;
    }

    // Inform the host of form container height changes via a resize observer.
    let container = $state<HTMLElement | null>(null);
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

<div class="schema-form-root" bind:this={container}>
    <header class="modal-header">
        <h3 class="text-gradient">{title}</h3>
    </header>

    <div class="form-content {uiSchema?.['ui:classNames'] || ''}">
        {#if schema && schema.properties}
            {#each uiSchema?.["ui:order"] || Object.keys(schema.properties) as key}
                {@const prop = schema.properties[key]}
                {#if prop}
                    {@const ui = (uiSchema || {})[key] || {}}

                    <div class="form-group {ui?.['ui:classNames'] || ''}">
                        <div class="label-row">
                            <label for={key}>{prop.title || key}</label>
                            {#if ui["ui:widget"] === "range"}
                                <span class="slider-value"
                                    >{formatValue(key, formData[key])}</span
                                >
                            {/if}
                        </div>

                        {#if ui["ui:widget"] === "map-picker"}
                            <MapPicker
                                lat={formData[key]?.lat}
                                lng={formData[key]?.lng}
                                markers={Array.isArray(formData[key])
                                    ? formData[key]
                                    : undefined}
                                onSelect={(lat, lng) => {
                                    if (!Array.isArray(formData[key])) {
                                        formData[key] = { lat, lng };
                                    }
                                }}
                            />
                        {:else if prop.enum || prop.oneOf}
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
                                {:else if prop.enum}
                                    {#each prop.enum as option}
                                        <option value={option}>{option}</option>
                                    {/each}
                                {/if}
                            </select>
                        {:else if prop.type === "array" && (prop.items?.enum || prop.items?.oneOf)}
                            <div class="checkbox-group">
                                {#each prop.items!.oneOf || prop.items!.enum!.map( (v) => ({ const: v, title: v }), ) as option}
                                    {@const val =
                                        option.const !== undefined
                                            ? option.const
                                            : option}
                                    <label class="checkbox-label">
                                        <input
                                            type="checkbox"
                                            checked={(
                                                formData[key] || []
                                            ).includes(val)}
                                            onchange={(e) => {
                                                const current =
                                                    formData[key] || [];
                                                const target =
                                                    e.target as HTMLInputElement;
                                                if (target.checked) {
                                                    formData[key] = [
                                                        ...current,
                                                        val,
                                                    ];
                                                } else {
                                                    formData[key] =
                                                        current.filter(
                                                            (v: any) =>
                                                                v !== val,
                                                        );
                                                }
                                            }}
                                        />
                                        {option.title || val}
                                    </label>
                                {/each}
                            </div>
                        {:else if prop.type === "array" && prop.items?.type === "string" && !prop.items?.enum && !prop.items?.oneOf}
                            <div class="repeater-group">
                                {#each formData[key] || [] as item, i}
                                    <div class="repeater-item">
                                        <input
                                            type="text"
                                            value={formData[key][i]}
                                            onchange={(e) => {
                                                const target =
                                                    e.target as HTMLInputElement;
                                                formData[key][i] = target.value;
                                            }}
                                            placeholder={prop.items.title ||
                                                "Item"}
                                        />
                                        <button
                                            class="btn-icon remove"
                                            title="Remove item"
                                            onclick={() => {
                                                formData[key] = (
                                                    formData[key] || []
                                                ).filter(
                                                    (_: any, index: number) =>
                                                        index !== i,
                                                );
                                            }}
                                        >
                                            <svg
                                                width="14"
                                                height="14"
                                                viewBox="0 0 24 24"
                                                fill="none"
                                                stroke="currentColor"
                                                stroke-width="2"
                                                stroke-linecap="round"
                                                stroke-linejoin="round"
                                                ><line
                                                    x1="18"
                                                    y1="6"
                                                    x2="6"
                                                    y2="18"
                                                ></line><line
                                                    x1="6"
                                                    y1="6"
                                                    x2="18"
                                                    y2="18"
                                                ></line></svg
                                            >
                                        </button>
                                    </div>
                                {/each}
                                <button
                                    class="btn-text"
                                    onclick={() => {
                                        formData[key] = [
                                            ...(formData[key] || []),
                                            "",
                                        ];
                                    }}
                                >
                                    + Add Item
                                </button>
                            </div>
                        {:else if prop.type === "integer" || prop.type === "number"}
                            {#if ui["ui:widget"] === "range"}
                                <div class="slider-group">
                                    <input
                                        type="range"
                                        id={key}
                                        min={ui["ui:options"]?.scale === "log10"
                                            ? Math.log10(prop.minimum || 1)
                                            : prop.minimum || 0}
                                        max={ui["ui:options"]?.scale === "log10"
                                            ? Math.log10(prop.maximum || 100)
                                            : prop.maximum || 100}
                                        step={ui["ui:options"]?.scale ===
                                        "log10"
                                            ? 0.1
                                            : prop.step || 1}
                                        value={getSliderValue(
                                            key,
                                            formData[key],
                                        )}
                                        oninput={(e) => {
                                            const target =
                                                e.target as HTMLInputElement;
                                            setSliderValue(
                                                key,
                                                parseFloat(target.value),
                                            );
                                        }}
                                    />
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
                        {:else if ui["ui:widget"] === "date" || prop.format === "date"}
                            <input
                                type="date"
                                id={key}
                                bind:value={formData[key]}
                            />
                        {:else if ui["ui:widget"] === "datetime-local" || prop.format === "date-time"}
                            <input
                                type="datetime-local"
                                id={key}
                                bind:value={formData[key]}
                            />
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
                {/if}
            {/each}
        {/if}
    </div>

    {#if formData.order !== undefined && formData.multiplier !== undefined}
        <div class="preview-stats">
            <span class="preview-label">Points per Series</span>
            <span class="preview-value">
                {Math.round(
                    formData.multiplier * Math.pow(10, formData.order),
                ).toLocaleString()}
            </span>
        </div>
    {/if}

    <div class="modal-footer">
        <button class="btn btn-secondary" onclick={handleCancel}>Cancel</button>
        <button class="btn btn-primary" onclick={handleSubmit}>OK</button>
    </div>

    {#if loading}
        <div class="form-loading-overlay">
            <div class="spinner"></div>
        </div>
    {/if}
</div>

<style>
    .schema-form-root {
        padding: 12px 14px;
        width: 100%;
        background: var(--bg-primary);
        color: var(--text-primary);
        display: flex;
        flex-direction: column;
        overflow: hidden;
    }

    .form-content {
        display: flex;
        flex-direction: column;
        gap: 6px;
        flex: 1;
    }

    .description {
        font-size: 0.75rem;
        color: var(--text-secondary);
        margin: -2px 0 0 0;
    }

    .checkbox-group {
        display: flex;
        flex-direction: column;
        gap: 6px;
        padding: 10px 14px;
        background: rgba(0, 0, 0, 0.2);
        border-radius: 8px;
        border: 1px solid var(--border-color);
        max-height: 160px;
        overflow-y: auto;
    }

    .checkbox-label {
        display: flex;
        align-items: center;
        gap: 12px;
        font-size: 0.95rem;
        cursor: pointer;
        transition: color 0.2s;
    }

    .repeater-group {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    .repeater-item {
        display: flex;
        align-items: center;
        gap: 6px;
    }

    .repeater-item input {
        flex: 1;
    }

    .btn-icon.remove {
        background: transparent;
        border: none;
        color: var(--text-secondary);
        cursor: pointer;
        padding: 6px;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 6px;
        transition: all 0.2s;
    }

    .btn-icon.remove:hover {
        background: rgba(239, 68, 68, 0.1);
        color: var(--error);
    }

    .btn-text {
        background: transparent;
        border: 1px dashed var(--border-color);
        color: var(--text-secondary);
        cursor: pointer;
        padding: 6px 12px;
        border-radius: 8px;
        font-size: 0.8rem;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        transition: all 0.2s;
        align-self: flex-start;
        margin-top: 2px;
    }

    .btn-text:hover {
        background: rgba(255, 255, 255, 0.05);
        color: var(--text-primary);
        border-color: var(--text-secondary);
    }

    .checkbox-label:hover {
        color: #fff;
    }

    .checkbox-label input {
        width: 18px;
        height: 18px;
        accent-color: var(--accent);
    }

    .label-row {
        display: flex;
        justify-content: space-between;
        align-items: baseline;
    }

    .slider-group {
        display: flex;
        flex-direction: column;
        gap: 2px;
    }

    .slider-value {
        font-size: 0.8rem;
        font-weight: 700;
        color: var(--accent);
        font-variant-numeric: tabular-nums;
    }

    .preview-stats {
        background: rgba(99, 102, 241, 0.08);
        border-radius: 8px;
        padding: 6px 12px;
        margin-top: 8px;
        margin-bottom: 2px;
        display: flex;
        justify-content: space-between;
        align-items: center;
        border: 1px solid rgba(99, 102, 241, 0.15);
    }

    .preview-label {
        font-size: 0.8rem;
        font-weight: 600;
        color: var(--text-secondary);
    }

    .preview-value {
        font-weight: 700;
        color: var(--accent);
        font-size: 1rem;
    }

    .form-loading-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.5);
        backdrop-filter: blur(4px);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 5000;
    }

    .spinner {
        width: 40px;
        height: 40px;
        border: 4px solid rgba(99, 102, 241, 0.2);
        border-top: 4px solid var(--accent);
        border-radius: 50%;
        animation: spin 1s linear infinite;
    }
</style>
