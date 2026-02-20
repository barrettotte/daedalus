<script lang="ts">
  // List column header with inline title editing, card count, drag handle, and a three-dot menu for list operations.

  import {
    boardConfig, boardData, addToast, saveWithToast,
    isAtLimit, listOrder,
  } from "../stores/board";
  import { SaveListConfig } from "../../wailsjs/go/main/App";
  import {
    getDisplayTitle, getCountDisplay, isOverLimit,
    formatListName, autoFocus, clickOutside,
  } from "../lib/utils";
  import {
    hideDefaultDragGhost, handleDragEnter, handleHeaderDragOver, handleDrop,
  } from "../lib/drag";
  import Icon from "./Icon.svelte";
  import CardIcon from "./CardIcon.svelte";
  import ListHeaderMenu from "./ListHeaderMenu.svelte";

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
  let menuOpen = $state(false);

  // Resets all menu state when clicking outside the menu wrapper.
  function closeMenu(): void {
    menuOpen = false;
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
    <input class="form-input edit-title-input" type="text" bind:value={editTitleValue}
      onblur={saveTitle} onkeydown={handleTitleKeydown} use:autoFocus
    />
  {:else}
    <button class="list-title-btn" title={locked ? "" : "Click to edit list name"}
      onclick={() => !locked && startEditTitle()}
    >
      {#if pinState}
        <span class="pin-icon"><Icon name="pin" size={11} /></span>
      {/if}
      {#if locked}
        <span class="lock-icon"><Icon name="lock" size={11} /></span>
      {/if}
      {#if icon}
        <span class="list-icon">
          <CardIcon name={icon} size={14} />
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
      <ListHeaderMenu {listKey} {locked} {color} {icon} {pinState} {hasLeftPin} {hasRightPin} {isLastList}
        bind:menuOpen
        {oncreatecard} {onfullcollapse} {onhalfcollapse} {onlock}
        {onpinleft} {onpinright} {onunpin}
        {onmoveallcards} {ondeleteallcards} {ondelete}
      />
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

  .list-icon {
    display: inline-flex;
    align-items: center;
    flex-shrink: 0;
    font-size: 0.9rem;
    line-height: 1;
  }
</style>
