<script lang="ts">
    import { appState } from "../../state/app.svelte.ts";

    // Compute grid dimensions based on current subplots
    // Compute grid dimensions based on current subplots
    function getGridInfo() {
        const series = appState.currentSeriesData;
        if (series.length === 0) {
            return {
                rows: 1,
                cols: 1,
                cells: [{ row: 0, col: 0, series: [], isBlocked: false }],
                ghostColumnBlocked: [false],
            };
        }

        const maxRow = Math.max(0, ...series.map((s) => s.subplotRow || 0));
        const maxCol = Math.max(0, ...series.map((s) => s.subplotCol || 0));
        const rows = maxRow + 1;
        const cols = maxCol + 1;

        const cells = [];
        const ghostColumnBlocked = [];

        for (let r = 0; r < rows; r++) {
            let leftNeighborOccupied = true;
            for (let c = 0; c < cols; c++) {
                const cellSeries = series.filter(
                    (s) =>
                        (s.subplotRow || 0) === r && (s.subplotCol || 0) === c,
                );
                const isOccupied = cellSeries.length > 0;
                const isBlocked = !isOccupied && !leftNeighborOccupied;

                cells.push({ row: r, col: c, series: cellSeries, isBlocked });
                leftNeighborOccupied = isOccupied;
            }
            // A new column is only allowed if the last cell in this row is occupied
            ghostColumnBlocked.push(!leftNeighborOccupied);
        }

        return { rows, cols, cells, ghostColumnBlocked };
    }

    let grid = $derived(getGridInfo());
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
            class="modal-content glass-panel visual-grid-modal"
            onclick={(e) => e.stopPropagation()}
            onkeydown={(e) => e.stopPropagation()}
            role="dialog"
            tabindex="-1"
        >
            <div class="modal-header">
                <h3 class="text-gradient">Add to Plot</h3>
            </div>

            <p class="dialog-msg">Select where to place the new data:</p>

            <div
                class="grid-container"
                style="--rows: {grid.rows + 1}; --cols: {grid.cols + 1};"
            >
                <!-- Existing Subplots -->
                {#each grid.cells as cell}
                    <button
                        class="grid-cell {cell.series.length > 0
                            ? 'occupied'
                            : 'ghost'} {cell.isBlocked ? 'blocked' : ''}"
                        disabled={cell.isBlocked}
                        onclick={() =>
                            appState.handleAddFileChoice({
                                row: cell.row,
                                col: cell.col,
                            })}
                        style="grid-row: {cell.row +
                            1}; grid-column: {cell.col + 1};"
                        title={cell.isBlocked
                            ? "Cannot skip cells in a row"
                            : cell.series.length > 0
                              ? `Overlay on ${cell.series
                                    .map((s) => s.name)
                                    .join(", ")}`
                              : "Place in empty cell"}
                    >
                        {#if cell.series.length > 0}
                            <div class="cell-label">
                                {cell.series[0].name.substring(0, 15) +
                                    (cell.series[0].name.length > 15
                                        ? "..."
                                        : "")}
                            </div>
                            {#if cell.series.length > 1}
                                <div class="cell-count">
                                    +{cell.series.length - 1}
                                </div>
                            {/if}
                        {:else}
                            <div class="cell-label subtitle">
                                {cell.row === 0 && cell.col === 0
                                    ? "Main"
                                    : `(${cell.row}, ${cell.col})`}
                            </div>
                            {#if !cell.isBlocked}
                                <svg
                                    viewBox="0 0 24 24"
                                    width="16"
                                    height="16"
                                    stroke="currentColor"
                                    stroke-width="3"
                                    fill="none"
                                    class="mt-4"
                                >
                                    <line x1="12" y1="5" x2="12" y2="19"></line>
                                    <line x1="5" y1="12" x2="19" y2="12"></line>
                                </svg>
                            {/if}
                        {/if}
                    </button>
                {/each}

                <!-- Ghost Row Below (First column only per request) -->
                <button
                    class="grid-cell ghost"
                    onclick={() =>
                        appState.handleAddFileChoice({
                            row: grid.rows,
                            col: 0,
                        })}
                    style="grid-row: {grid.rows + 1}; grid-column: 1;"
                    title="Add as new row"
                >
                    <svg
                        viewBox="0 0 24 24"
                        width="16"
                        height="16"
                        stroke="currentColor"
                        stroke-width="3"
                        fill="none"
                    >
                        <line x1="12" y1="5" x2="12" y2="19"></line>
                        <line x1="5" y1="12" x2="19" y2="12"></line>
                    </svg>
                </button>

                <!-- Ghost Column Right -->
                {#each Array(grid.rows) as _, r}
                    <button
                        class="grid-cell ghost {grid.ghostColumnBlocked[r]
                            ? 'blocked'
                            : ''}"
                        disabled={grid.ghostColumnBlocked[r]}
                        onclick={() =>
                            appState.handleAddFileChoice({
                                row: r,
                                col: grid.cols,
                            })}
                        style="grid-row: {r + 1}; grid-column: {grid.cols + 1};"
                        title={grid.ghostColumnBlocked[r]
                            ? "Cannot skip columns"
                            : "Add as new column"}
                    >
                        {#if !grid.ghostColumnBlocked[r]}
                            <svg
                                viewBox="0 0 24 24"
                                width="16"
                                height="16"
                                stroke="currentColor"
                                stroke-width="3"
                                fill="none"
                            >
                                <line x1="12" y1="5" x2="12" y2="19"></line>
                                <line x1="5" y1="12" x2="19" y2="12"></line>
                            </svg>
                        {/if}
                    </button>
                {/each}
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
    .visual-grid-modal {
        width: 500px;
    }

    .grid-container {
        display: grid;
        grid-template-rows: repeat(var(--rows), 100px);
        grid-template-columns: repeat(var(--cols), 1fr);
        gap: 12px;
        margin-bottom: 24px;
        max-height: 50vh;
        overflow-y: auto;
        padding: 4px;
    }

    .grid-cell {
        border-radius: 12px;
        cursor: pointer;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
        font-family: inherit;
        position: relative;
        padding: 8px;
        border: 2px solid transparent;
        background: none;
    }

    .grid-cell.occupied {
        background: var(--bg-surface);
        border: 2px solid var(--border-color);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
    }

    :global(.dark-mode) .grid-cell.occupied {
        background: rgba(255, 255, 255, 0.05);
        border-color: var(--accent);
    }

    .grid-cell.occupied:hover {
        transform: scale(1.02);
        background: var(--accent-glow);
        border-color: var(--accent);
    }

    .grid-cell.ghost {
        background: transparent;
        border: 2px dashed var(--border-color);
        color: var(--text-secondary);
    }

    .grid-cell.ghost:hover {
        background: var(--accent-glow);
        border-color: var(--accent);
        color: var(--accent);
        transform: scale(1.02);
    }

    .grid-cell.blocked {
        opacity: 0.3;
        cursor: not-allowed;
    }

    .cell-label {
        font-size: 0.8rem;
        font-weight: 700;
        text-align: center;
        word-break: break-all;
        color: var(--text-primary);
    }

    .cell-label.subtitle {
        font-weight: 500;
        opacity: 0.7;
        margin-bottom: 4px;
    }

    .grid-cell svg {
        margin-top: 4px;
    }

    .cell-count {
        position: absolute;
        top: 6px;
        right: 6px;
        background: var(--accent);
        color: white;
        font-size: 0.7rem;
        padding: 2px 6px;
        border-radius: 10px;
        font-weight: 800;
    }
</style>
