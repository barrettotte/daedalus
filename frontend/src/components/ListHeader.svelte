<script lang="ts">
  // List column header with inline title editing, card count, drag handle, and a three-dot menu for list operations.

  import { boardConfig, boardData, boardPath, addToast, saveWithToast, isAtLimit, listOrder } from "../stores/board";
  import { SaveListConfig, OpenFileExternal, SaveListOrder } from "../../wailsjs/go/main/App";
  import {
    getDisplayTitle, getCountDisplay, isOverLimit,
    formatListName, autoFocus,
  } from "../lib/utils";
  import {
    handleDragEnter, handleHeaderDragOver, handleDrop,
  } from "../lib/drag";
  import Icon from "./Icon.svelte";

  let {
    listKey,
    locked = false,
    pinState = null,
    hasLeftPin = false,
    hasRightPin = false,
    isLastList = false,
    oncreatecard,
    onfullcollapse,
    onhalfcollapse,
    onlock,
    onpinleft,
    onpinright,
    onunpin,
    onreload,
    onlistdragstart,
    onlistdragend,
    ondelete,
  }: {
    listKey: string;
    locked?: boolean;
    pinState?: "left" | "right" | null;
    hasLeftPin?: boolean;
    hasRightPin?: boolean;
    isLastList?: boolean;
    oncreatecard: () => void;
    onfullcollapse: () => void;
    onhalfcollapse: () => void;
    onlock: () => void;
    onpinleft: () => void;
    onpinright: () => void;
    onunpin: () => void;
    onreload: () => void;
    onlistdragstart: () => void;
    onlistdragend: () => void;
    ondelete: () => void;
  } = $props();

  let editingTitle = $state(false);
  let editingLimit = $state(false);
  let editTitleValue = $state("");
  let editLimitValue = $state(0);
  let confirmingDelete = $state(false);
  let menuOpen = $state(false);
  let menuRef: HTMLDivElement | undefined = $state();
  let menuFlip = $derived(pinState === 'right' || isLastList);
  let movingPosition = $state(false);
  let movePositionValue = $state(1);

  // Auto-cancel delete confirmation after 3 seconds.
  $effect(() => {
    if (confirmingDelete) {
      const timer = setTimeout(() => { confirmingDelete = false; }, 3000);
      return () => clearTimeout(timer);
    }
  });

  // Close menu when clicking outside.
  $effect(() => {
    if (!menuOpen) { return; }
    function handleClick(e: MouseEvent) {
      if (menuRef && !menuRef.contains(e.target as Node)) {
        menuOpen = false;
        movingPosition = false;
      }
    }
    document.addEventListener("click", handleClick, true);
    return () => document.removeEventListener("click", handleClick, true);
  });

  // Starts inline editing of the list title.
  function startEditTitle(): void {
    editingTitle = true;
    editTitleValue = getDisplayTitle(listKey, $boardConfig);
  }

  // Saves the edited title via backend and updates the config store.
  async function saveTitle(): Promise<void> {
    editingTitle = false;
    const cfg = $boardConfig[listKey] || { title: "", limit: 0 };
    const newTitle = editTitleValue.trim();
    const formatted = formatListName(listKey);

    // If the user typed the same as the formatted default, treat as empty (no override)
    const titleToSave = newTitle === formatted ? "" : newTitle;

    try {
      await SaveListConfig(listKey, titleToSave, cfg.limit || 0);

      boardConfig.update(c => {
        if (titleToSave === "" && (cfg.limit || 0) === 0) {
          delete c[listKey];
        } else {
          c[listKey] = { ...cfg, title: titleToSave };
        }
        return c;
      });
    } catch (e) {
      addToast(`Failed to save list title: ${e}`);
    }
  }

  // Starts inline editing of the list's card limit.
  function startEditLimit(): void {
    editingLimit = true;
    const cfg = $boardConfig[listKey];
    editLimitValue = cfg?.limit || 0;
  }

  // Saves the edited limit via backend and updates the config store.
  async function saveLimit(): Promise<void> {
    editingLimit = false;
    const cfg = $boardConfig[listKey] || { title: "", limit: 0 };
    const newLimit = Math.max(0, Math.floor(editLimitValue));

    try {
      await SaveListConfig(listKey, cfg.title || "", newLimit);

      boardConfig.update(c => {
        if ((cfg.title || "") === "" && newLimit === 0) {
          delete c[listKey];
        } else {
          c[listKey] = { ...cfg, limit: newLimit };
        }
        return c;
      });
    } catch (e) {
      addToast(`Failed to save list limit: ${e}`);
    }
  }

  // Handles keydown events on the title input.
  function handleTitleKeydown(e: KeyboardEvent): void {
    if (e.key === "Enter") {
      saveTitle();
    } else if (e.key === "Escape") {
      editingTitle = false;
    }
  }

  // Opens the list's directory in the system file explorer.
  function openInExplorer(): void {
    saveWithToast(OpenFileExternal($boardPath + "/" + listKey), "open folder");
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

  // Handles keydown events on the limit input.
  function handleLimitKeydown(e: KeyboardEvent): void {
    if (e.key === "Enter") {
      saveLimit();
    } else if (e.key === "Escape") {
      editingLimit = false;
    }
  }
</script>

<div class="list-header" role="group"
  ondragenter={handleDragEnter}
  ondragover={(e) => handleHeaderDragOver(e, listKey)}
  ondrop={(e) => handleDrop(e, listKey, onreload)}
>
  {#if !pinState}
    <div class="list-drag-handle" draggable="true" role="button" tabindex="0"
      aria-label="Drag to reorder list" ondragstart={(e) => {

        // WebKitGTK requires setData for the drop event to fire
        e.dataTransfer!.setData('text/plain', listKey);
        e.dataTransfer!.effectAllowed = 'move';

        const ghost = document.createElement('div');
        ghost.style.cssText = 'width:1px;height:1px;opacity:0';
        document.body.appendChild(ghost);

        e.dataTransfer!.setDragImage(ghost, 0, 0);
        requestAnimationFrame(() => document.body.removeChild(ghost));
        onlistdragstart();
      }}
      ondragend={onlistdragend}
      title="Drag to reorder"
    >
      <Icon name="drag-handle" size={10} />
    </div>
  {/if}
  {#if editingTitle}
    <input class="edit-title-input" type="text" bind:value={editTitleValue} onblur={saveTitle} onkeydown={handleTitleKeydown} use:autoFocus/>
  {:else}
    <button class="list-title-btn" title={locked ? "" : "Click to edit list name"} onclick={() => !locked && startEditTitle()}>
      {#if pinState}
        <span class="pin-icon"><Icon name="pin" size={11} /></span>
      {/if}
      {#if locked}
        <span class="lock-icon"><Icon name="lock" size={11} /></span>
      {/if}
      {getDisplayTitle(listKey, $boardConfig)}
    </button>
  {/if}
  <div class="header-right">
    <button class="collapse-btn" onclick={oncreatecard} title="Add card">
      <Icon name="plus" size={12} />
    </button>
    {#if editingLimit}
      <input class="edit-limit-input" type="number" min="0" bind:value={editLimitValue}
        onblur={saveLimit} onkeydown={handleLimitKeydown} use:autoFocus
      />
    {:else}
      <button class="count-btn" title="Click to edit card limit"
        class:at-limit={isAtLimit(listKey, $boardData, $boardConfig)}
        class:over-limit={isOverLimit(listKey, $boardData, $boardConfig)}
        onclick={startEditLimit}
      >
        {getCountDisplay(listKey, $boardData, $boardConfig)}
      </button>
    {/if}
    <div class="menu-wrapper" bind:this={menuRef}>
      <button class="collapse-btn" onclick={() => menuOpen = !menuOpen} title="More actions">
        <Icon name="menu-dots" size={12} />
      </button>
      {#if menuOpen}
        <div class="header-menu" class:menu-flip={menuFlip}>
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
          {#if movingPosition}
            <div class="menu-item move-position-row">
              <Icon name="move" size={12} />
              <span>Position</span>
              <input class="move-position-input" type="number" min="1" max={$listOrder.length}
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
          {#if confirmingDelete}
            <button class="menu-item menu-item-danger" title="Click to permanently delete this list and all cards" onclick={() => { menuOpen = false; ondelete(); }}>
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
    </div>
  </div>
</div>

<style lang="scss">
  .list-header {
    padding: 8px 10px;
    border-bottom: 1px solid var(--color-border-medium);
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 4px;
  }

  .list-drag-handle {
    cursor: grab;
    color: var(--color-text-muted);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    padding: 2px;
    border-radius: 3px;
    opacity: 0.5;

    &:hover {
      opacity: 1;
      color: var(--color-text-secondary);
      background: var(--overlay-hover);
    }

    &:active {
      cursor: grabbing;
    }
  }

  .menu-wrapper {
    position: relative;
  }

  .header-menu {
    position: absolute;
    top: 100%;
    left: 0;
    z-index: 100;

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
    font-size: 0.78rem;
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

  .header-right {
    display: flex;
    align-items: center;
    gap: 4px;
    flex-shrink: 0;
  }

  .collapse-btn {
    all: unset;
    cursor: pointer;
    color: var(--color-text-muted);
    display: flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    border-radius: 4px;

    &:hover {
      color: var(--color-text-secondary);
      background: var(--overlay-hover);
    }
  }

  .list-title-btn {
    all: unset;
    font-size: 0.85rem;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    cursor: pointer;
    color: inherit;
    flex: 1;
    text-align: left;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .lock-icon,
  .pin-icon {
    display: inline-flex;
    flex-shrink: 0;
    color: var(--color-text-muted);
  }

  .count-btn {
    all: unset;
    background: var(--color-border-medium);
    padding: 2px 8px;
    border-radius: 10px;
    font-family: var(--font-mono);
    font-size: 0.8rem;
    cursor: pointer;
    flex-shrink: 0;
    color: inherit;

    &.at-limit {
      background: var(--overlay-warning);
      color: var(--color-warning);
    }

    &.over-limit {
      background: var(--overlay-error-limit);
      color: #ff6b6b;
    }
  }

  .move-position-row {
    cursor: default;

    span {
      flex: 1;
    }
  }

  .move-position-input {
    background: var(--color-bg-inset);
    border: 1px solid var(--color-accent);
    color: var(--color-text-primary);
    font-family: var(--font-mono);
    font-size: 0.78rem;
    padding: 2px 4px;
    border-radius: 4px;
    outline: none;
    width: 40px;
    text-align: center;
  }

  .edit-title-input {
    background: var(--color-bg-inset);
    border: 1px solid var(--color-accent);
    color: var(--color-text-primary);
    font-size: 0.85rem;
    font-weight: 600;
    padding: 2px 6px;
    border-radius: 4px;
    outline: none;
    width: 100%;
    min-width: 0;
    margin-right: 8px;
  }

  .edit-limit-input {
    background: var(--color-bg-inset);
    border: 1px solid var(--color-accent);
    color: var(--color-text-primary);
    font-family: var(--font-mono);
    font-size: 0.8rem;
    padding: 2px 6px;
    border-radius: 10px;
    outline: none;
    width: 60px;
    text-align: center;
    flex-shrink: 0;
  }
</style>
