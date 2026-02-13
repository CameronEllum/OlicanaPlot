<script lang="ts">
    import { fade } from "svelte/transition";

    // Define the structure for individual context menu items.
    interface MenuItem {
        label: string;
        action?: () => void; // Make action optional for headers
        header?: boolean;
    }

    // Receive configuration for positioning, visibility, and interaction handlers
    // via props.
    let {
        x,
        y,
        visible,
        items,
        onClose,
    }: {
        x: number;
        y: number;
        visible: boolean;
        items: MenuItem[];
        onClose: () => void;
    } = $props();

    let menuElement = $state<HTMLElement>();
    let adjustedX = $state(0);
    let adjustedY = $state(0);

    // Calculate and adjust the menu position to ensure it remains within the
    // viewport boundaries.
    $effect(() => {
        if (visible && menuElement) {
            const rect = menuElement.getBoundingClientRect();
            const padding = 10;

            let newX = x;
            let newY = y;

            if (newX + rect.width > window.innerWidth - padding) {
                newX = window.innerWidth - rect.width - padding;
            }
            if (newY + rect.height > window.innerHeight - padding) {
                newY = window.innerHeight - rect.height - padding;
            }

            adjustedX = Math.max(padding, newX);
            adjustedY = Math.max(padding, newY);
        }
    });

    // Execute the associated action and close the menu when an item is selected.
    function handleItemClick(action: () => void) {
        action();
        onClose();
    }

    // Dismiss the menu when the user clicks outside of the menu container.
    function handleWindowClick(e: MouseEvent) {
        const target = e.target as HTMLElement;
        if (visible && !target.closest(".context-menu")) {
            onClose();
        }
    }
</script>

<svelte:window onclick={handleWindowClick} oncontextmenu={handleWindowClick} />

{#if visible}
    <div
        bind:this={menuElement}
        class="context-menu"
        style="left: {adjustedX}px; top: {adjustedY}px;"
        oncontextmenu={(e) => e.preventDefault()}
        transition:fade={{ duration: 100 }}
        role="menu"
        tabindex="-1"
    >
        <ul>
            {#each items as item}
                {#if item.header}
                    <li class="header">
                        {item.label}
                    </li>
                {:else}
                    <li
                        onclick={() =>
                            item.action && handleItemClick(item.action)}
                        onkeydown={(e) =>
                            (e.key === "Enter" || e.key === " ") &&
                            item.action &&
                            handleItemClick(item.action)}
                        role="menuitem"
                        tabindex="0"
                    >
                        {item.label}
                    </li>
                {/if}
            {/each}
        </ul>
    </div>
{/if}

<style>
    .context-menu {
        position: fixed;
        z-index: 1000;
        background: white;
        border: 1px solid #ddd;
        box-shadow: 2px 2px 5px rgba(0, 0, 0, 0.1);
        border-radius: 4px;
        padding: 4px 0;
        min-width: 120px;
        background-color: #fff;
        color: #333;
    }

    :global(.dark-mode) .context-menu {
        background-color: #333;
        border-color: #444;
        color: #eee;
    }

    ul {
        list-style: none;
        margin: 0;
        padding: 0;
    }

    li {
        padding: 8px 12px;
        cursor: pointer;
        font-size: 14px;
        transition: background 0.2s;
    }

    li.header {
        cursor: default;
        font-weight: bold;
        border-bottom: 1px solid #eee;
        margin-bottom: 4px;
        padding-bottom: 6px;
        font-size: 12px;
        text-transform: uppercase;
        color: #666;
    }

    :global(.dark-mode) li.header {
        border-bottom-color: #444;
        color: #aaa;
    }

    li:hover {
        background-color: #f0f0f0;
    }

    :global(.dark-mode) li:hover {
        background-color: #444;
    }
</style>
