<script lang="ts">
  // Modal for creating a new card with title, body, and sidebar metadata.

  import {
    selectedCard, draftListKey, draftPosition,
    addCardToBoard, boardConfig, boardData, addToast, isAtLimit,
  } from "../stores/board";
  import { CreateCard, SaveCard } from "../../wailsjs/go/main/App";
  import { autoFocus, backdropClose, wordCount } from "../lib/utils";
  import type { daedalus } from "../../wailsjs/go/models";
  import {
    toggleChecklistItem, addChecklistItem, editChecklistItem,
    removeChecklistItem, reorderChecklistItem,
  } from "../lib/checklist";
  import Icon from "./Icon.svelte";
  import DraftSidebar from "./DraftSidebar.svelte";
  import ChecklistSection from "./ChecklistSection.svelte";

  let draftTitle = $state("");
  let draftBody = $state("");
  let saving = $state(false);

  // Live character and word counts for the body textarea.
  let charCount = $derived(draftBody.length);
  let wCount = $derived(wordCount(draftBody));

  // Sidebar metadata fields bound to DraftSidebar.
  let draftLabels = $state<string[]>([]);
  let draftIcon = $state("");
  let draftDue = $state<string | null>(null);
  let draftRange = $state<{ start: string; end: string } | null>(null);
  let draftEstimate = $state<number | null>(null);
  let draftCounter = $state<daedalus.Counter | null>(null);
  let draftChecklist = $state<daedalus.CheckListItem[] | null>(null);

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

  // Local state that syncs with stores so DraftSidebar can bind to them.
  let localListKey = $state("");
  let localPosition = $state("top");

  // Sync store -> local when draft opens.
  $effect(() => {
    if ($draftListKey) {
      localListKey = $draftListKey;
    }
  });
  $effect(() => {
    if ($draftPosition) {
      localPosition = $draftPosition;
    }
  });

  // Sync local -> store when user changes via DraftSidebar.
  $effect(() => {
    if (localListKey && $draftListKey && localListKey !== $draftListKey) {
      draftListKey.set(localListKey);
    }
  });
  $effect(() => {
    if (localPosition !== $draftPosition) {
      draftPosition.set(localPosition);
    }
  });

  // Closes the draft modal and clears all draft state.
  function close(): void {
    draftTitle = "";
    draftBody = "";
    draftLabels = [];
    draftIcon = "";
    draftDue = null;
    draftRange = null;
    draftEstimate = null;
    draftCounter = null;
    draftChecklist = null;
    saving = false;
    draftListKey.set(null);
    selectedCard.set(null);
  }

  // Validates and saves the draft card to disk, then adds it to the board store.
  async function saveDraft(): Promise<void> {
    if (!draftTitle.trim()) {
      return;
    }
    if (isAtLimit($draftListKey!, $boardData, $boardConfig)) {
      addToast("List is at its card limit");
      return;
    }
    saving = true;

    try {
      const pos = $draftPosition;
      let card = await CreateCard($draftListKey!, draftTitle.trim(), draftBody, pos);

      // Persist extra metadata if any sidebar fields were filled in.
      const hasExtraMeta = draftLabels.length > 0 || draftIcon || draftDue || draftRange
        || draftEstimate != null || draftCounter || draftChecklist;
      if (hasExtraMeta) {
        const fullBody = `# ${draftTitle.trim()}\n\n${draftBody}`;

        const meta = {
          ...card.metadata,
          labels: draftLabels.length > 0 ? draftLabels : card.metadata.labels,
          icon: draftIcon || card.metadata.icon,
          due: draftDue || card.metadata.due,
          range: draftRange || card.metadata.range,
          estimate: draftEstimate ?? card.metadata.estimate,
          counter: draftCounter || card.metadata.counter,
          checklist: draftChecklist || card.metadata.checklist,
        } as daedalus.CardMetadata;

        card = await SaveCard(card.filePath, meta, fullBody);
      }

      addCardToBoard($draftListKey!, card, pos);
      addToast("Card created", "success");
      close();
    } catch (e) {
      addToast(`Failed to create card: ${e}`);
    }
    saving = false;
  }

  // Checklist handlers -- mutate draftChecklist directly since there's no backend yet.
  function toggleCheckItem(idx: number): void {
    if (!draftChecklist) {
      return;
    }
    draftChecklist = toggleChecklistItem(draftChecklist, idx);
  }

  function addCheckItem(desc: string): void {
    if (!draftChecklist) {
      return;
    }
    draftChecklist = addChecklistItem(draftChecklist, desc);
  }

  function removeCheckItem(idx: number): void {
    if (!draftChecklist) {
      return;
    }
    draftChecklist = removeChecklistItem(draftChecklist, idx);
  }

  function editCheckItem(idx: number, desc: string): void {
    if (!draftChecklist) {
      return;
    }
    draftChecklist = editChecklistItem(draftChecklist, idx, desc);
  }

  function reorderCheckItem(fromIdx: number, toIdx: number): void {
    if (!draftChecklist) {
      return;
    }
    draftChecklist = reorderChecklistItem(draftChecklist, fromIdx, toIdx);
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
</script>

<svelte:window onkeydown={handleKeydown} />

{#if $draftListKey}
  <div class="modal-backdrop scrollable" role="presentation" use:backdropClose={close} onkeydown={handleKeydown}>
    <div class="modal-dialog size-lg draft-dialog" role="dialog">
      <div class="modal-header card-editor">
        <input class="edit-title-input" type="text" bind:value={draftTitle} placeholder="Card title"
          onkeydown={e => e.key === 'Enter' && saveDraft()} use:autoFocus
        />
        <div class="header-btns">
          <button class="modal-close" onclick={close} title="Close">
            <Icon name="close" size={16} />
          </button>
        </div>
      </div>
      <div class="modal-body">
        <div class="main-col">
          <textarea class="edit-body-textarea" bind:value={draftBody} placeholder="Card description (markdown)"></textarea>
          <div class="edit-footer">
            <span>{charCount} chars, {wCount} words</span>
          </div>
          {#if draftChecklist}
            <ChecklistSection
              checklist={draftChecklist}
              ontoggle={toggleCheckItem}
              onadd={addCheckItem}
              onremove={removeCheckItem}
              onedit={editCheckItem}
              onreorder={reorderCheckItem}
              ondelete={() => { draftChecklist = null; }}
            />
          {/if}
        </div>
        <DraftSidebar
          {nextCardId}
          bind:draftListKey={localListKey}
          bind:draftPosition={localPosition}
          bind:draftLabels
          bind:draftIcon
          bind:draftDue
          bind:draftRange
          bind:draftEstimate
          bind:draftCounter
          bind:draftChecklist
        />
      </div>
      <div class="draft-footer">
        <button class="save-btn" onclick={saveDraft} disabled={saving || !draftTitle.trim()}>
          {saving ? "Saving..." : "Create"}
        </button>
        <button class="cancel-btn" onclick={close}>Cancel</button>
      </div>
    </div>
  </div>
{/if}

<style lang="scss">
  .draft-dialog {
    margin-bottom: 48px;

    :global(.edit-title-input) {
      max-width: calc(100% - 220px);
    }
  }

  .main-col {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .draft-footer {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
    padding: 0 20px 20px 20px;
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

</style>
