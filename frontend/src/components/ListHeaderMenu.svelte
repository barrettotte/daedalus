<script lang="ts">
  // Context menu for list columns - color picker, icon picker, move/pin/delete actions.

  import {
    boardConfig, boardData, boardPath, addToast, saveWithToast,
    isLocked, listOrder, sortedListKeys,
  } from "../stores/board";
  import { SaveListConfig, OpenFileExternal, SaveListOrder } from "../../wailsjs/go/main/App";
  import { getDisplayTitle, autoFocus, getListConfig, joinPath } from "../lib/utils";
  import Icon from "./Icon.svelte";
  import CardIcon from "./CardIcon.svelte";
  import ColorPicker from "./ColorPicker.svelte";
  import { getIconNames } from "../lib/icons";

  let {
    listKey,
    locked = false,
    color = '',
    icon = '',
    pinState = null,
    hasLeftPin = false,
    hasRightPin = false,
    isLastList = false,
    menuOpen = $bindable(false),
    oncreatecard,
    onfullcollapse,
    onhalfcollapse,
    onlock,
    onpinleft,
    onpinright,
    onunpin,
    onmoveallcards,
    ondeleteallcards,
    ondelete,
  }: {
    listKey: string;
    locked?: boolean;
    color?: string;
    icon?: string;
    pinState?: "left" | "right" | null;
    hasLeftPin?: boolean;
    hasRightPin?: boolean;
    isLastList?: boolean;
    menuOpen: boolean;
    oncreatecard: () => void;
    onfullcollapse: () => void;
    onhalfcollapse: () => void;
    onlock: () => void;
    onpinleft: () => void;
    onpinright: () => void;
    onunpin: () => void;
    onmoveallcards: (targetKey: string) => void;
    ondeleteallcards: () => void;
    ondelete: () => void;
  } = $props();

  const CONFIRM_TIMEOUT_MS = 3000;

  let confirmingDelete = $state(false);
  let confirmingDeleteAll = $state(false);
  let menuFlip = $derived(pinState === 'right' || isLastList);
  let movingPosition = $state(false);
  let movePositionValue = $state(1);
  let movingAllCards = $state(false);
  let colorPickerOpen = $state(false);
  let iconPickerOpen = $state(false);
  let iconFileNames: string[] = $state([]);

  // Saves a color for this list via the backend and updates the store.
  async function saveColor(hex: string): Promise<void> {
    const cfg = getListConfig(listKey, $boardConfig);
    try {
      await SaveListConfig(listKey, cfg.title || "", cfg.limit || 0, hex, cfg.icon || "");
      boardConfig.update(c => {
        c[listKey] = { ...cfg, color: hex };
        return c;
      });
    } catch (e) {
      addToast(`Failed to save list color: ${e}`);
    }
  }

  // Saves an icon for this list via the backend and updates the store.
  async function saveIcon(value: string): Promise<void> {
    const cfg = getListConfig(listKey, $boardConfig);
    try {
      await SaveListConfig(listKey, cfg.title || "", cfg.limit || 0, cfg.color || "", value);
      boardConfig.update(c => {
        c[listKey] = { ...cfg, icon: value };
        return c;
      });
    } catch (e) {
      addToast(`Failed to save list icon: ${e}`);
    }
  }

  // Opens the icon picker panel and loads available file icons.
  function openIconPicker(): void {
    iconPickerOpen = true;
    getIconNames()
      .then(names => { iconFileNames = names; })
      .catch(err => {
        addToast(`Failed to load icons: ${err}`);
        iconPickerOpen = false;
      });
  }

  // Auto-cancel delete confirmations after 3 seconds.
  $effect(() => {
    if (!confirmingDelete && !confirmingDeleteAll) {
      return;
    }
    const timer = setTimeout(() => {
      confirmingDelete = false;
      confirmingDeleteAll = false;
    }, CONFIRM_TIMEOUT_MS);
    return () => clearTimeout(timer);
  });

  // Opens the list's directory in the system file explorer.
  function openInExplorer(): void {
    saveWithToast(OpenFileExternal(joinPath($boardPath, listKey)), "open folder");
  }

  // Shows the move-to-position input, pre-filled with the current 1-indexed position.
  function startMovePosition(): void {
    const currentIdx = $listOrder.indexOf(listKey);
    movePositionValue = currentIdx >= 0 ? currentIdx + 1 : 1;
    movingPosition = true;
  }

  // Moves this list to the entered position, clamping to valid range.
  function commitMovePosition(): void {
    movingPosition = false;
    menuOpen = false;

    const order = [...$listOrder];
    const srcIdx = order.indexOf(listKey);
    if (srcIdx === -1) {
      return;
    }

    const maxPos = order.length;
    const clamped = isNaN(movePositionValue) ? 1 : movePositionValue;
    const targetPos = Math.max(1, Math.min(maxPos, Math.floor(clamped)));
    const targetIdx = targetPos - 1;

    if (targetIdx === srcIdx) {
      return;
    }

    order.splice(srcIdx, 1);
    order.splice(targetIdx, 0, listKey);
    listOrder.set(order);
    saveWithToast(SaveListOrder(order), "save list order");
  }

  // Handles keydown events on the move position input.
  function handleMoveKeydown(e: KeyboardEvent): void {
    if (e.key === "Enter") {
      commitMovePosition();
    } else if (e.key === "Escape") {
      movingPosition = false;
    }
  }
