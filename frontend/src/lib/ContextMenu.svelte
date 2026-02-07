<script lang="ts">
    import { fade } from "svelte/transition";

    interface MenuItem {
        label: string;
        action: () => void;
    }

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

    function handleItemClick(action: () => void) {
        action();
        onClose();
    }

    // Close menu when clicking outside
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
                <li
                    onclick={() => handleItemClick(item.action)}
                    onkeydown={(e) =>
                        (e.key === "Enter" || e.key === " ") &&
                        handleItemClick(item.action)}
                    role="menuitem"
                    tabindex="0"
                >
                    {item.label}
                </li>
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

    li:hover {
        background-color: #f0f0f0;
    }

    :global(.dark-mode) li:hover {
        background-color: #444;
    }
</style>
