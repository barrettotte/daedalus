<script lang="ts">
  // List column header with inline title editing, card count, drag handle, and a three-dot menu for list operations.

  import {
    boardConfig, boardData, boardPath, addToast, saveWithToast,
    isAtLimit, isLocked, listOrder, sortedListKeys,
  } from "../stores/board";
  import { SaveListConfig, OpenFileExternal, SaveListOrder } from "../../wailsjs/go/main/App";
  import {
    getDisplayTitle, getCountDisplay, isOverLimit,
    formatListName, autoFocus, clickOutside,
  } from "../lib/utils";
  import { hideDefaultDragGhost } from "../lib/drag";
  import {
    handleDragEnter, handleHeaderDragOver, handleDrop,
  } from "../lib/drag";
  import Icon from "./Icon.svelte";
  import CardIcon from "./CardIcon.svelte";
  import ColorPicker from "./ColorPicker.svelte";
  import { getIconNames } from "../lib/icons";
  import { isFileIcon } from "../lib/utils";

  let {
    listKey,
    locked = false,
    color = '',
    icon = '',
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
    onmoveallcards: (targetKey: string) => void;
    ondeleteallcards: () => void;
    ondelete: () => void;
  } = $props();

  let editingTitle = $state(false);
  let editingLimit = $state(false);
  let editTitleValue = $state("");
  let editLimitValue = $state(0);
  let confirmingDelete = $state(false);
  let confirmingDeleteAll = $state(false);
  let menuOpen = $state(false);
  let menuFlip = $derived(pinState === 'right' || isLastList);
  let movingPosition = $state(false);
  let movePositionValue = $state(1);
  let movingAllCards = $state(false);
  let colorPickerOpen = $state(false);
  let iconPickerOpen = $state(false);
  let iconFileNames: string[] = $state([]);
  let iconEmojiValue = $state("");

  // Saves a color for this list via the backend and updates the store.
  async function saveColor(hex: string): Promise<void> {
    const cfg = $boardConfig[listKey] || { title: "", limit: 0, locked: false, color: "", icon: "" };
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
    const cfg = $boardConfig[listKey] || { title: "", limit: 0, locked: false, color: "", icon: "" };
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
    iconEmojiValue = (icon && !isFileIcon(icon)) ? icon : "";
    iconPickerOpen = true;
    getIconNames().then(names => { iconFileNames = names; });
  }

  // Auto-cancel delete confirmation after 3 seconds.
  $effect(() => {
    if (confirmingDelete) {
      const timer = setTimeout(() => { confirmingDelete = false; }, 3000);
      return () => clearTimeout(timer);
    }
  });

  // Auto-cancel delete-all confirmation after 3 seconds.
  $effect(() => {
    if (confirmingDeleteAll) {
      const timer = setTimeout(() => {
        confirmingDeleteAll = false;
      }, 3000);

      return () => clearTimeout(timer);
    }
  });

  // Resets all menu state when clicking outside the menu wrapper.
  function closeMenu(): void {
    menuOpen = false;
    movingPosition = false;
    movingAllCards = false;
    colorPickerOpen = false;
    iconPickerOpen = false;
  }

  // Starts inline editing of the list title.
  function startEditTitle(): void {
    editingTitle = true;
    editTitleValue = getDisplayTitle(listKey, $boardConfig);
  }

  // Saves the edited title via backend and updates the config store.
  async function saveTitle(): Promise<void> {
    editingTitle = false;
    const cfg = $boardConfig[listKey] || { title: "", limit: 0, locked: false, color: "", icon: "" };
    const newTitle = editTitleValue.trim();
    const formatted = formatListName(listKey);

    // If the user typed the same as the formatted default, treat as empty (no override)
    const titleToSave = newTitle === formatted ? "" : newTitle;

    try {
      await SaveListConfig(listKey, titleToSave, cfg.limit || 0, cfg.color || "", cfg.icon || "");

      boardConfig.update(c => {
        if (titleToSave === "" && (cfg.limit || 0) === 0 && !(cfg.color)) {
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
    const cfg = $boardConfig[listKey] || { title: "", limit: 0, locked: false, color: "", icon: "" };
    const newLimit = Math.max(0, Math.floor(editLimitValue));

    try {
      await SaveListConfig(listKey, cfg.title || "", newLimit, cfg.color || "", cfg.icon || "");

      boardConfig.update(c => {
        if ((cfg.title || "") === "" && newLimit === 0 && !(cfg.color)) {
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

{#if color}
  <div class="accent-bar" style="background: {color}"></div>
{/if}
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

        hideDefaultDragGhost(e);
        onlistdragstart();
      }}
      ondragend={onlistdragend}
      title="Drag to reorder"
    >
      <Icon name="drag-handle" size={10} />
    </div>
  {/if}
  {#if editingTitle}
    <input class="form-input edit-title-input" type="text" bind:value={editTitleValue} onblur={saveTitle} onkeydown={handleTitleKeydown} use:autoFocus/>
  {:else}
    <button class="list-title-btn" title={locked ? "" : "Click to edit list name"} onclick={() => !locked && startEditTitle()}>
      {#if pinState}
        <span class="pin-icon"><Icon name="pin" size={11} /></span>
      {/if}
      {#if locked}
        <span class="lock-icon"><Icon name="lock" size={11} /></span>
      {/if}
      {#if icon}
        <span class="list-icon">
          {#if isFileIcon(icon)}
            <CardIcon name={icon} size={14} />
          {:else}
            {icon}
          {/if}
        </span>
      {/if}
      {getDisplayTitle(listKey, $boardConfig)}
    </button>
  {/if}
  <div class="header-right">
    <button class="btn-icon collapse-btn" onclick={oncreatecard} title="Add card">
      <Icon name="plus" size={12} />
    </button>
    {#if editingLimit}
      <input class="form-input edit-limit-input" type="number" min="0" bind:value={editLimitValue}
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
    <div class="menu-wrapper" use:clickOutside={closeMenu}>
      <button class="btn-icon collapse-btn" onclick={() => menuOpen = !menuOpen} title="More actions">
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
              <div class="emoji-row">
                <input class="form-input emoji-input" type="text" placeholder="Type emoji..."
                  bind:value={iconEmojiValue}
                  onkeydown={e => {
                    if (e.key === 'Enter') {
                      saveIcon(iconEmojiValue.trim());
                      iconPickerOpen = false;
                    }
                  }}
                />
                <button class="emoji-save-btn" onclick={() => { saveIcon(iconEmojiValue.trim()); iconPickerOpen = false; }}>
                  Set
                </button>
              </div>
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
            <button class="menu-item" title="Set an icon or emoji for this list" onclick={openIconPicker}>
              <span class="icon-indicator">
                {#if icon && isFileIcon(icon)}
                  <CardIcon name={icon} size={12} />
                {:else if icon}
                  {icon}
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
                  menuOpen = false; confirmingDeleteAll = false; ondeleteallcards();
                }}
              >
                Confirm delete all?
              </button>
            {:else}
              <button class="menu-item menu-item-danger" title="Remove all cards but keep the list"
                onclick={() => { confirmingDeleteAll = true; }}
              >
                <Icon name="trash" size={12} />
                Delete all cards
              </button>
            {/if}
          {/if}
          {#if confirmingDelete}
            <button class="menu-item menu-item-danger"
              title="Click to permanently delete this list and all cards"
              onclick={() => {
                menuOpen = false; ondelete();
              }}
            >
              Confirm delete?
            </button>
          {:else}
            <button class="menu-item menu-item-danger" title="Remove this list and all its cards"
              onclick={() => { confirmingDelete = true; }}
            >
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

  .header-right {
    display: flex;
    align-items: center;
    gap: 4px;
    flex-shrink: 0;
  }

  .collapse-btn {
    width: 22px;
    height: 22px;
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

  .edit-title-input {
    font-size: 0.85rem;
    font-weight: 600;
    padding: 2px 6px;
    width: 100%;
    min-width: 0;
    margin-right: 8px;
    border-color: var(--color-accent);
  }

  .edit-limit-input {
    font-family: var(--font-mono);
    font-size: 0.8rem;
    padding: 2px 6px;
    border-radius: 10px;
    width: 60px;
    text-align: center;
    flex-shrink: 0;
    border-color: var(--color-accent);
  }

  .accent-bar {
    height: 3px;
    border-radius: 8px 8px 0 0;
    flex-shrink: 0;
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

  .list-icon {
    display: inline-flex;
    align-items: center;
    flex-shrink: 0;
    font-size: 0.9rem;
    line-height: 1;
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
