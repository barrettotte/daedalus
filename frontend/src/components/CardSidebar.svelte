<script lang="ts">
  // Card detail sidebar. Shows move-to-list dropdown, labels, dates, counter, and timestamps.

  import {
    selectedCard, boardConfig, boardData, sortedListKeys, listOrder,
    moveCardInBoard, computeListOrder, addToast, isAtLimit, isLocked, labelColors,
  } from "../stores/board";
  import { MoveCard, LoadBoard } from "../../wailsjs/go/main/App";
  import { getDisplayTitle, formatDateTime, labelColor } from "../lib/utils";
  import type { daedalus } from "../../wailsjs/go/models";
  import CounterControl from "./CounterControl.svelte";
  import DateSection from "./DateSection.svelte";

  let {
    meta,
    moveDropdownOpen = $bindable(false),
    onsavecounter,
    onsavedates,
  }: {
    meta: daedalus.CardMetadata;
    moveDropdownOpen?: boolean;
    onsavecounter?: (counter: daedalus.Counter | null) => void;
    onsavedates?: (
      due: string | null,
      range: { start: string; end: string } | null,
    ) => void;
  } = $props();

  let dropdownEl: HTMLElement | undefined = $state();

  // Closes the move dropdown when clicking outside of it.
  function handleWindowClick(e: MouseEvent): void {
    if (moveDropdownOpen && dropdownEl && !dropdownEl.contains(e.target as Node)) {
      moveDropdownOpen = false;
    }
  }

  // Extracts the raw directory name from the selected card's file path.
  let cardListKey = $derived.by(() => {
    if (!$selectedCard) {
      return "";
    }
    const parts = $selectedCard.filePath.split("/");
    return parts[parts.length - 2] || "";
  });

  // Derives the list display name from config title or formatted directory name.
  let listDisplayName = $derived.by(() => {
    if (!cardListKey) {
      return "";
    }
    return getDisplayTitle(cardListKey, $boardConfig);
  });

  // Derives the card's 1-based position and list size from boardData.
  let cardPosition = $derived.by(() => {
    if (!$selectedCard || !cardListKey) {
      return "";
    }
    const cards = $boardData[cardListKey];
    if (!cards) {
      return "";
    }

    const idx = cards.findIndex(
      c => c.filePath === $selectedCard!.filePath,
    );
    if (idx === -1) {
      return "";
    }
    return `${idx + 1} / ${cards.length}`;
  });

  // Moves the current card to a different list, placing it at the top.
  async function moveToList(targetListKey: string): Promise<void> {
    if (!$selectedCard || targetListKey === cardListKey) {
      return;
    }
    if (isLocked(cardListKey, $boardConfig) || isLocked(targetListKey, $boardConfig)) {
      addToast("List is locked");
      return;
    }
    if (isAtLimit(targetListKey, $boardData, $boardConfig)) {
      addToast("List is at its card limit");
      return;
    }

    const targetCards = $boardData[targetListKey] || [];
    const targetIndex = 0;
    const newListOrder = computeListOrder(targetCards, targetIndex);
    const originalPath = $selectedCard.filePath;

    moveCardInBoard(originalPath, cardListKey, targetListKey, targetIndex, newListOrder);

    try {
      const result = await MoveCard(originalPath, targetListKey, newListOrder);

      // Sync the filePath in boardData so derived lookups match the new selectedCard.
      if (result.filePath !== originalPath) {
        boardData.update(lists => {
          const targetCards = lists[targetListKey];
          if (targetCards) {
            const idx = targetCards.findIndex(c => c.metadata.id === result.metadata.id);
            if (idx !== -1) {
              targetCards[idx] = {
                ...targetCards[idx],
                filePath: result.filePath,
                listName: result.listName,
              } as daedalus.KanbanCard;
            }
          }
          return lists;
        });
      }

      selectedCard.set(result);
    } catch (err) {
      addToast(`Failed to move card: ${err}`);
      const response = await LoadBoard("");
      boardData.set(response.lists);
    }
  }
</script>

