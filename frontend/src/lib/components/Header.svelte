<script lang="ts">
    import { appState } from "../state/app.svelte.ts";
    import * as ConfigService from "../../../bindings/olicanaplot/internal/appconfig/configservice";
</script>

<header class="main-header">
    <div
        class="logo"
        onclick={() => appState.resetToDefault()}
        role="button"
        tabindex="0"
        onkeydown={(e) => e.key === "Enter" && appState.resetToDefault()}
        oncontextmenu={(e) => appState.handleLogoContextMenu(e)}
        style="cursor: pointer;"
    >
        <svg
            viewBox="0 0 24 24"
            width="24"
            height="24"
            stroke="currentColor"
            stroke-width="2"
            fill="none"
            ><path d="M3 3v18h18" /><path d="M18 9l-5 5-2-2-4 4" /></svg
        >
        <span>OlicanaPlot</span>
    </div>

    <nav class="menu-bar">
        <button onclick={() => appState.toggleTheme()} title="Toggle Dark Mode">
            {#if appState.isDarkMode}
                <svg
                    viewBox="0 0 24 24"
                    width="16"
                    height="16"
                    stroke="currentColor"
                    stroke-width="2"
                    fill="none"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    ><circle cx="12" cy="12" r="5"></circle><line
                        x1="12"
                        y1="1"
                        x2="12"
                        y2="3"
                    ></line><line x1="12" y1="21" x2="12" y2="23"></line><line
                        x1="4.22"
                        y1="4.22"
                        x2="5.64"
                        y2="5.64"
                    ></line><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"
                    ></line><line x1="1" y1="12" x2="3" y2="12"></line><line
                        x1="21"
                        y1="12"
                        x2="23"
                        y2="12"
                    ></line><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"
                    ></line><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"
                    ></line></svg
                >
            {:else}
                <svg
                    viewBox="0 0 24 24"
                    width="16"
                    height="16"
                    stroke="currentColor"
                    stroke-width="2"
                    fill="none"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    ><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"
                    ></path></svg
                >
            {/if}
        </button>
        <button onclick={() => appState.loadFile()}>
            <svg
                viewBox="0 0 24 24"
                width="16"
                height="16"
                stroke="currentColor"
                stroke-width="2"
                fill="none"
                ><path
                    d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"
                /><polyline points="13 2 13 9 20 9" /></svg
            >
            Load File
        </button>
        <button
            onclick={(e) => appState.addFile(e)}
            title="Add data to current chart (Ctrl = add as subplots)"
        >
            <svg
                viewBox="0 0 24 24"
                width="16"
                height="16"
                stroke="currentColor"
                stroke-width="2"
                fill="none"
                ><path
                    d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"
                /><polyline points="13 2 13 9 20 9" /><line
                    x1="12"
                    y1="18"
                    x2="12"
                    y2="12"
                /><line x1="9" y1="15" x2="15" y2="15" /></svg
            >
            Add File
        </button>
        {#if appState.showGeneratorsMenu}
            <button onclick={(e) => appState.showGenerateMenu(e)}>
                <svg
                    viewBox="0 0 24 24"
                    width="16"
                    height="16"
                    stroke="currentColor"
                    stroke-width="2"
                    fill="none"
                    ><polyline points="22 12 18 12 15 21 9 3 6 12 2 12" /></svg
                >
                Generators
            </button>
        {/if}
        <button
            onclick={() => (ConfigService as any).OpenOptions()}
            title="Application Options"
        >
            <svg
                viewBox="0 0 24 24"
                width="16"
                height="16"
                stroke="currentColor"
                stroke-width="2"
                fill="none"
                stroke-linecap="round"
                stroke-linejoin="round"
            >
                <circle cx="12" cy="12" r="3"></circle>
                <path
                    d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"
                />
            </svg>
        </button>
    </nav>
</header>

<style>
    .main-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 0 20px;
        height: 50px;
        background-color: #f8f9fa;
        border-bottom: 1px solid #dee2e6;
        flex-shrink: 0;
    }

    :global(.dark-mode) .main-header {
        background-color: #2b2b2b;
        border-bottom-color: #444;
        color: #eee;
    }

    .logo {
        display: flex;
        align-items: center;
        gap: 10px;
        font-weight: bold;
        font-size: 1.2rem;
        color: #333;
    }

    :global(.dark-mode) .logo {
        color: #eee;
    }

    .menu-bar {
        display: flex;
        gap: 10px;
    }

    button {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 6px 12px;
        border: 1px solid #ced4da;
        background-color: white;
        border-radius: 4px;
        font-size: 0.9rem;
        cursor: pointer;
        transition: all 0.2s;
        color: #495057;
    }

    :global(.dark-mode) button {
        background-color: #3c3f41;
        border-color: #444;
        color: #bbb;
    }

    button:hover {
        background-color: #e9ecef;
        border-color: #adb5bd;
    }

    :global(.dark-mode) button:hover {
        background-color: #4b4d4f;
        border-color: #555;
        color: #eee;
    }
</style>
