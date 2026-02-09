<script lang="ts">
  import { appState } from "./lib/state/app.svelte.ts";
  import Header from "./lib/components/Header.svelte";
  import ChartWrapper from "./lib/components/ChartWrapper.svelte";
  import GlobalContextMenu from "./lib/components/GlobalContextMenu.svelte";
  import PluginSelectionDialog from "./lib/components/dialogs/PluginSelectionDialog.svelte";
  import RenameDialog from "./lib/RenameDialog.svelte";

  // Synchronize the root document class with the dark mode state.
  $effect(() => {
    if (appState.isDarkMode) {
      document.documentElement.classList.add("dark-mode");
    } else {
      document.documentElement.classList.remove("dark-mode");
    }
  });
</script>

<RenameDialog
  visible={appState.renameVisible}
  title={appState.renameModalTitle}
  label={appState.renameModalLabel}
  value={appState.renameModalValue}
  onConfirm={(val) => {
    appState.renameVisible = false;
    appState.renameModalCallback(val);
  }}
  onCancel={() => (appState.renameVisible = false)}
/>

<div class="app-container" class:dark-mode={appState.isDarkMode}>
  <Header />

  <ChartWrapper />

  <footer class="status-bar">
    <span>{appState.loading ? "Loading..." : "Ready"}</span>
    <span>Data: {appState.dataSource}</span>
  </footer>

  <GlobalContextMenu />

  <PluginSelectionDialog />
</div>

<style>
  :global(html),
  :global(body),
  :global(#app) {
    margin: 0;
    padding: 0;
    width: 100%;
    height: 100%;
    overflow: hidden;
    background-color: var(--bg-primary);
    color: var(--text-primary);
    font-family: "Inter", sans-serif;
  }

  .app-container {
    height: 100vh;
    display: flex;
    flex-direction: column;
    background-color: var(--bg-primary);
  }

  .status-bar {
    height: 32px;
    background: var(--bg-secondary);
    border-top: 1px solid var(--border-color);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 20px;
    font-size: 0.8rem;
    color: var(--text-secondary);
    font-weight: 500;
  }
</style>
