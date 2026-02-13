<script lang="ts">
    import { onMount, onDestroy } from "svelte";
    import { appState } from "../state/app.svelte.ts";
    import MeasurementResult from "./MeasurementResult.svelte";

    let chartContainer = $state<HTMLElement>();
    let resizeObserver: ResizeObserver | null = null;

    onMount(() => {
        if (chartContainer) {
            appState.initChart(chartContainer);

            resizeObserver = new ResizeObserver(() => {
                appState.chartAdapter?.resize();
            });
            resizeObserver.observe(chartContainer);
        }
    });

    onDestroy(() => {
        if (resizeObserver) resizeObserver.disconnect();
        if (appState.chartAdapter) appState.chartAdapter.destroy();
    });
</script>

<main class="content-area">
    <div class="chart-container" bind:this={chartContainer}></div>

    <MeasurementResult
        visible={appState.measurementResult !== null}
        deltaX={appState.measurementResult?.dx || 0}
        deltaY={appState.measurementResult?.dy || 0}
        onClose={() => {
            appState.measurementResult = null;
        }}
    />
</main>

<style>
    .content-area {
        flex: 1;
        background-color: var(--bg-primary);
        padding: 0;
        overflow: hidden;
        position: relative;
        min-height: 0;
    }

    .chart-container {
        width: 100%;
        height: 100%;
    }
</style>
