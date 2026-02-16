<script lang="ts">
  // Modal for creating a new card with title, body, labels, and position selector.

  import {
    selectedCard, draftListKey, draftPosition,
    addCardToBoard, boardConfig, boardData, addToast, isAtLimit, labelColors,
  } from "../stores/board";
  import { CreateCard, SaveCard } from "../../wailsjs/go/main/App";
  import { formatListName, labelColor, autoFocus, backdropClose } from "../lib/utils";
  import type { daedalus } from "../../wailsjs/go/models";
  import Icon from "./Icon.svelte";

  // Draft mode state for creating new cards before they exist on disk
  let draftTitle = $state("");
  let draftBody = $state("");
  let draftLabels = $state<string[]>([]);
  let labelDropdownOpen = $state(false);
  let saving = $state(false);

  // Number of cards in the draft target list.
  let draftListCount = $derived.by(() => {
    if (!$draftListKey) {
      return 0;
    }
    return ($boardData[$draftListKey] || []).length;
  });

  // 1-based position number derived from the current draftPosition value.
  let positionDisplayValue = $derived.by(() => {
    const pos = $draftPosition;
    if (pos === "top") {
      return 1;
    }
    if (pos === "bottom") {
      return draftListCount + 1;
    }

    const idx = parseInt(pos, 10);
    if (!isNaN(idx)) {
      return idx + 1;
    }
    return 1;
  });

  // Speculative ID for the next card (current max + 1).
  let nextCardId = $derived.by(() => {
    let max = 0;
    for (const cards of Object.values($boardData)) {
      for (const card of cards) {
        if (card.metadata.id > max) {
          max = card.metadata.id;
        }
      }
    }
    return max + 1;
  });

  // Derives the list display name from config title or formatted directory name.
  let draftListDisplayName = $derived.by(() => {
    if (!$draftListKey) {
      return "";
    }
    const cfg = $boardConfig[$draftListKey];
    if (cfg && cfg.title) {
      return cfg.title;
    }
    return formatListName($draftListKey);
  });

  // Board labels not currently selected for this draft, sorted alphabetically.
  let availableLabels = $derived.by(() => {
    const current = new Set(draftLabels);
    return Object.keys($labelColors).filter(l => !current.has(l)).sort();
  });

  function addDraftLabel(label: string): void {
    draftLabels = [...draftLabels, label];
  }

  function removeDraftLabel(label: string): void {
    draftLabels = draftLabels.filter(l => l !== label);
  }

  // Closes the draft modal and clears all draft state.
  function close(): void {
    draftTitle = "";
    draftBody = "";
    draftLabels = [];
    labelDropdownOpen = false;
    saving = false;
    draftListKey.set(null);
    selectedCard.set(null);
  }

  // Validates and saves the draft card to disk, then adds it to the board store.
  async function saveDraft(): Promise<void> {
    if (!draftTitle.trim()) {
      return;
    }

    // Re-check limit in case the list filled while the modal was open.
    if (isAtLimit($draftListKey!, $boardData, $boardConfig)) {
      addToast("List is at its card limit");
      return;
    }
    saving = true;

    try {
      const pos = $draftPosition;
      let card = await CreateCard($draftListKey!, draftTitle.trim(), draftBody, pos);

      if (draftLabels.length > 0) {
        const fullBody = `# ${draftTitle.trim()}\n\n${draftBody}`;
        card = await SaveCard(card.filePath, { ...card.metadata, labels: draftLabels } as daedalus.CardMetadata, fullBody);
      }

      addCardToBoard($draftListKey!, card, pos);
      addToast("Card created", "success");

      draftTitle = "";
      draftBody = "";
      draftLabels = [];
      labelDropdownOpen = false;
      draftListKey.set(null);
      selectedCard.set(null);

    } catch (e) {
      addToast(`Failed to create card: ${e}`);
    }
    saving = false;
  }

  // Converts 1-based user input to the store's position value, clamping to valid range.
  function handlePositionInput(e: Event): void {
    const input = e.target as HTMLInputElement;
    const raw = parseInt(input.value, 10);
    if (isNaN(raw)) {
      input.value = String(positionDisplayValue);
      return;
    }

    const max = draftListCount + 1;
    const val = Math.max(1, Math.min(raw, max));
    if (val !== raw) {
      input.value = String(val);
    }

    if (val === 1) {
      draftPosition.set("top");
    } else if (val === max) {
      draftPosition.set("bottom");
    } else {
      draftPosition.set(String(val - 1));
    }
  }

  let labelAddEl: HTMLElement | undefined = $state();

  // Closes label dropdown when clicking outside.
  function handleWindowClick(e: MouseEvent): void {
    const target = e.target as Node;
    if (!target.isConnected) {
      return;
    }
    if (labelDropdownOpen && labelAddEl && !labelAddEl.contains(target)) {
      labelDropdownOpen = false;
    }
  }

  // Handles keyboard shortcuts: Ctrl/Cmd+Enter saves draft, Escape closes.
  function handleKeydown(e: KeyboardEvent): void {
    if (!$draftListKey) {
      return;
    }

    if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
      e.preventDefault();
      saveDraft();
    } else if (e.key === "Escape") {
      close();
    }
  }

  // Resets draft fields only on transition from null to non-null (new draft started).
  let prevDraftListKey: string | null = null;
  $effect(() => {
    const current = $draftListKey;
    if (current && current !== prevDraftListKey) {
      draftTitle = "";
      draftBody = "";
      draftLabels = [];
      labelDropdownOpen = false;
      saving = false;
    }
    prevDraftListKey = current;
  });
