<script lang="ts">
  // Card detail sidebar. Shows labels, dates, counter, and timestamps.

  import {
    boardPath, addToast, labelColors,
    selectedCard, boardConfig, boardData, sortedListKeys, listOrder,
    moveCardInBoard, computeListOrder, isAtLimit, isLocked,
  } from "../stores/board";
  import { formatDateTime, labelColor, autoFocus, getDisplayTitle } from "../lib/utils";
  import { MoveCard, LoadBoard } from "../../wailsjs/go/main/App";
  import type { daedalus } from "../../wailsjs/go/models";
  import CounterControl from "./CounterControl.svelte";
  import DateSection from "./DateSection.svelte";
  import Icon from "./Icon.svelte";
  import CardIcon from "./CardIcon.svelte";
  import IconPicker from "./IconPicker.svelte";

  let {
    meta,
    onsavecounter,
    onsavedates,
    onsaveestimate,
    onsaveicon,
    onsavechecklist,
    onsavelabels,
  }: {
    meta: daedalus.CardMetadata;
    onsavecounter?: (counter: daedalus.Counter | null) => void;
    onsavedates?: (
      due: string | null,
      range: { start: string; end: string } | null,
    ) => void;
    onsaveestimate?: (estimate: number | null) => void;
    onsaveicon?: (icon: string) => void;
    onsavechecklist?: (checklist: daedalus.CheckListItem[] | null) => void;
    onsavelabels?: (labels: string[]) => void;
  } = $props();

  let iconPickerOpen = $state(false);
  let listDropdownOpen = $state(false);
  let positionDropdownOpen = $state(false);

  let editingEstimate = $state(false);
  let estimateInput = $state("");
  let labelDropdownOpen = $state(false);

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

  // Board labels not currently on this card, sorted alphabetically.
  let availableLabels = $derived.by(() => {
    const current = new Set(meta.labels || []);
    return Object.keys($labelColors).filter(l => !current.has(l)).sort();
  });

  // Removes a label from this card.
  function removeLabel(label: string): void {
    onsavelabels?.((meta.labels || []).filter(l => l !== label));
  }

  // Adds a label to this card from the registry. Dropdown stays open for multi-select.
  function addLabel(label: string): void {
    onsavelabels?.([...(meta.labels || []), label]);
  }

  let iconSectionEl: HTMLElement | undefined = $state();
  let labelSectionEl: HTMLElement | undefined = $state();
  let listDropdownEl: HTMLElement | undefined = $state();
  let positionDropdownEl: HTMLElement | undefined = $state();

  // Closes dropdowns/pickers when clicking outside of them.
  function handleWindowClick(e: MouseEvent): void {
    const target = e.target as Node;
    // Ignore clicks on elements removed from DOM by a reactive branch swap
    if (!target.isConnected) {
      return;
    }
    if (listDropdownOpen && listDropdownEl && !listDropdownEl.contains(target)) {
      listDropdownOpen = false;
    }
    if (positionDropdownOpen && positionDropdownEl && !positionDropdownEl.contains(target)) {
      positionDropdownOpen = false;
    }
    if (iconPickerOpen && iconSectionEl && !iconSectionEl.contains(target)) {
      iconPickerOpen = false;
    }
    if (labelDropdownOpen && labelSectionEl && !labelSectionEl.contains(target)) {
      labelDropdownOpen = false;
    }
  }

  // The list key where the currently selected card lives.
  let cardListKey = $derived.by(() => {
    if (!$selectedCard) {
      return "";
    }
    for (const key of Object.keys($boardData)) {
      if ($boardData[key].some(c => c.filePath === $selectedCard!.filePath)) {
        return key;
      }
    }
    return "";
  });

  // Selected list key for the position editor (tracks user selection).
  let selectedListKey = $state("");
  // Selected 0-based position index for the position editor.
  let selectedPosition = $state(0);

  // Display name of the selected list.
  let selectedListDisplay = $derived(selectedListKey ? getDisplayTitle(selectedListKey, $boardConfig) : "");

  // Display text for the selected position.
  let selectedPositionDisplay = $derived.by(() => {
    const n = selectedPosition + 1;
    if (positionCount <= 1) {
      return `${n}`;
    }
    if (selectedPosition === 0) {
      return `${n} (top)`;
    }
    if (selectedPosition === positionCount - 1) {
      return `${n} (bottom)`;
    }
    return `${n}`;
  });

  // Selects a list and closes the list dropdown.
  function selectList(key: string): void {
    selectedListKey = key;
    listDropdownOpen = false;
  }

  // Selects a position and closes the position dropdown.
  function selectPosition(idx: number): void {
    selectedPosition = idx;
    positionDropdownOpen = false;
  }

  // Reset dropdowns when the selected card changes.
  $effect(() => {
    if ($selectedCard && cardListKey) {
      selectedListKey = cardListKey;
      const cards = $boardData[cardListKey] || [];
      const idx = cards.findIndex(c => c.filePath === $selectedCard!.filePath);
      selectedPosition = idx === -1 ? 0 : idx;
    }
  });

  // Reset position to top when switching to a different list, restore actual position when switching back.
  let prevSelectedListKey = $state("");
  $effect(() => {
    if (selectedListKey !== prevSelectedListKey) {
      if (selectedListKey !== cardListKey) {
        selectedPosition = 0;
      } else if ($selectedCard) {
        const cards = $boardData[cardListKey] || [];
        const idx = cards.findIndex(c => c.filePath === $selectedCard!.filePath);
        selectedPosition = idx === -1 ? 0 : idx;
      }
      prevSelectedListKey = selectedListKey;
    }
  });

  // Cards in the currently selected target list.
  let targetCards = $derived($boardData[selectedListKey] || []);

  // Number of position slots available in the target list.
  // Same list: N slots (the card's existing positions).
  // Different list: N+1 slots (insert before each card, plus append).
  let positionCount = $derived.by(() => {
    const count = targetCards.length;
    if (selectedListKey === cardListKey) {
      return count;
    }
    return count + 1;
  });

  // Whether the current selection differs from the card's actual position.
  let hasPendingMove = $derived.by(() => {
    if (selectedListKey !== cardListKey) {
      return true;
    }
    const cards = $boardData[cardListKey] || [];
    const currentIdx = cards.findIndex(c => c.filePath === $selectedCard!.filePath);
    return selectedPosition !== currentIdx;
  });

  // Moves the card to the selected list and position.
  async function executeMove(): Promise<void> {
    if (!$selectedCard || !hasPendingMove) {
      return;
    }
    if (isLocked(cardListKey, $boardConfig) || isLocked(selectedListKey, $boardConfig)) {
      addToast("List is locked");
      return;
    }
    if (selectedListKey !== cardListKey && isAtLimit(selectedListKey, $boardData, $boardConfig)) {
      addToast("List is at its card limit");
      return;
    }

    // When reordering within the same list, account for the card being removed first.
    let cardsForOrder: daedalus.KanbanCard[];
    if (selectedListKey === cardListKey) {
      cardsForOrder = targetCards.filter(c => c.filePath !== $selectedCard!.filePath);
    } else {
      cardsForOrder = targetCards;
    }

    const newListOrder = computeListOrder(cardsForOrder, selectedPosition);
    const originalPath = $selectedCard.filePath;

    moveCardInBoard(originalPath, cardListKey, selectedListKey, selectedPosition, newListOrder);

    try {
      const result = await MoveCard(originalPath, selectedListKey, newListOrder);

      // Sync file path if it changed (cross-list move).
      if (result.filePath !== originalPath) {
        boardData.update(lists => {
          const tc = lists[selectedListKey];
          if (tc) {
            const idx = tc.findIndex(c => c.metadata.id === result.metadata.id);
            if (idx !== -1) {
              tc[idx] = { ...tc[idx], filePath: result.filePath, listName: result.listName } as daedalus.KanbanCard;
            }
          }
          return lists;
        });
      }
      selectedCard.set(result);
      addToast("Card moved", "success");
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
    <h4 class="sidebar-title">Card #{meta.id}</h4>
    <div class="position-editor">
      <div class="position-field">
        <span class="position-label">List</span>
        <div class="pos-dropdown" bind:this={listDropdownEl}>
          <button class="pos-trigger" onclick={() => listDropdownOpen = !listDropdownOpen}>
            <span class="pos-trigger-text">{selectedListDisplay}</span>
            <svg class="pos-chevron" class:open={listDropdownOpen} viewBox="0 0 16 16" width="12" height="12">
              <path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
          {#if listDropdownOpen}
            <div class="pos-menu">
              {#each sortedListKeys($boardData, $listOrder) as key}
                {@const locked = isLocked(key, $boardConfig)}
                {@const full = key !== cardListKey && isAtLimit(key, $boardData, $boardConfig)}
                {@const blocked = locked || full}
                <button class="pos-option" class:active={key === selectedListKey} class:disabled={blocked} disabled={blocked} onclick={() => selectList(key)}>
                  {getDisplayTitle(key, $boardConfig)}
                  {#if locked}<span class="pos-hint">(locked)</span>{/if}
                  {#if full}<span class="pos-hint">(full)</span>{/if}
                </button>
              {/each}
            </div>
          {/if}
        </div>
      </div>
      <div class="position-field">
        <span class="position-label">Position</span>
        <div class="pos-dropdown" bind:this={positionDropdownEl}>
          <button class="pos-trigger" onclick={() => positionDropdownOpen = !positionDropdownOpen}>
            <span class="pos-trigger-text">{selectedPositionDisplay}</span>
            <svg class="pos-chevron" class:open={positionDropdownOpen} viewBox="0 0 16 16" width="12" height="12">
              <path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
          {#if positionDropdownOpen}
            <div class="pos-menu">
              {#each Array.from({ length: positionCount }, (_, i) => i) as idx}
                <button class="pos-option" class:active={idx === selectedPosition} onclick={() => selectPosition(idx)}>
                  {idx + 1}{idx === 0 ? " (top)" : ""}{idx === positionCount - 1 && positionCount > 1 ? " (bottom)" : ""}
                </button>
              {/each}
            </div>
          {/if}
        </div>
      </div>
      {#if hasPendingMove}
        <button class="position-move-btn" onclick={executeMove}>Move</button>
      {/if}
    </div>
  </div>

  <div class="sidebar-section" bind:this={labelSectionEl}>
    <h4 class="sidebar-title">Labels</h4>
    {#if meta.labels && meta.labels.length > 0}
      <div class="sidebar-labels">
        {#each [...meta.labels].sort() as label}
          <button class="label label-removable" title="Remove {label}" style="background: {labelColor(label, $labelColors)}" onclick={() => removeLabel(label)}>
            {label}
          </button>
        {/each}
      </div>
    {/if}
    {#if availableLabels.length > 0}
      <div class="label-add-wrapper">
        <button class="add-counter-btn" onclick={() => labelDropdownOpen = !labelDropdownOpen}>+ Add label</button>
        {#if labelDropdownOpen}
          <div class="label-add-menu">
            {#each availableLabels as label}
              <button class="label-add-option" onclick={() => addLabel(label)}>
                <span class="label-add-swatch" style="background: {$labelColors[label]}"></span>
                {label}
              </button>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </div>

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

  .label-removable {
    cursor: pointer;
    border: none;
    text-align: center;

    &:hover {
      opacity: 0.7;
    }
  }

  .label-add-wrapper {
    position: relative;
    margin-top: 4px;
  }

  .label-add-menu {
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

  .label-add-option {
    all: unset;
    display: flex;
    align-items: center;
    gap: 6px;
    width: 100%;
    padding: 5px 8px;
    font-size: 0.8rem;
    color: var(--color-text-primary);
    cursor: pointer;
    box-sizing: border-box;

    &:hover {
      background: var(--overlay-hover);
    }
  }

  .label-add-swatch {
    width: 10px;
    height: 10px;
    border-radius: 2px;
    flex-shrink: 0;
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

  .position-editor {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .position-field {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .position-label {
    font-size: 0.68rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-text-tertiary);
  }

  .pos-dropdown {
    position: relative;
  }

  .pos-trigger {
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

  .pos-trigger-text {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }

  .pos-chevron {
    color: var(--color-text-tertiary);
    transition: transform 0.15s;
    flex-shrink: 0;

    &.open {
      transform: rotate(180deg);
    }
  }

  .pos-menu {
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

  .pos-option {
    all: unset;
    display: flex;
    align-items: center;
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

  .pos-hint {
    font-size: 0.7rem;
    color: var(--color-text-muted);
    margin-left: auto;
  }

  .position-move-btn {
    all: unset;
    text-align: center;
    background: var(--color-accent);
    color: var(--color-text-inverse);
    font-size: 0.78rem;
    font-weight: 600;
    padding: 4px 8px;
    border-radius: 4px;
    cursor: pointer;

    &:hover {
      opacity: 0.9;
    }
  }

</style>