</script>

{#if menuOpen}
  <div class="header-menu" class:menu-flip={menuFlip}>
    <button class="menu-item" title="Add a new card to this list" onclick={() => { menuOpen = false; oncreatecard(); }}>
      <Icon name="plus" size={12} />
      Add card
    </button>
    <div class="menu-divider"></div>
    <button class="menu-item" title="Show first 5 cards, minimize the rest" onclick={() => { menuOpen = false; onhalfcollapse(); }}>
      <Icon name="chevron-down" size={12} />
      Half collapse
    </button>
    <button class="menu-item" title="Collapse to a vertical title bar" onclick={() => { menuOpen = false; onfullcollapse(); }}>
      <Icon name="chevron-double-down" size={12} />
      Full collapse
    </button>
    <div class="menu-divider"></div>

    <button class="menu-item" title="Open this list's directory in your file manager" onclick={() => { menuOpen = false; openInExplorer(); }}>
      <Icon name="folder" size={12} />
      Open in explorer
    </button>
    <button class="menu-item" title={locked ? "Allow cards to be moved in and out" : "Prevent cards from being moved in or out"}
      onclick={() => { menuOpen = false; onlock(); }}
    >
      <Icon name={locked ? "lock-open" : "lock"} size={12} />
      {locked ? "Unlock list" : "Lock list"}
    </button>
    {#if colorPickerOpen}
      <div class="color-picker-panel">
        <ColorPicker bind:color onchange={saveColor} />
      </div>
    {:else}
      <button class="menu-item" title="Set a color accent for this list" onclick={() => { colorPickerOpen = true; }}>
        <span class="color-indicator" style={color ? `background: ${color}` : ''}></span>
        List color
      </button>
    {/if}
    {#if iconPickerOpen}
      <div class="icon-picker-panel">
        {#if iconFileNames.length > 0}
          <div class="icon-grid">
            {#each iconFileNames as name}
              <button class="icon-grid-option" class:active={name === icon} title={name} onclick={() => { saveIcon(name); iconPickerOpen = false; }}>
                <CardIcon name={name} size={16} />
              </button>
            {/each}
          </div>
        {/if}
        {#if icon}
          <button class="menu-item menu-item-danger" onclick={() => { saveIcon(""); iconPickerOpen = false; }}>
            <Icon name="trash" size={12} />
            Remove icon
          </button>
        {/if}
      </div>
    {:else}
      <button class="menu-item" title="Set an icon for this list" onclick={openIconPicker}>
        <span class="icon-indicator">
          {#if icon}
            <CardIcon name={icon} size={12} />
          {:else}
            <Icon name="image" size={12} />
          {/if}
        </span>
        List icon
      </button>
    {/if}
    {#if !locked && ($boardData[listKey]?.length || 0) > 0}
      {#if movingAllCards}
        <div class="menu-submenu">
          {#each sortedListKeys($boardData, $listOrder) as targetKey}
            {#if targetKey !== listKey && !isLocked(targetKey, $boardConfig)}
              <button class="menu-item" onclick={() => { menuOpen = false; movingAllCards = false; onmoveallcards(targetKey); }}>
                {getDisplayTitle(targetKey, $boardConfig)}
              </button>
            {/if}
          {/each}
        </div>
      {:else}
        <button class="menu-item" title="Move all cards from this list to another" onclick={() => { movingAllCards = true; }}>
          <Icon name="move" size={12} />
          Move all cards to...
        </button>
      {/if}
    {/if}
    {#if movingPosition}
      <div class="menu-item move-position-row">
        <Icon name="move" size={12} />
        <span>Position</span>
        <input class="form-input move-position-input" type="number" min="1" max={$listOrder.length}
          bind:value={movePositionValue} onblur={commitMovePosition} onkeydown={handleMoveKeydown} use:autoFocus
        />
      </div>
    {:else}
      <button class="menu-item" title="Move list to a specific position (1-{$listOrder.length})" onclick={startMovePosition}>
        <Icon name="move" size={12} />
        Move to position
      </button>
    {/if}
    <div class="menu-divider"></div>

    {#if pinState}
      <button class="menu-item" title="Return this list to the scrollable area" onclick={() => { menuOpen = false; onunpin(); }}>
        <Icon name="unpin" size={12} style="opacity: 0.6" />
        Unpin
      </button>
    {:else}
      {#if !hasLeftPin}
        <button class="menu-item" title="Pin this list to the left edge" onclick={() => { menuOpen = false; onpinleft(); }}>
          <Icon name="pin" size={12} style="transform: rotate(-20deg)" />
          Pin to left
        </button>
      {/if}
      {#if !hasRightPin}
        <button class="menu-item" title="Pin this list to the right edge" onclick={() => { menuOpen = false; onpinright(); }}>
          <Icon name="pin" size={12} style="transform: rotate(20deg)" />
          Pin to right
        </button>
      {/if}
    {/if}
    <div class="menu-divider"></div>

    {#if !locked && ($boardData[listKey]?.length || 0) > 0}
      {#if confirmingDeleteAll}
        <button class="menu-item menu-item-danger" title="Click to permanently delete all cards in this list"
          onclick={() => {
            menuOpen = false;
            confirmingDeleteAll = false;
            ondeleteallcards();
          }}
        >
          Confirm delete all?
        </button>
      {:else}
        <button class="menu-item menu-item-danger" title="Remove all cards but keep the list" onclick={() => { confirmingDeleteAll = true; }}>
          <Icon name="trash" size={12} />
          Delete all cards
        </button>
      {/if}
    {/if}
    {#if confirmingDelete}
      <button class="menu-item menu-item-danger" title="Click to permanently delete this list and all cards"
        onclick={() => {
          menuOpen = false;
          ondelete();
        }}
      >
        Confirm delete?
      </button>
    {:else}
      <button class="menu-item menu-item-danger" title="Remove this list and all its cards" onclick={() => { confirmingDelete = true; }}>
        <Icon name="trash" size={12} />
        Delete list
      </button>
    {/if}
  </div>
{/if}

<style lang="scss">
  .header-menu {
    position: absolute;
    top: 100%;
    left: 0;
    z-index: var(--z-dropdown);

    &.menu-flip {
      left: auto;
      right: 0;
    }
    background: var(--color-bg-surface);
    border: 1px solid var(--color-border-medium);
    border-radius: 6px;
    padding: 4px 0;
    min-width: 160px;
    box-shadow: var(--shadow-sm);
  }

  .menu-submenu {
    border-top: 1px solid var(--color-border-medium);
    border-bottom: 1px solid var(--color-border-medium);
    padding: 4px 0;
    max-height: 200px;
    overflow-y: auto;
  }

  .menu-divider {
    height: 1px;
    background: var(--color-border-medium);
    margin: 4px 0;
  }

  .menu-item {
    all: unset;
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 6px 12px;
    font-size: 0.8rem;
    color: var(--color-text-secondary);
    cursor: pointer;
    box-sizing: border-box;

    &:hover {
      background: var(--overlay-hover);
      color: var(--color-text-primary);
    }
  }

  .menu-item-danger {
    color: var(--color-error);

    &:hover {
      color: var(--color-error);
    }
  }

  .move-position-row {
    cursor: default;

    span {
      flex: 1;
    }
  }

  .move-position-input {
    font-family: var(--font-mono);
    font-size: 0.78rem;
    padding: 2px 4px;
    width: 40px;
    text-align: center;
    border-color: var(--color-accent);
  }

  .color-indicator {
    display: inline-block;
    width: 12px;
    height: 12px;
    border-radius: 3px;
    background: var(--color-text-muted);
    opacity: 0.5;
    flex-shrink: 0;
  }

  .color-picker-panel {
    padding: 6px 12px;
  }

  .icon-indicator {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 0.75rem;
    line-height: 1;
    flex-shrink: 0;
  }

  .icon-picker-panel {
    padding: 6px 12px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
</style>
