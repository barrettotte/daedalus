<script lang="ts">
  import {
    selectedCard, boardConfig, boardData, sortedListKeys,
    moveCardInBoard, computeListOrder, addToast, isAtLimit,
  } from "../stores/board";
  import { MoveCard, LoadBoard } from "../../wailsjs/go/main/App";
  import {
    formatListName, formatDate, formatDateTime, labelColor,
  } from "../lib/utils";
  import type { daedalus } from "../../wailsjs/go/models";

  let {
    meta,
    moveDropdownOpen = $bindable(false),
  }: {
    meta: daedalus.CardMetadata;
    moveDropdownOpen?: boolean;
  } = $props();

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
    return getListDisplayName(cardListKey);
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

  // Whether the due date has passed.
  let isOverdue = $derived(
    meta.due ? new Date(meta.due) < new Date() : false,
  );

  // Counter completion percentage.
  let counterPct = $derived(
    meta.counter && meta.counter.max > 0
      ? (meta.counter.current / meta.counter.max) * 100
      : 0,
  );

  // Resolves a list key to its display name via config title or formatted dir name.
  function getListDisplayName(listKey: string): string {
    const cfg = $boardConfig[listKey];
    if (cfg && cfg.title) {
      return cfg.title;
    }
    return formatListName(listKey);
  }

  // Moves the current card to a different list, placing it at the top.
  async function moveToList(targetListKey: string): Promise<void> {
    if (!$selectedCard || targetListKey === cardListKey) {
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
      selectedCard.set(result);
    } catch (err) {
      addToast(`Failed to move card: ${err}`);
      const response = await LoadBoard("");
      boardData.set(response.lists);
    }
  }
</script>

<div class="sidebar">
  <div class="sidebar-section">
    <h4 class="sidebar-title">Card</h4>
    <div class="sidebar-value">#{meta.id}</div>
  </div>

  <div class="sidebar-section">
    <h4 class="sidebar-title">List</h4>
    <div class="move-dropdown">
      <button class="move-trigger" onclick={() => moveDropdownOpen = !moveDropdownOpen}>
        <span>{listDisplayName}</span>
        <svg class="move-chevron" class:open={moveDropdownOpen} viewBox="0 0 16 16" width="12" height="12">
          <path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </button>
      {#if moveDropdownOpen}
        <div class="move-menu">
          {#each sortedListKeys($boardData) as key}
            {@const full = key !== cardListKey
              && isAtLimit(key, $boardData, $boardConfig)}
            <button class="move-option" class:active={key === cardListKey} class:disabled={full}
              disabled={full} onclick={() => { moveDropdownOpen = false; moveToList(key); }}
            >
              {getListDisplayName(key)}
              {#if full} <span class="move-full">(full)</span>{/if}
            </button>
          {/each}
        </div>
      {/if}
    </div>
  </div>

  {#if cardPosition}
    <div class="sidebar-section">
      <h4 class="sidebar-title">Position</h4>
      <div class="sidebar-value">{cardPosition}</div>
    </div>
  {/if}

  {#if meta.labels && meta.labels.length > 0}
    <div class="sidebar-section">
      <h4 class="sidebar-title">Labels</h4>
      <div class="sidebar-labels">
        {#each meta.labels as label}
          <span class="label" style="background: {labelColor(label)}">
            {label}
          </span>
        {/each}
      </div>
    </div>
  {/if}

  {#if meta.due}
    <div class="sidebar-section">
      <h4 class="sidebar-title">Due Date</h4>
      <div class="sidebar-badge" class:overdue={isOverdue} class:on-time={!isOverdue}>
        <svg class="badge-icon" viewBox="0 0 24 24">
          <circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="2"/>
          <polyline points="12 6 12 12 16 14" fill="none" stroke="currentColor" stroke-width="2"/>
        </svg>
        {formatDate(meta.due)}
      </div>
    </div>
  {/if}

  {#if meta.range}
    <div class="sidebar-section">
      <h4 class="sidebar-title">Date Range</h4>
      <div class="sidebar-value">
        {formatDate(meta.range.start)} &ndash; {formatDate(meta.range.end)}
      </div>
    </div>
  {/if}

  {#if meta.counter}
    <div class="sidebar-section">
      <h4 class="sidebar-title">{meta.counter.label || "Counter"}</h4>
      <div class="counter-value">
        {meta.counter.current} / {meta.counter.max}
      </div>
      <div class="progress-bar sidebar-progress">
        <div class="progress-fill" class:complete={counterPct === 100} style="width: {counterPct}%"></div>
      </div>
    </div>
  {/if}

  {#if meta.created}
    <div class="sidebar-section">
      <h4 class="sidebar-title">Created</h4>
      <div class="sidebar-value">{formatDateTime(meta.created)}</div>
    </div>
  {/if}

  {#if meta.updated && meta.updated !== meta.created}
    <div class="sidebar-section">
      <h4 class="sidebar-title">Updated</h4>
      <div class="sidebar-value">{formatDateTime(meta.updated)}</div>
    </div>
  {/if}
</div>

<style lang="scss">
  .sidebar {
    flex: 0 0 168px;
  }

  .sidebar-section {
    background: var(--overlay-hover-light);
    border-radius: 6px;
    padding: 10px 12px;
    margin-bottom: 8px;
  }

  .sidebar-title {
    font-size: 0.7rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-text-tertiary);
    margin: 0 0 6px 0;
  }

  .sidebar-value {
    font-size: 0.8rem;
    color: #b6c2d1;
  }

  .sidebar-labels {
    display: flex;
    gap: 4px;
    flex-wrap: wrap;
  }

  .label {
    font-size: 0.7rem;
    font-weight: 600;
    padding: 4px 0;
    border-radius: 3px;
    color: #fff;
    text-align: center;
    flex: 0 0 calc(50% - 2px);
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

  .sidebar-badge {
    font-size: 0.8rem;
    font-weight: 600;
    padding: 4px 8px;
    border-radius: 3px;
    display: inline-flex;
    align-items: center;
    gap: 6px;

    &.on-time {
      background: var(--overlay-success-strong);
      color: var(--color-success);
    }

    &.overdue {
      background: var(--overlay-error-strong);
      color: var(--color-error);
    }
  }

  .badge-icon {
    width: 14px;
    height: 14px;
  }

  .counter-value {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--color-text-primary);
    margin-bottom: 6px;
  }

  .progress-bar {
    height: 6px;
    background: var(--color-border);
    border-radius: 3px;
    overflow: hidden;
    margin-bottom: 8px;
    max-width: 100%;
    box-sizing: border-box;
  }

  .progress-fill {
    height: 100%;
    background: var(--color-accent);
    border-radius: 4px;
    transition: width 0.3s;

    &.complete {
      background: var(--color-success);
    }
  }

  .sidebar-progress {
    margin-bottom: 0;
  }
</style>
