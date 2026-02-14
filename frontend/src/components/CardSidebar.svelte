<script lang="ts">
  import {
    selectedCard, boardConfig, boardData, sortedListKeys, listOrder,
    moveCardInBoard, computeListOrder, addToast, isAtLimit, labelColors,
  } from "../stores/board";
  import { MoveCard, LoadBoard } from "../../wailsjs/go/main/App";
  import {
    formatListName, formatDateTime, labelColor, toLocalISO,
  } from "../lib/utils";
  import type { daedalus } from "../../wailsjs/go/models";
  import DatePicker from "./DatePicker.svelte";

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

  // Counter settings panel state
  let counterSettingsOpen = $state(false);
  let editLabel = $state("");
  let editStart = $state(0);
  let editMax = $state(0);
  let editStep = $state(1);

  // Closes counter settings when the selected card changes.
  $effect(() => {
    $selectedCard;
    counterSettingsOpen = false;
  });

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

  // Counter completion percentage
  let counterPct = $derived.by(() => {
    if (!meta.counter) {
      return 0;
    }
    const { current, max, start } = meta.counter;
    const range = max - start;

    if (range === 0) {
      return 0;
    }
    const pct = ((current - start) / range) * 100;
    return Math.max(0, Math.min(100, pct));
  });

  // Resolves a list key to its display name via config title or formatted dir name.
  function getListDisplayName(listKey: string): string {
    const cfg = $boardConfig[listKey];
    if (cfg && cfg.title) {
      return cfg.title;
    }
    return formatListName(listKey);
  }

  // Whether the counter counts down (start > max).
  let countingDown = $derived(meta.counter ? (meta.counter.start || 0) > meta.counter.max : false);

  // Whether the counter has reached its start or goal bound.
  let atStart = $derived.by(() => {
    if (!meta.counter) {
      return false;
    }
    return meta.counter.current === (meta.counter.start || 0);
  });
  let atGoal = $derived.by(() => {
    if (!meta.counter) {
      return false;
    }
    return meta.counter.current === meta.counter.max;
  });

  // Increments or decrements the counter's current value by step and saves.
  function adjustCounter(delta: number): void {
    if (!meta.counter || !onsavecounter) {
      return;
    }
    const step = meta.counter.step || 1;
    const lo = Math.min(meta.counter.start || 0, meta.counter.max);
    const hi = Math.max(meta.counter.start || 0, meta.counter.max);
    const next = Math.max(lo, Math.min(hi, meta.counter.current + delta * step));

    if (next === meta.counter.current) {
      return;
    }
    const updated = { ...meta.counter, current: next };
    onsavecounter(updated as daedalus.Counter);
  }

  // Opens the counter settings panel, populating edit fields from current values.
  function openCounterSettings(): void {
    if (meta.counter) {
      editLabel = meta.counter.label || "";
      editStart = meta.counter.start || 0;
      editMax = meta.counter.max;
      editStep = meta.counter.step || 1;
    }
    counterSettingsOpen = true;
  }

  // Saves counter settings from the edit fields.
  function saveCounterSettings(): void {
    if (!meta.counter || !onsavecounter) {
      return;
    }
    const lo = Math.min(editStart, editMax);
    const hi = Math.max(editStart, editMax);
    const cur = meta.counter.current;
    const needsReset = cur < lo || cur > hi;

    const updated = {
      ...meta.counter,
      label: editLabel,
      start: editStart,
      max: editMax,
      step: Math.max(1, editStep || 1),
      current: needsReset ? editStart : cur,
    };

    onsavecounter(updated as daedalus.Counter);
    counterSettingsOpen = false;
  }

  // Adds a new default counter to the card.
  function addCounter(): void {
    if (!onsavecounter) {
      return;
    }
    onsavecounter({ current: 0, max: 10, start: 0, label: "" } as daedalus.Counter);
  }

  // Removes the counter from the card.
  function removeCounter(): void {
    if (!onsavecounter) {
      return;
    }
    onsavecounter(null);
    counterSettingsOpen = false;
  }

  // Adds a single date (due) set to now.
  function addDate(): void {
    if (!onsavedates) {
      return;
    }
    onsavedates(toLocalISO(new Date()), null);
  }

  // Removes all dates (due and range).
  function removeAllDates(): void {
    if (!onsavedates) {
      return;
    }
    onsavedates(null, null);
  }

  // Handles due date selection from the date picker.
  function onDueDateSelect(iso: string): void {
    if (!onsavedates) {
      return;
    }
    onsavedates(iso, null);
  }

  // Promotes a single due date to a range by adding an end date (start + 1 day).
  function addEndDate(): void {
    if (!onsavedates || !meta.due) {
      return;
    }
    const start = meta.due;
    const dt = new Date(start);
    dt.setDate(dt.getDate() + 1);
    const end = toLocalISO(dt);
    onsavedates(null, { start, end });
  }

  // Demotes a range back to a single due date (keeps the start date).
  function removeEndDate(): void {
    if (!onsavedates || !meta.range) {
      return;
    }
    onsavedates(meta.range.start, null);
  }

  // Handles range date selection from the date picker, swapping if start > end.
  function onRangeDateSelect(field: "start" | "end", iso: string): void {
    if (!onsavedates || !meta.range) {
      return;
    }
    let start = field === "start" ? iso : meta.range.start;
    let end = field === "end" ? iso : meta.range.end;
    if (start > end) {
      [start, end] = [end, start];
    }
    onsavedates(null, { start, end });
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
    <h4 class="sidebar-title">
      Card #{meta.id}
      {#if cardPosition}<span class="position-hint">{cardPosition}</span>{/if}
    </h4>
    <div class="move-dropdown">
      <button class="move-trigger" title="Move card to a different list" onclick={() => moveDropdownOpen = !moveDropdownOpen}>
        <span>{listDisplayName}</span>
        <svg class="move-chevron" class:open={moveDropdownOpen} viewBox="0 0 16 16" width="12" height="12">
          <path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </button>
      {#if moveDropdownOpen}
        <div class="move-menu">
          {#each sortedListKeys($boardData, $listOrder) as key}
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

  {#if meta.labels && meta.labels.length > 0}
    <div class="sidebar-section">
      <h4 class="sidebar-title">Labels</h4>
      <div class="sidebar-labels">
        {#each meta.labels as label}
          <span class="label" style="background: {labelColor(label, $labelColors)}">
            {label}
          </span>
        {/each}
      </div>
    </div>
  {/if}

  {#if meta.range}
    <div class="sidebar-section">
      <div class="date-header">
        <h4 class="sidebar-title">Date Range</h4>
        <button class="counter-header-btn remove" title="Remove dates" onclick={removeAllDates}>
          <svg viewBox="0 0 24 24" width="12" height="12">
            <polyline points="3 6 5 6 21 6" fill="none" stroke="currentColor" stroke-width="2"
              stroke-linecap="round" stroke-linejoin="round"
            />
            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"
              fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
            />
          </svg>
        </button>
      </div>
      <div class="counter-range-row">
        <DatePicker value={meta.range.start} onselect={d => onRangeDateSelect('start', d)} />
        <span class="range-text">to</span>
        <DatePicker value={meta.range.end} onselect={d => onRangeDateSelect('end', d)} />
      </div>
      <button class="date-expand-btn" title="Convert to single date" onclick={removeEndDate}>- End date</button>
    </div>
  {:else if meta.due}
    <div class="sidebar-section">
      <div class="date-header">
        <h4 class="sidebar-title">Date</h4>
        <button class="counter-header-btn remove" title="Remove date" onclick={removeAllDates}>
          <svg viewBox="0 0 24 24" width="12" height="12">
            <polyline points="3 6 5 6 21 6" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"
              fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
            />
          </svg>
        </button>
      </div>
      <DatePicker value={meta.due} onselect={onDueDateSelect} />
      <button class="date-expand-btn" title="Add an end date to create a range" onclick={addEndDate}>+ End date</button>
    </div>
  {:else}
    <div class="sidebar-section">
      <button class="add-counter-btn" title="Add a due date" onclick={addDate}>+ Add date</button>
    </div>
  {/if}

  {#if meta.counter}
    <div class="sidebar-section">
      <div class="counter-header">
        <h4 class="sidebar-title">{meta.counter.label || "Counter"}</h4>
        <div class="counter-header-right">
          {#if counterSettingsOpen}
            <button class="counter-header-btn save" title="Save settings" onclick={saveCounterSettings}>
              <svg viewBox="0 0 24 24" width="12" height="12">
                <polyline points="20 6 9 17 4 12" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </button>
          {:else}
            <button class="counter-header-btn" title="Counter settings" onclick={openCounterSettings}>
              <svg viewBox="0 0 24 24" width="12" height="12">
                <circle cx="12" cy="12" r="3" fill="none" stroke="currentColor" stroke-width="2"/>
                <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65
                  1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9
                  19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0
                  4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65
                  0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65
                  0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0
                  1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0
                  1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"
                  fill="none" stroke="currentColor" stroke-width="2"
                />
              </svg>
            </button>
          {/if}
          <button class="counter-header-btn remove" title="Remove counter" onclick={removeCounter}>
            <svg viewBox="0 0 24 24" width="12" height="12">
              <polyline points="3 6 5 6 21 6" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"
                fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
              />
            </svg>
          </button>
        </div>
      </div>
      <div class="counter-progress-row">
        <button class="counter-btn" title="Decrease" disabled={atStart} onclick={() => adjustCounter(countingDown ? 1 : -1)}>-</button>
        <div class="progress-bar sidebar-progress">
          <div class="progress-fill" class:complete={counterPct >= 100} style="width: {counterPct}%"></div>
        </div>
        <span class="counter-fraction">{meta.counter.current}/{meta.counter.max}</span>
        <button class="counter-btn" title="Increase" disabled={atGoal} onclick={() => adjustCounter(countingDown ? -1 : 1)}>+</button>
      </div>
      {#if counterSettingsOpen}
        <div class="counter-settings">
          <input type="text" class="counter-input" bind:value={editLabel} placeholder="Label" onkeydown={e => e.key === 'Enter' && saveCounterSettings()}/>
          <div class="counter-range-row">
            <input type="number" class="counter-input range-input" bind:value={editStart}
              onblur={() => { editStart = Math.max(0, Number(editStart) || 0); if (editStart === editMax) { editMax = editStart + 1; } }}
              onkeydown={e => e.key === 'Enter' && saveCounterSettings()}
            />
            <span class="range-text">to</span>
            <input type="number" class="counter-input range-input" bind:value={editMax}
              onblur={() => { editMax = Math.max(0, Number(editMax) || 0); if (editMax === editStart) { editMax = editStart + 1; } }}
              onkeydown={e => e.key === 'Enter' && saveCounterSettings()}
            />
            <span class="range-text">by</span>
            <input type="number" class="counter-input range-input" bind:value={editStep} min="1"
              onblur={() => editStep = Math.max(1, editStep || 1)}
              onkeydown={e => e.key === 'Enter' && saveCounterSettings()}
            />
          </div>
        </div>
      {/if}
    </div>
  {:else}
    <div class="sidebar-section">
      <button class="add-counter-btn" title="Add a progress counter" onclick={addCounter}>+ Add counter</button>
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
    flex: 0 0 200px;
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
    color: var(--color-text-secondary);
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
    flex: 1;
  }

  .counter-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 4px;

    .sidebar-title {
      margin: 0;
    }
  }

  .counter-header-right {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .counter-header-btn {
    all: unset;
    color: var(--color-text-muted);
    cursor: pointer;
    padding: 2px;
    border-radius: 3px;
    display: flex;

    &:hover {
      color: var(--color-text-primary);
    }

    &.save:hover {
      color: var(--color-success);
    }

    &.remove:hover {
      color: var(--color-error);
    }
  }

  .counter-btn {
    all: unset;
    width: 22px;
    height: 22px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    border-radius: 4px;
    color: var(--color-text-primary);
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;
    box-sizing: border-box;

    &:hover:not(:disabled) {
      background: var(--overlay-hover);
      border-color: var(--color-text-tertiary);
    }

    &:disabled {
      opacity: 0.3;
      cursor: not-allowed;
    }
  }

  .counter-progress-row {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .counter-fraction {
    font-size: 0.7rem;
    font-weight: 600;
    color: var(--color-text-primary);
    flex-shrink: 0;
  }

  .counter-settings {
    margin-top: 10px;
    padding-top: 10px;
    border-top: 1px solid var(--color-border);
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .counter-input {
    width: 100%;
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    color: var(--color-text-primary);
    font-size: 0.75rem;
    padding: 5px 8px;
    border-radius: 4px;
    outline: none;
    box-sizing: border-box;
    appearance: textfield;
    -moz-appearance: textfield;

    &::-webkit-inner-spin-button,
    &::-webkit-outer-spin-button {
      -webkit-appearance: none;
      margin: 0;
    }

    &:focus {
      border-color: var(--color-accent);
    }
  }

  .counter-range-row {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .range-text {
    font-size: 0.7rem;
    color: var(--color-text-muted);
    flex-shrink: 0;
  }

  .range-input {
    width: 0;
    flex: 1;
    padding: 4px 6px;
    text-align: center;
  }

  .date-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 4px;

    .sidebar-title {
      margin: 0;
    }
  }

  .date-expand-btn {
    all: unset;
    display: block;
    font-size: 0.7rem;
    color: var(--color-text-muted);
    cursor: pointer;
    margin-top: 6px;

    &:hover {
      color: var(--color-text-primary);
    }
  }

  .add-counter-btn {
    all: unset;
    width: 100%;
    text-align: left;
    font-size: 0.75rem;
    color: var(--color-text-muted);
    cursor: pointer;
    padding: 2px 0;

    &:hover {
      color: var(--color-text-primary);
    }
  }
</style>
