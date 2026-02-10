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
            <div class="selection-list">
                {#each appState.pluginSelectionCandidates as plugin}
                    <button
                        class="selection-item"
                        onclick={() => appState.handlePluginSelection(plugin)}
                    >
                        <span class="item-primary">{plugin}</span>
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
    p {
        color: var(--text-secondary);
        margin-bottom: 24px;
        line-height: 1.5;
        font-size: 0.95rem;
    }
</style>
