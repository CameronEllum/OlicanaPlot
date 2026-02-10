<script lang="ts">
    import { appState } from "../../state/app.svelte.ts";
</script>

{#if appState.addFileChoiceVisible}
    <div
        class="modal-backdrop"
        onclick={() => appState.handleAddFileChoice(null)}
        onkeydown={(e) => {
            if (e.key === "Escape") appState.handleAddFileChoice(null);
        }}
        role="button"
        tabindex="-1"
        aria-label="Close choice modal"
    >
        <div
            class="modal-content glass-panel"
            onclick={(e) => e.stopPropagation()}
            onkeydown={(e) => e.stopPropagation()}
            role="dialog"
            tabindex="-1"
        >
            <div class="modal-header">
                <h3 class="text-gradient">Add File</h3>
            </div>

            <p class="dialog-msg">Choose how to integrate the new data:</p>

            <div class="choice-list">
                <button
                    class="choice-item"
                    onclick={() => appState.handleAddFileChoice(true)}
                >
                    <div class="choice-icon">
                        <svg
                            viewBox="0 0 24 24"
                            width="24"
                            height="24"
                            stroke="currentColor"
                            stroke-width="2"
                            fill="none"
                        >
                            <rect
                                x="3"
                                y="3"
                                width="18"
                                height="18"
                                rx="2"
                                ry="2"
                            ></rect>
                            <line x1="3" y1="12" x2="21" y2="12"></line>
                        </svg>
                    </div>
                    <div class="choice-info">
                        <span class="choice-title">Add as Subplot</span>
                        <span class="choice-desc"
                            >Create a separate chart area below the current one</span
                        >
                    </div>
                    <div class="choice-arrow">
                        <svg
                            viewBox="0 0 24 24"
                            width="16"
                            height="16"
                            stroke="currentColor"
                            stroke-width="2.5"
                            fill="none"
                        >
                            <path d="M9 18l6-6-6-6" />
                        </svg>
                    </div>
                </button>

                <button
                    class="choice-item"
                    onclick={() => appState.handleAddFileChoice(false)}
                >
                    <div class="choice-icon">
                        <svg
                            viewBox="0 0 24 24"
                            width="24"
                            height="24"
                            stroke="currentColor"
                            stroke-width="2"
                            fill="none"
                        >
                            <path d="M3 3v18h18"></path>
                            <path d="M18 9l-5 5-2-2-4 4"></path>
                        </svg>
                    </div>
                    <div class="choice-info">
                        <span class="choice-title">Overlay on Current Axes</span
                        >
                        <span class="choice-desc"
                            >Plot data directly on top of the existing chart</span
                        >
                    </div>
                    <div class="choice-arrow">
                        <svg
                            viewBox="0 0 24 24"
                            width="16"
                            height="16"
                            stroke="currentColor"
                            stroke-width="2.5"
                            fill="none"
                        >
                            <path d="M9 18l6-6-6-6" />
                        </svg>
                    </div>
                </button>
            </div>

            <div class="modal-footer">
                <button
                    class="btn btn-secondary"
                    onclick={() => appState.handleAddFileChoice(null)}
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
        color: var(--text-primary);
    }

    .dialog-msg {
        color: var(--text-secondary);
        margin-bottom: 20px;
        font-size: 0.95rem;
    }

    .choice-list {
        display: flex;
        flex-direction: column;
        gap: 12px;
        margin-bottom: 20px;
    }

    .choice-item {
        background: var(--bg-secondary);
        border: 1px solid var(--border-color);
        border-radius: 12px;
        padding: 16px;
        color: var(--text-primary);
        cursor: pointer;
        display: flex;
        align-items: center;
        gap: 16px;
        transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
        text-align: left;
        width: 100%;
        font-family: inherit;
        position: relative;
        overflow: hidden;
    }

    .choice-item:hover {
        background: var(--accent-glow);
        border-color: var(--accent);
        transform: translateY(-2px);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    }

    .choice-icon {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 44px;
        height: 44px;
        background: var(--bg-primary);
        border-radius: 10px;
        color: var(--accent);
        flex-shrink: 0;
        border: 1px solid var(--border-color);
    }

    .choice-item:hover .choice-icon {
        background: var(--accent);
        color: white;
        border-color: var(--accent);
    }

    .choice-info {
        display: flex;
        flex-direction: column;
        gap: 2px;
        flex-grow: 1;
    }

    .choice-title {
        font-weight: 700;
        font-size: 1.05rem;
        color: var(--text-primary);
    }

    .choice-desc {
        font-size: 0.8rem;
        color: var(--text-secondary);
        line-height: 1.3;
    }

    .choice-arrow {
        opacity: 0;
        transform: translateX(-10px);
        transition: all 0.2s ease;
        color: var(--accent);
    }

    .choice-item:hover .choice-arrow {
        opacity: 1;
        transform: translateX(0);
    }

    :global(.dark-mode) .choice-item {
        background: rgba(255, 255, 255, 0.03);
    }

    :global(.dark-mode) .choice-item:hover {
        background: rgba(99, 102, 241, 0.15);
    }

    .modal-footer {
        display: flex;
        justify-content: flex-end;
    }

    .btn {
        padding: 10px 24px;
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
        border-color: var(--text-secondary);
    }
</style>
