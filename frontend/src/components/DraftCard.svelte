<script lang="ts">
  import {
    selectedCard, draftListKey, draftPosition,
    addCardToBoard, boardConfig, boardData, addToast, isAtLimit,
  } from "../stores/board";
  import { CreateCard } from "../../wailsjs/go/main/App";
  import { formatListName, autoFocus } from "../lib/utils";

  // Draft mode state for creating new cards before they exist on disk
  let draftTitle = $state("");
  let draftBody = $state("");
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

  // Closes the draft modal and clears all draft state.
  function close(): void {
    draftTitle = "";
    draftBody = "";
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
      const card = await CreateCard($draftListKey!, draftTitle.trim(), draftBody, pos);
      addCardToBoard($draftListKey!, card, pos);

      draftTitle = "";
      draftBody = "";
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

  // Tracks whether mousedown started on the backdrop.
  let mouseDownOnBackdrop = false;

  // Records that mousedown landed directly on the backdrop.
  function handleBackdropMousedown(e: MouseEvent): void {
    mouseDownOnBackdrop = e.target === e.currentTarget;
  }

  // Closes the modal only if both mousedown and mouseup targeted the backdrop.
  function handleBackdropMouseup(e: MouseEvent): void {
    if (mouseDownOnBackdrop && e.target === e.currentTarget) {
      close();
    }
    mouseDownOnBackdrop = false;
  }

  // Resets draft fields only on transition from null to non-null (new draft started).
  let prevDraftListKey: string | null = null;
  $effect(() => {
    const current = $draftListKey;
    if (current && current !== prevDraftListKey) {
      draftTitle = "";
      draftBody = "";
      saving = false;
    }
    prevDraftListKey = current;
  });
</script>

<svelte:window onkeydown={handleKeydown} />

{#if $draftListKey}
  <div class="backdrop" role="presentation"
    onmousedown={handleBackdropMousedown}
    onmouseup={handleBackdropMouseup}
    onkeydown={handleKeydown}
  >
    <div class="modal" role="dialog">
      <div class="modal-header">
        <div class="draft-header-col">
          <div class="draft-list-name">
            Drafting a card in <strong>{draftListDisplayName}</strong>
          </div>
          <input class="edit-title-input" type="text" bind:value={draftTitle} placeholder="Card title"
            onkeydown={e => e.key === 'Enter' && saveDraft()} use:autoFocus
          />
        </div>
        <div class="header-btns">
          <button class="header-btn" onclick={close} title="Close">
            <svg viewBox="0 0 24 24" width="16" height="16">
              <line x1="18" y1="6" x2="6" y2="18" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
              <line x1="6" y1="6" x2="18" y2="18" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            </svg>
          </button>
        </div>
      </div>
      <div class="modal-body">
        <div class="main-col">
          <div class="section">
            <textarea class="edit-body-textarea" bind:value={draftBody} placeholder="Card description (markdown)"></textarea>
          </div>
          <div class="draft-actions">
            <div class="position-section">
              <span class="position-label">Position</span>
              <div class="position-toggle">
                <button class="pos-btn" title="Add card to top of list" class:active={$draftPosition === 'top'} onclick={() => draftPosition.set('top')}>Top</button>
                <button class="pos-btn" title="Add card to bottom of list" class:active={$draftPosition === 'bottom'} onclick={() => draftPosition.set('bottom')}>Bottom</button>
              </div>
              <div class="position-specific-row">
                <input class="position-input" type="number" min="1" max={draftListCount + 1}
                  value={positionDisplayValue} oninput={handlePositionInput}
                />
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
    </div>
  </div>
{/if}

<style lang="scss">
  .backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: var(--overlay-backdrop);
    display: flex;
    align-items: flex-start;
    justify-content: center;
    z-index: 1000;
    padding-top: 48px;
    overflow-y: auto;
  }

  .modal {
    background: var(--color-bg-elevated);
    border-radius: 8px;
    max-width: 720px;
    width: 95%;
    position: relative;
    color: var(--color-text-secondary);
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI",
      Roboto, Oxygen, Ubuntu, sans-serif;
    margin-bottom: 48px;
    text-align: left;
  }

  .modal-header {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    padding: 16px 16px 12px 20px;
  }

  .modal-body {
    display: flex;
    gap: 16px;
    padding: 0 20px 20px 20px;
  }

  .main-col {
    flex: 1;
    min-width: 0;
  }

  .section {
    margin-bottom: 20px;
  }

  .header-btns {
    display: flex;
    gap: 4px;
    flex-shrink: 0;
  }

  .header-btn {
    background: var(--overlay-hover);
    border: none;
    color: var(--color-text-secondary);
    cursor: pointer;
    width: 32px;
    height: 32px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;

    &:hover {
      background: var(--overlay-hover-strong);
      color: var(--color-text-primary);
    }
  }

  .draft-header-col {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .draft-list-name {
    font-size: 0.78rem;
    color: var(--color-text-tertiary);

    strong {
      color: var(--color-text-secondary);
    }
  }

  .edit-title-input {
    flex: 1;
    min-width: 0;
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

  .edit-body-textarea {
    width: 100%;
    min-height: 200px;
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    color: var(--color-text-secondary);
    font-size: 0.9rem;
    font-family: monospace;
    padding: 12px;
    border-radius: 6px;
    outline: none;
    resize: vertical;
    box-sizing: border-box;
    line-height: 1.5;

    &:focus {
      border-color: var(--color-accent);
    }
  }

  .draft-actions {
    display: flex;
    align-items: flex-end;
    gap: 8px;
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
    appearance: textfield;
    -moz-appearance: textfield;

    &:focus {
      border-color: var(--color-accent);
    }

    &::-webkit-inner-spin-button,
    &::-webkit-outer-spin-button {
      -webkit-appearance: none;
      margin: 0;
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

  .cancel-btn {
    background: var(--overlay-hover);
    color: var(--color-text-secondary);
    border: none;
    padding: 8px 16px;
    border-radius: 4px;
    font-size: 0.85rem;
    cursor: pointer;

    &:hover {
      background: var(--overlay-hover-strong);
      color: var(--color-text-primary);
    }
  }
</style>
