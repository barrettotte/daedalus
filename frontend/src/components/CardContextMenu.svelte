<script lang="ts">
  // Right-click context menu for cards. Provides Edit, Move, and Delete actions.

  import {
    contextMenu, selectedCard, draftListKey, openInEditMode, boardData, boardConfig,
    sortedListKeys, listOrder, moveCardInBoard, computeListOrder,
    removeCardFromBoard, addToast, isAtLimit, isLocked,
  } from "../stores/board";
  import { MoveCard, DeleteCard, LoadBoard } from "../../wailsjs/go/main/App";
  import type { daedalus } from "../../wailsjs/go/models";
  import { getDisplayTitle, clickOutside } from "../lib/utils";
  import Icon from "./Icon.svelte";

  let moveSubmenuOpen = $state(false);
  let confirmingDelete = $state(false);

  let menu = $derived($contextMenu);

  // Sorted list keys for the move submenu.
  let listKeys = $derived(sortedListKeys($boardData, $listOrder));

  // Close context menu when any modal opens.
  $effect(() => {
    if ($selectedCard || $draftListKey) {
      contextMenu.set(null);
    }
  });

  function close(): void {
    moveSubmenuOpen = false;
    confirmingDelete = false;
    contextMenu.set(null);
  }

  // Opens the card detail modal in edit mode.
  function editCard(): void {
    if (!menu) {
      return;
    }
    openInEditMode.set(true);
    selectedCard.set(menu.card);
    close();
  }

  // Moves the card to a different list, placing it at the top.
  async function moveToList(targetListKey: string): Promise<void> {
    if (!menu || targetListKey === menu.listKey) {
      return;
    }

    const targetCards = $boardData[targetListKey] || [];
    const targetIndex = 0;
    const newListOrder = computeListOrder(targetCards, targetIndex);
    const originalPath = menu.card.filePath;

    moveCardInBoard(originalPath, menu.listKey, targetListKey, targetIndex, newListOrder);
    close();

    try {
      const result = await MoveCard(originalPath, targetListKey, newListOrder);

      // Sync filePath if it changed (cross-list move renames the file on disk).
      if (result.filePath !== originalPath) {
        boardData.update(lists => {
          const tc = lists[targetListKey];
          if (tc) {
            const idx = tc.findIndex(c => c.metadata.id === result.metadata.id);
            if (idx !== -1) {
              tc[idx] = { ...tc[idx], filePath: result.filePath, listName: result.listName } as daedalus.KanbanCard;
            }
          }
          return lists;
        });
      }
      addToast("Card moved", "success");
    } catch (err) {
      addToast(`Failed to move card: ${err}`);
      const response = await LoadBoard("");
      boardData.update(() => response.lists);
    }
  }

  // Deletes the card from disk and board.
  async function deleteCard(): Promise<void> {
    if (!menu) {
      return;
    }
    const filePath = menu.card.filePath;
    close();

    try {
      await DeleteCard(filePath);
      removeCardFromBoard(filePath);
      addToast("Card deleted", "success");
    } catch (e) {
      addToast(`Failed to delete card: ${e}`);
    }
  }

  // Close on Escape.
  function handleKeydown(e: KeyboardEvent): void {
    if (menu && e.key === "Escape") {
      e.stopPropagation();
      close();
    }
  }

  // Clamp menu position so it doesn't overflow the viewport.
  let style = $derived.by(() => {
    if (!menu) {
      return "";
    }
    const menuWidth = 160;
    const menuHeight = 120;
    const x = menu.x + menuWidth > window.innerWidth ? menu.x - menuWidth : menu.x;
    const y = menu.y + menuHeight > window.innerHeight ? menu.y - menuHeight : menu.y;
    return `left: ${x}px; top: ${y}px;`;
  });
</script>

<svelte:window onkeydown={handleKeydown} />

{#if menu}
  <div class="context-menu" use:clickOutside={close} style={style}>
    <button class="ctx-item" onclick={editCard}>
      <Icon name="pencil" size={14} /> Edit
    </button>

    <button class="ctx-item has-submenu" onclick={() => moveSubmenuOpen = !moveSubmenuOpen}>
      <Icon name="move" size={14} /> Move to...
    </button>
    {#if moveSubmenuOpen}
      <div class="ctx-submenu">
        {#each listKeys as key}
          {@const isCurrent = key === menu.listKey}
          {@const locked = isLocked(key, $boardConfig)}
          {@const full = !isCurrent && isAtLimit(key, $boardData, $boardConfig)}
          {@const blocked = isCurrent || locked || full}
          <button class="ctx-item" class:active={isCurrent} class:disabled={blocked} disabled={blocked}
            title={getDisplayTitle(key, $boardConfig)} onclick={() => moveToList(key)}
          >
            <span class="ctx-label">{getDisplayTitle(key, $boardConfig)}</span>
            {#if locked}<span class="ctx-hint">(locked)</span>{/if}
            {#if full}<span class="ctx-hint">(full)</span>{/if}
          </button>
        {/each}
      </div>
    {/if}

    <div class="ctx-separator"></div>

    {#if confirmingDelete}
      <div class="ctx-confirm">
        <span class="ctx-confirm-text">Delete?</span>
        <button class="ctx-confirm-btn delete" onclick={deleteCard}>Yes</button>
        <button class="ctx-confirm-btn" onclick={() => confirmingDelete = false}>No</button>
      </div>
    {:else}
      <button class="ctx-item delete" onclick={() => confirmingDelete = true}>
        <Icon name="trash" size={14} /> Delete
      </button>
    {/if}
  </div>
{/if}

<style lang="scss">
  .context-menu {
    position: fixed;
    z-index: 1000;
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: 6px;
    padding: 4px 0;
    min-width: 160px;
    box-shadow: var(--shadow-md);
  }

  .ctx-item {
    all: unset;
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 6px 12px;
    font-size: 0.82rem;
    color: var(--color-text-primary);
    cursor: pointer;
    box-sizing: border-box;

    &:hover:not(:disabled) {
      background: var(--overlay-hover);
    }

    &.active {
      color: var(--color-accent);
    }

    &.disabled {
      color: var(--color-text-muted);
      cursor: not-allowed;
    }

    &.delete {
      color: var(--color-error);
    }

    &.has-submenu::after {
      content: "\25B8";
      margin-left: auto;
      font-size: 0.7rem;
      color: var(--color-text-muted);
    }
  }

  .ctx-submenu {
    border-top: 1px solid var(--color-border);
    border-bottom: 1px solid var(--color-border);
    padding: 2px 0;
    max-height: 200px;
    overflow-y: auto;

    .ctx-item {
      padding-left: 20px;
      font-size: 0.78rem;
    }
  }

  .ctx-separator {
    height: 1px;
    background: var(--color-border);
    margin: 4px 0;
  }

  .ctx-label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }

  .ctx-hint {
    font-size: 0.7rem;
    color: var(--color-text-muted);
    margin-left: auto;
    flex-shrink: 0;
    white-space: nowrap;
  }

  .ctx-confirm {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 12px;
  }

  .ctx-confirm-text {
    font-size: 0.82rem;
    color: var(--color-error);
    font-weight: 600;
    margin-right: auto;
  }

  .ctx-confirm-btn {
    all: unset;
    font-size: 0.78rem;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 3px;
    cursor: pointer;
    color: var(--color-text-secondary);

    &:hover {
      background: var(--overlay-hover);
    }

    &.delete {
      color: var(--color-error);
    }
  }
</style>
