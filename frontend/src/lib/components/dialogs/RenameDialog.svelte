<script lang="ts">
    // Receive configuration for the rename operation via props.
    let {
        title = "Rename",
        label = "New Name:",
        value = "",
        visible,
        onConfirm,
        onCancel,
    }: {
        title?: string;
        label?: string;
        value?: string;
        visible: boolean;
        onConfirm: (newValue: string) => void;
        onCancel: () => void;
    } = $props();

    let inputValue = $state("");
    let inputElement = $state<HTMLInputElement>();

    // Synchronize the local input state with the provided value when the dialog
    // becomes visible.
    $effect(() => {
        if (visible) {
            inputValue = value;
            // Focus the input field automatically for a better user experience.
            setTimeout(() => inputElement?.focus(), 50);
        }
    });

    // Handle the confirmation action, ensuring the value has changed.
    function handleConfirm() {
        if (inputValue !== value) {
            onConfirm(inputValue);
        } else {
            onCancel();
        }
    }

    // Process keyboard shortcuts for submission and cancellation.
    function handleKeyDown(e: KeyboardEvent) {
        if (e.key === "Enter") {
            handleConfirm();
        } else if (e.key === "Escape") {
            onCancel();
        }
    }
</script>

{#if visible}
    <div
        class="modal-backdrop"
        onclick={onCancel}
        onkeydown={(e) => e.key === "Escape" && onCancel()}
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
            <h3>{title}</h3>
            <div class="form-group">
                <label for="rename-input">{label}</label>
                <input
                    id="rename-input"
                    type="text"
                    bind:value={inputValue}
                    bind:this={inputElement}
                    onkeydown={handleKeyDown}
                    autocomplete="off"
                />
            </div>
            <div class="actions">
                <button class="btn btn-secondary" onclick={onCancel}
                    >Cancel</button
                >
                <button class="btn btn-primary" onclick={handleConfirm}
                    >OK</button
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
        background: rgba(0, 0, 0, 0.4);
        display: flex;
        justify-content: center;
        align-items: center;
        z-index: 2000;
        backdrop-filter: blur(2px);
    }

    .modal-content {
        background: white;
        padding: 24px;
        border-radius: 12px;
        min-width: 320px;
        box-shadow: 0 10px 25px rgba(0, 0, 0, 0.2);
        display: flex;
        flex-direction: column;
        gap: 20px;
        border: 1px solid #ddd;
    }

    :global(.dark-mode) .modal-content {
        background: #2d2d2d;
        border-color: #444;
        color: #eee;
    }

    h3 {
        margin: 0;
        font-size: 1.25rem;
        font-weight: 600;
        color: #1a1a1a;
    }

    :global(.dark-mode) h3 {
        color: #fff;
    }

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    label {
        font-size: 0.9rem;
        font-weight: 500;
        color: #666;
    }

    :global(.dark-mode) label {
        color: #aaa;
    }

    input {
        padding: 10px 12px;
        border: 1.5px solid #ddd;
        border-radius: 6px;
        font-size: 1rem;
        transition: all 0.2s;
        background: #fff;
        color: #333;
    }

    :global(.dark-mode) input {
        background: #1e1e1e;
        border-color: #444;
        color: #eee;
    }

    input:focus {
        outline: none;
        border-color: #636efa;
        box-shadow: 0 0 0 3px rgba(99, 110, 250, 0.2);
    }

    .actions {
        display: flex;
        justify-content: flex-end;
        gap: 12px;
    }

    .btn {
        padding: 8px 20px;
        border-radius: 6px;
        font-size: 0.95rem;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s;
        border: none;
    }

    .btn-secondary {
        background: #f0f0f0;
        color: #444;
    }

    :global(.dark-mode) .btn-secondary {
        background: #444;
        color: #eee;
    }

    .btn-secondary:hover {
        background: #e5e5e5;
    }

    :global(.dark-mode) .btn-secondary:hover {
        background: #555;
    }

    .btn-primary {
        background: #636efa;
        color: white;
    }

    .btn-primary:hover {
        background: #4f58ca;
        transform: translateY(-1px);
    }

    .btn-primary:active {
        transform: translateY(0);
    }
</style>
