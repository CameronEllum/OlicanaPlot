<script lang="ts">
    import ContextMenu from "../ContextMenu.svelte";
    import { appState } from "../state/app.svelte.ts";
</script>

<ContextMenu
    x={appState.menuX}
    y={appState.menuY}
    visible={appState.menuVisible}
    items={appState.menuItems}
    onClose={() => (appState.menuVisible = false)}
/>
