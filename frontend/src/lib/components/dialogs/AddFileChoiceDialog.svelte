<script lang="ts">
    import { appState } from "../../state/app.svelte.ts";

    const seriesInCell = (r: number, c: number) =>
        appState.currentSeriesData.filter(
            (s) => s.subplot.row === r && s.subplot.col === c,
        );

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

        const maxRow = Math.max(0, ...series.map((s) => s.subplot.row));
        const maxCol = Math.max(0, ...series.map((s) => s.subplot.col));
        const rows = maxRow + 1;
        const cols = maxCol + 1;

        const cells = [];
        const ghostColumnBlocked = [];

        for (let r = 0; r < rows; r++) {
            let leftNeighborOccupied = true;
            for (let c = 0; c < cols; c++) {
                const cellSeries = seriesInCell(r, c);
                const isOccupied = cellSeries.length > 0;
                const isBlocked = !isOccupied && !leftNeighborOccupied;

                cells.push({ row: r, col: c, series: cellSeries, isBlocked });
                leftNeighborOccupied = isOccupied;
            }
            ghostColumnBlocked.push(!leftNeighborOccupied);
        }

        return { rows, cols, cells, ghostColumnBlocked };
    }

    let grid = $derived(getGridInfo());
</script>

{#snippet plusIcon()}
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
{/snippet}

{#snippet occupiedCell(series: any[])}
    <div class="cell-label">
        {series[0].name.substring(0, 15) +
            (series[0].name.length > 15 ? "..." : "")}
    </div>
    {#if series.length > 1}
        <div class="cell-count">+{series.length - 1}</div>
    {/if}
{/snippet}

{#snippet emptyCell(r: number, c: number, isBlocked: boolean)}
    <div class="cell-label subtitle">
        {r === 0 && c === 0 ? "Main" : `(${r}, ${c})`}
    </div>
    {#if !isBlocked}
        <div class="mt-4">{@render plusIcon()}</div>
    {/if}
{/snippet}

{#snippet gridButton(cell: any)}
    <button
        class="grid-cell {cell.isOccupied
            ? 'occupied'
            : 'ghost'} {cell.isBlocked ? 'blocked' : ''}"
        disabled={cell.isBlocked}
        onclick={() =>
            appState.handleAddFileChoice({ row: cell.row, col: cell.col })}
        style="grid-row: {cell.row + 1}; grid-column: {cell.col + 1};"
        title={cell.title}
    >
        {#if cell.isOccupied}
            {@render occupiedCell(cell.series)}
        {:else if cell.isGhostIcon}
            {@render plusIcon()}
        {:else}
            {@render emptyCell(cell.row, cell.col, cell.isBlocked)}
        {/if}
    </button>
{/snippet}

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
                <!-- Existing Cells -->
                {#each grid.cells as cell}
                    {@render gridButton({
                        ...cell,
                        isOccupied: cell.series.length > 0,
                        title: cell.isBlocked
                            ? "Cannot skip cells"
                            : cell.series.length > 0
                              ? "Overlay on series"
                              : "Place in empty cell",
                    })}
                {/each}

                <!-- New Row Button -->
                {@render gridButton({
                    row: grid.rows,
                    col: 0,
                    isGhostIcon: true,
                    title: "Add as new row",
                })}

                <!-- New Column Buttons -->
                {#each Array(grid.rows) as _, r}
                    {@render gridButton({
                        row: r,
                        col: grid.cols,
                        isBlocked: grid.ghostColumnBlocked[r],
                        isGhostIcon: !grid.ghostColumnBlocked[r],
                        title: grid.ghostColumnBlocked[r]
                            ? "Cannot skip columns"
                            : "Add as new column",
                    })}
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