</script>

<svelte:window onkeydown={handleKeydown} onclick={handleWindowClick} />

{#if $draftListKey}
  <div class="modal-backdrop scrollable" role="presentation" use:backdropClose={close} onkeydown={handleKeydown}>
    <div class="modal-dialog size-lg draft-dialog" role="dialog">
      <div class="modal-header draft-header">
        <div class="draft-list-name">
          Drafting card <span class="draft-card-id">#{nextCardId}</span> in <strong>{draftListDisplayName}</strong>
        </div>
        <div class="header-btns">
          <button class="modal-close" onclick={close} title="Close">
            <Icon name="close" size={16} />
          </button>
        </div>
      </div>
      <div class="draft-body">
        <input class="edit-title-input" type="text" bind:value={draftTitle} placeholder="Card title"
          onkeydown={e => e.key === 'Enter' && saveDraft()} use:autoFocus
        />
        <textarea class="edit-body-textarea" bind:value={draftBody} placeholder="Card description (markdown)"></textarea>
        {#if draftLabels.length > 0 || Object.keys($labelColors).length > 0}
          <div class="draft-labels">
            {#if draftLabels.length > 0}
              <div class="draft-label-chips">
                {#each [...draftLabels].sort() as label}
                  <button class="draft-label" title="Remove {label}" style="background: {labelColor(label, $labelColors)}" onclick={() => removeDraftLabel(label)}>
                    {label}
                  </button>
                {/each}
              </div>
            {/if}
            {#if availableLabels.length > 0}
              <div class="draft-label-add" bind:this={labelAddEl}>
                <button class="draft-add-label-btn" onclick={() => labelDropdownOpen = !labelDropdownOpen}>+ Add label</button>
                {#if labelDropdownOpen}
                  <div class="draft-label-menu">
                    {#each availableLabels as label}
                      <button class="draft-label-option" onclick={() => addDraftLabel(label)}>
                        <span class="draft-label-swatch" style="background: {$labelColors[label]}"></span>
                        {label}
                      </button>
                    {/each}
                  </div>
                {/if}
              </div>
            {/if}
          </div>
        {/if}
      </div>
      <div class="draft-actions">
        <div class="position-section">
          <span class="position-label">Position</span>
          <div class="position-toggle">
            <button class="pos-btn" title="Add card to top of list" class:active={$draftPosition === 'top'} onclick={() => draftPosition.set('top')}>Top</button>
            <button class="pos-btn" title="Add card to bottom of list" class:active={$draftPosition === 'bottom'} onclick={() => draftPosition.set('bottom')}>Bottom</button>
          </div>
          <div class="position-specific-row">
            <input class="position-input" type="number" min="1" max={draftListCount + 1} value={positionDisplayValue} oninput={handlePositionInput}/>
            <span class="position-hint">of {draftListCount + 1}</span>
          </div>
        </div>
        <div class="draft-btns">
          <button class="save-btn" onclick={saveDraft} disabled={saving || !draftTitle.trim()}>
            {saving ? "Saving..." : "Save"}
          </button>
          <button class="cancel-btn" onclick={close}>Cancel</button>
        </div>
      </div>
    </div>
  </div>
{/if}

<style lang="scss">
  .draft-dialog {
    margin-bottom: 48px;
  }

  .draft-header {
    border-bottom: none;
    padding: 16px 20px 0 20px;
  }

  .draft-list-name {
    font-size: 0.78rem;
    color: var(--color-text-tertiary);

    strong {
      color: var(--color-text-secondary);
    }
  }

  .draft-card-id {
    color: var(--color-text-muted);
    font-family: monospace;
    font-size: 0.75rem;
  }

  .draft-body {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 16px 20px 0 20px;
  }

  .edit-title-input {
    background: var(--color-bg-base);
    border: 1px solid var(--color-accent);
    color: var(--color-text-primary);
    font-size: 1.25rem;
    font-weight: 600;
    padding: 2px 10px;
    border-radius: 4px;
    outline: none;
    box-sizing: border-box;
    font-family: inherit;
  }

  .draft-actions {
    display: flex;
    align-items: flex-end;
    gap: 8px;
    padding: 16px 20px 20px 20px;
  }

  .position-section {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-right: auto;
  }

  .position-label {
    font-size: 0.78rem;
    font-weight: 600;
    color: var(--color-text-secondary);
  }

  .position-toggle {
    display: flex;
    border-radius: 4px;
    overflow: hidden;
    border: 1px solid var(--color-border);
    width: fit-content;
  }

  .position-specific-row {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .draft-btns {
    display: flex;
    gap: 8px;
    flex-shrink: 0;
  }

  .position-input {
    width: 44px;
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    color: var(--color-text-primary);
    font-size: 0.78rem;
    padding: 4px 6px;
    border-radius: 4px;
    outline: none;
    text-align: center;
    &:focus {
      border-color: var(--color-accent);
    }
  }

  .position-hint {
    font-size: 0.75rem;
    color: var(--color-text-tertiary);
  }

  .pos-btn {
    all: unset;
    padding: 5px 12px;
    font-size: 0.78rem;
    font-weight: 500;
    cursor: pointer;
    color: var(--color-text-secondary);
    background: transparent;

    &:hover {
      background: var(--overlay-hover-light);
    }

    &.active {
      background: var(--overlay-accent);
      color: var(--color-accent);
    }
  }

  .save-btn {
    background: var(--color-accent);
    color: var(--color-bg-base);
    border: none;
    padding: 8px 20px;
    border-radius: 4px;
    font-weight: 600;
    font-size: 0.85rem;
    cursor: pointer;

    &:hover:not(:disabled) {
      background: var(--color-accent-hover);
    }

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  }

  .draft-labels {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .draft-label-chips {
    display: flex;
    gap: 4px;
    flex-wrap: wrap;
  }

  .draft-label {
    font-size: 0.7rem;
    font-weight: 600;
    padding: 4px 8px;
    border-radius: 3px;
    color: var(--color-text-inverse);
    border: none;
    cursor: pointer;
    text-align: center;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 120px;

    &:hover {
      opacity: 0.7;
    }
  }

  .draft-label-add {
    position: relative;
  }

  .draft-add-label-btn {
    all: unset;
    font-size: 0.78rem;
    color: var(--color-text-tertiary);
    cursor: pointer;

    &:hover {
      color: var(--color-text-secondary);
    }
  }

  .draft-label-menu {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    min-width: 180px;
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: 4px;
    padding: 4px 0;
    z-index: 10;
    max-height: 200px;
    overflow-y: auto;
  }

  .draft-label-option {
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

  .draft-label-swatch {
    width: 10px;
    height: 10px;
    border-radius: 2px;
    flex-shrink: 0;
  }

</style>