<svelte:window onclick={handleWindowClick} />
<div class="sidebar">
  <div class="sidebar-section">
    <h4 class="sidebar-title">
      Card #{meta.id}
      {#if cardPosition}<span class="position-hint">{cardPosition}</span>{/if}
    </h4>
    <div class="move-dropdown" bind:this={dropdownEl}>
      <button class="move-trigger" title="Move card to a different list" onclick={() => moveDropdownOpen = !moveDropdownOpen}>
        <span>{listDisplayName}</span>
        <svg class="move-chevron" class:open={moveDropdownOpen} viewBox="0 0 16 16" width="12" height="12">
          <path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </button>
      {#if moveDropdownOpen}
        {@const sourceLocked = isLocked(cardListKey, $boardConfig)}
        <div class="move-menu">
          {#each sortedListKeys($boardData, $listOrder) as key}
            {@const locked = isLocked(key, $boardConfig)}
            {@const full = key !== cardListKey && isAtLimit(key, $boardData, $boardConfig)}
            {@const blocked = (key !== cardListKey && (full || locked))
              || (key === cardListKey && locked)
              || sourceLocked}
            <button class="move-option"
              class:active={key === cardListKey}
              class:disabled={blocked}
              disabled={blocked}
              onclick={() => { moveDropdownOpen = false; moveToList(key); }}
            >
              {getDisplayTitle(key, $boardConfig)}
              {#if locked}
                <span class="move-full">(locked)</span>
              {:else if full}
                <span class="move-full">(full)</span>
              {/if}
            </button>
          {/each}
        </div>
      {/if}
    </div>
  </div>

  {#if meta.labels && meta.labels.length > 0}
    <div class="sidebar-section">
      <h4 class="sidebar-title">Labels</h4>
      <div class="sidebar-labels">
        {#each meta.labels as label}
          <span class="label" title={label} style="background: {labelColor(label, $labelColors)}">
            {label}
          </span>
        {/each}
      </div>
    </div>
  {/if}

  <DateSection due={meta.due} range={meta.range} onsave={onsavedates} />
  <CounterControl counter={meta.counter} onsave={onsavecounter} />

  {#if meta.created}
    <div class="sidebar-section">
      <h4 class="sidebar-title">Created</h4>
      <div class="sidebar-value">{formatDateTime(meta.created)}</div>
    </div>
  {/if}

  <div class="sidebar-section">
    <h4 class="sidebar-title">Updated</h4>
    <div class="sidebar-value">
      {formatDateTime(meta.updated && meta.updated !== meta.created ? meta.updated : meta.created)}
    </div>
  </div>

</div>

<style lang="scss">
  .sidebar {
    flex: 0 0 200px;
    min-width: 0;
    overflow: hidden;
  }

  .position-hint {
    font-weight: 400;
    text-transform: none;
    letter-spacing: normal;
    float: right;
  }

  .sidebar-labels {
    display: flex;
    gap: 4px;
    flex-wrap: wrap;
  }

  .label {
    font-size: 0.7rem;
    font-weight: 600;
    padding: 4px 8px;
    border-radius: 3px;
    color: #fff;
    text-align: center;
    flex: 0 0 calc(50% - 2px);
    box-sizing: border-box;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .move-dropdown {
    position: relative;
  }

  .move-trigger {
    all: unset;
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 4px;
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    color: var(--color-text-primary);
    font-size: 0.8rem;
    padding: 4px 6px;
    border-radius: 4px;
    cursor: pointer;
    box-sizing: border-box;

    &:hover {
      border-color: var(--color-text-tertiary);
    }
  }

  .move-chevron {
    color: var(--color-text-tertiary);
    transition: transform 0.15s;
    flex-shrink: 0;

    &.open {
      transform: rotate(180deg);
    }
  }

  .move-menu {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    right: 0;
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: 4px;
    padding: 4px 0;
    z-index: 10;
    max-height: 200px;
    overflow-y: auto;
  }

  .move-option {
    all: unset;
    display: block;
    width: 100%;
    padding: 5px 8px;
    font-size: 0.8rem;
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
  }

  .move-full {
    font-size: 0.7rem;
    color: var(--color-text-muted);
  }

</style>
