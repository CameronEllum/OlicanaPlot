<script lang="ts">
    import { appState } from "../../state/app.svelte.ts";
</script>

{#if appState.pluginSelectionVisible}
    <div
        class="modal-backdrop"
        onclick={() => (appState.pluginSelectionVisible = false)}
        onkeydown={(e) => {
            if (e.key === "Escape" || e.key === "Enter")
                appState.pluginSelectionVisible = false;
        }}
        role="button"
        tabindex="-1"
        aria-label="Close selection modal"
    >
        <div
            class="modal-content"
            onclick={(e) => e.stopPropagation()}
            onkeydown={(e) => e.stopPropagation()}
            role="dialog"
            tabindex="-1"
        >
            <h3 class="text-gradient">Select Plugin</h3>
            <p>
                Multiple plugins can handle this file. Which one would you like
                to use?
            </p>
            <div class="candidate-list">
                {#each appState.pluginSelectionCandidates as plugin}
                    <button
                        class="candidate-item"
                        onclick={() => appState.handlePluginSelection(plugin)}
                    >
                        <span class="plugin-name">{plugin}</span>
                        <svg
                            viewBox="0 0 24 24"
                            width="16"
                            height="16"
                            stroke="currentColor"
                            stroke-width="2"
                            fill="none"
                        >
                            <path d="M5 12h14M12 5l7 7-7 7" />
                        </svg>
                    </button>
                {/each}
            </div>
            <div class="modal-footer">
                <button
                    class="btn btn-secondary"
                    onclick={() => (appState.pluginSelectionVisible = false)}
                    >Cancel</button
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
        backdrop-filter: blur(4px);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
    }

    .modal-content {
        background: var(--bg-secondary);
        border: 1px solid var(--border-color);
        padding: 30px;
        border-radius: 20px;
        max-width: 450px;
        width: 90%;
        box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
        animation: modalIn 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
    }

    @keyframes modalIn {
        from {
            transform: translateY(20px) scale(0.95);
            opacity: 0;
        }
        to {
            transform: translateY(0) scale(1);
            opacity: 1;
        }
    }

    h3 {
        margin-top: 0;
        margin-bottom: 12px;
        font-size: 1.5rem;
        font-weight: 800;
    }

    p {
        color: var(--text-secondary);
        margin-bottom: 24px;
        line-height: 1.5;
    }

    .candidate-list {
        display: flex;
        flex-direction: column;
        gap: 8px;
        margin-bottom: 24px;
    }

    .candidate-item {
        background: rgba(255, 255, 255, 0.05);
        border: 1px solid var(--border-color);
        border-radius: 12px;
        padding: 16px;
        color: var(--text-primary);
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: space-between;
        transition: all 0.2s;
        text-align: left;
        width: 100%;
        font-family: inherit;
    }

    .candidate-item:hover {
        background: rgba(99, 110, 250, 0.15);
        border-color: var(--accent);
        transform: translateX(4px);
    }

    .candidate-item .plugin-name {
        font-weight: 600;
    }

    .modal-footer {
        display: flex;
        justify-content: flex-end;
    }

    .btn {
        padding: 10px 20px;
        border-radius: 10px;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s;
        border: none;
        font-family: inherit;
    }

    .btn-secondary {
        background: rgba(0, 0, 0, 0.05);
        color: var(--text-primary);
        border: 1px solid var(--border-color);
    }

    :global(.dark-mode) .btn-secondary {
        background: rgba(255, 255, 255, 0.05);
    }

    .btn-secondary:hover {
        background: rgba(0, 0, 0, 0.1);
    }
</style>
