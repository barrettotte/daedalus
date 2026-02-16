<script lang="ts">
  // Card detail sidebar. Shows move-to-list dropdown, labels, dates, counter, and timestamps.

  import {
    selectedCard, boardConfig, boardData, boardPath, sortedListKeys, listOrder,
    moveCardInBoard, computeListOrder, addToast, isAtLimit, isLocked, labelColors,
  } from "../stores/board";
  import { MoveCard, LoadBoard } from "../../wailsjs/go/main/App";
  import { getDisplayTitle, formatDateTime, labelColor, autoFocus } from "../lib/utils";
  import type { daedalus } from "../../wailsjs/go/models";
  import CounterControl from "./CounterControl.svelte";
  import DateSection from "./DateSection.svelte";
  import Icon from "./Icon.svelte";
  import CardIcon from "./CardIcon.svelte";
  import IconPicker from "./IconPicker.svelte";

  let {
    meta,
    moveDropdownOpen = $bindable(false),
    onsavecounter,
    onsavedates,
    onsaveestimate,
    onsaveicon,
    onsavechecklist,
  }: {
    meta: daedalus.CardMetadata;
    moveDropdownOpen?: boolean;
    onsavecounter?: (counter: daedalus.Counter | null) => void;
    onsavedates?: (
      due: string | null,
      range: { start: string; end: string } | null,
    ) => void;
    onsaveestimate?: (estimate: number | null) => void;
    onsaveicon?: (icon: string) => void;
    onsavechecklist?: (checklist: daedalus.CheckListItem[] | null) => void;
  } = $props();

  let iconPickerOpen = $state(false);

  let editingEstimate = $state(false);
  let estimateInput = $state("");

  // Opens the estimate field for inline editing.
  function startEditEstimate(): void {
    estimateInput = meta.estimate != null ? String(meta.estimate) : "";
    editingEstimate = true;
  }

  // Saves the estimate on blur. Empty or zero clears it.
  function blurEstimate(): void {
    editingEstimate = false;
    const val = parseFloat(estimateInput);
    if (isNaN(val) || val <= 0) {
      if (meta.estimate != null) {
        onsaveestimate?.(null);
      }
    } else if (val !== meta.estimate) {
      onsaveestimate?.(val);
    }
  }

  let dropdownEl: HTMLElement | undefined = $state();
  let iconSectionEl: HTMLElement | undefined = $state();

  // Closes dropdowns/pickers when clicking outside of them.
  function handleWindowClick(e: MouseEvent): void {
    const target = e.target as Node;
    // Ignore clicks on elements removed from DOM by a reactive branch swap
    if (!target.isConnected) {
      return;
    }
    if (moveDropdownOpen && dropdownEl && !dropdownEl.contains(target)) {
      moveDropdownOpen = false;
    }
    if (iconPickerOpen && iconSectionEl && !iconSectionEl.contains(target)) {
      iconPickerOpen = false;
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
            <button class="move-option" class:active={key === cardListKey} class:disabled={blocked} disabled={blocked}
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
        {#each [...meta.labels].sort() as label}
          <span class="label" title={label} style="background: {labelColor(label, $labelColors)}">
            {label}
          </span>
        {/each}
      </div>
    </div>
  {/if}

  {#if meta.icon || iconPickerOpen}
    <div class="sidebar-section" bind:this={iconSectionEl}>
      <div class="section-header">
        <h4 class="sidebar-title">Icon</h4>
        {#if meta.icon && !iconPickerOpen}
          <button class="icon-current" title={`${$boardPath}/assets/icons/${meta.icon}`} onclick={() => iconPickerOpen = true}>
            <CardIcon name={meta.icon} size={14} />
          </button>
        {/if}
        <div class="section-header-actions">
          {#if iconPickerOpen}
            <button class="counter-header-btn save" title="Done" onclick={() => iconPickerOpen = false}>
              <Icon name="check" size={12} />
            </button>
          {:else if meta.icon}
            <button class="counter-header-btn" title="Change icon" onclick={() => iconPickerOpen = true}>
              <Icon name="pencil" size={12} />
            </button>
          {/if}
          <button class="counter-header-btn remove" title="Remove icon" onclick={() => { iconPickerOpen = false; onsaveicon?.(""); }}>
            <Icon name="trash" size={12} />
          </button>
        </div>
      </div>
      {#if iconPickerOpen}
        <IconPicker currentIcon={meta.icon || ""} onselect={(name) => { iconPickerOpen = false; onsaveicon?.(name); }}/>
      {/if}
    </div>
  {:else}
    <div class="sidebar-section" bind:this={iconSectionEl}>
      <button class="add-counter-btn" onclick={() => iconPickerOpen = true}>+ Add icon</button>
    </div>
  {/if}

  <DateSection due={meta.due} range={meta.range} onsave={onsavedates} />

  {#if meta.estimate != null || editingEstimate}
    <div class="sidebar-section">
      <div class="section-header">
        <h4 class="sidebar-title">Estimate</h4>
        {#if editingEstimate}
          <span class="estimate-inline">
            <span class="estimate-sep">-</span>
            <input class="estimate-input" type="number" step="0.5" min="0"
              bind:value={estimateInput}
              onblur={blurEstimate}
              onkeydown={e => e.key === 'Enter' && (e.target as HTMLInputElement).blur()}
              use:autoFocus
            />
          </span>
          <div class="section-header-actions">
            <button class="counter-header-btn save" title="Confirm" onclick={() => (document.querySelector('.estimate-input') as HTMLInputElement)?.blur()}>
              <Icon name="check" size={12} />
            </button>
            <button class="counter-header-btn remove" title="Remove estimate" onclick={() => onsaveestimate?.(null)}>
              <Icon name="trash" size={12} />
            </button>
          </div>
        {:else}
          <span class="estimate-inline">
            <span class="estimate-sep">-</span>
            <button class="estimate-value" onclick={startEditEstimate}>{meta.estimate}h</button>
          </span>
          <div class="section-header-actions">
            <button class="counter-header-btn" title="Edit estimate" onclick={startEditEstimate}>
              <Icon name="pencil" size={12} />
            </button>
            <button class="counter-header-btn remove" title="Remove estimate" onclick={() => onsaveestimate?.(null)}>
              <Icon name="trash" size={12} />
            </button>
          </div>
        {/if}
      </div>
    </div>
  {:else}
    <div class="sidebar-section">
      <button class="add-counter-btn" onclick={startEditEstimate}>+ Add estimate</button>
    </div>
  {/if}

  <CounterControl counter={meta.counter} onsave={onsavecounter} />

  {#if meta.checklist_title || (meta.checklist && meta.checklist.length > 0)}
    {@const items = meta.checklist || []}
    {@const done = items.filter(i => i.done).length}
    <div class="sidebar-section">
      <div class="section-header">
        <h4 class="sidebar-title">{meta.checklist_title || "Checklist"}</h4>
        {#if items.length > 0}
          <span class="estimate-inline">
            <span class="estimate-sep">-</span>
            <span class="checklist-hint" class:all-done={done === items.length}>
              {done}/{items.length}
            </span>
          </span>
        {/if}
        <div class="section-header-actions">
          <button class="counter-header-btn remove" title="Remove checklist" onclick={() => onsavechecklist?.(null)}>
            <Icon name="trash" size={12} />
          </button>
        </div>
      </div>
    </div>
  {:else}
    <div class="sidebar-section">
      <button class="add-counter-btn" title="Add a checklist" onclick={() => onsavechecklist?.([])}>+ Add checklist</button>
    </div>
  {/if}

  <div class="sidebar-section timestamps">
    {#if meta.created}
      <div class="timestamp-row">
        <span class="timestamp-label">Created</span>
        <span class="sidebar-value">{formatDateTime(meta.created)}</span>
      </div>
    {/if}
    <div class="timestamp-row">
      <span class="timestamp-label">Updated</span>
      <span class="sidebar-value">
        {formatDateTime(meta.updated && meta.updated !== meta.created ? meta.updated : meta.created)}
      </span>
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
    color: var(--color-text-inverse);
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

  .icon-current {
    all: unset;
    display: inline-flex;
    margin-left: 10px;
    margin-right: auto;
    color: var(--color-text-secondary);
    cursor: pointer;

    &:hover {
      color: var(--color-text-primary);
    }
  }

  .estimate-inline {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 0.7rem;
    color: var(--color-text-tertiary);
    margin-right: auto;
    margin-left: 4px;
  }

  .estimate-sep {
    line-height: 0;
  }

  .estimate-value {
    all: unset;
    font-size: 0.8rem;
    color: var(--color-text-secondary);
    cursor: pointer;

    &:hover {
      color: var(--color-text-primary);
    }
  }

  .estimate-input {
    width: 50px;
    background: var(--color-bg-base);
    border: 1px solid var(--color-accent);
    color: var(--color-text-primary);
    font-size: 0.8rem;
    padding: 1px 4px;
    border-radius: 4px;
    outline: none;
    box-sizing: border-box;
  }

  .checklist-hint {
    font-size: 0.8rem;
    color: var(--color-text-secondary);

    &.all-done {
      color: var(--color-success);
    }
  }

  .timestamps {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .timestamp-row {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
  }

  .timestamp-label {
    font-size: 0.68rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-text-tertiary);
  }

</style>
