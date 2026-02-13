<script lang="ts">
    // Receive measurement deltas and visibility state via props.
    let {
        deltaX,
        deltaY,
        visible,
        onClose,
    }: {
        deltaX: number;
        deltaY: number;
        visible: boolean;
        onClose: () => void;
    } = $props();

    // Convert a number into a human-readable string using localization or
    // exponential notation for very small values.
    function formatNumber(num: number): string {
        if (Math.abs(num) < 0.0001 && num !== 0) return num.toExponential(4);
        return num.toLocaleString(undefined, {
            maximumFractionDigits: 4,
            minimumFractionDigits: 0,
        });
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
            <h3>Measurement Results</h3>
            <div class="results-grid">
                <div class="label">Delta X (Time):</div>
                <div class="value">{formatNumber(deltaX)}</div>
                <div class="label">Delta Y (Value):</div>
                <div class="value">{formatNumber(deltaY)}</div>
            </div>
            <button class="close-btn" onclick={onClose}>Close</button>
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
        background: rgba(0, 0, 0, 0.5);
        display: flex;
        justify-content: center;
        align-items: center;
        z-index: 2000;
        backdrop-filter: blur(2px);
    }

    .modal-content {
        background: white;
        padding: 24px;
        border-radius: 8px;
        min-width: 300px;
        box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
        display: flex;
        flex-direction: column;
        gap: 16px;
    }

    :global(.dark-mode) .modal-content {
        background: #2d2d2d;
        color: #eee;
    }

    h3 {
        margin: 0;
        font-size: 1.2rem;
        color: #2a3f5f;
    }

    :global(.dark-mode) h3 {
        color: #fff;
    }

    .results-grid {
        display: grid;
        grid-template-columns: auto 1fr;
        gap: 12px 24px;
        align-items: center;
        background: #f8f9fa;
        padding: 16px;
        border-radius: 6px;
    }

    :global(.dark-mode) .results-grid {
        background: #1e1e1e;
    }

    .label {
        font-weight: 600;
        color: #506784;
    }

    :global(.dark-mode) .label {
        color: #a0a0a0;
    }

    .value {
        font-family: "JetBrains Mono", "Courier New", monospace;
        font-size: 1.1rem;
        color: #2a3f5f;
        text-align: right;
    }

    :global(.dark-mode) .value {
        color: #4db8ff;
    }

    .close-btn {
        align-self: flex-end;
        padding: 8px 16px;
        background: #2a3f5f;
        color: white;
        border: none;
        border-radius: 4px;
        cursor: pointer;
        font-weight: 600;
        transition: background 0.2s;
    }

    .close-btn:hover {
        background: #1e2e46;
    }

    :global(.dark-mode) .close-btn {
        background: #444;
    }

    :global(.dark-mode) .close-btn:hover {
        background: #555;
    }
</style>
