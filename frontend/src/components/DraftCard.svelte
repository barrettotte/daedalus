<script lang="ts">
  // Modal for creating a new card with title, body, and sidebar metadata.

  import {
    selectedCard, draftListKey, draftPosition,
    addCardToBoard, boardConfig, boardData, addToast, isAtLimit, maxCardId,
  } from "../stores/board";
  import { CreateCard, SaveCard } from "../../wailsjs/go/main/App";
  import { autoFocus, backdropClose, wordCount, createBlurGuard, copyToClipboard } from "../lib/utils";
  import type { daedalus } from "../../wailsjs/go/models";
  import {
    toggleChecklistItem, addChecklistItem, editChecklistItem,
    removeChecklistItem, reorderChecklistItem,
  } from "../lib/checklist";
  import Icon from "./Icon.svelte";
  import DraftSidebar from "./DraftSidebar.svelte";
  import ChecklistSection from "./ChecklistSection.svelte";
  import TimeSeriesSection from "./TimeSeriesSection.svelte";

  let draftTitle = $state("");
  let draftBody = $state("");
  let draftUrl = $state("");
  let editingUri = $state(false);
  let editingBody = $state(false);
  let saving = $state(false);

  // Suppress blur when a context menu just opened so right-click paste works.
  const { oncontextmenu: onFieldContextMenu, guardedBlur } = createBlurGuard();

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
  let draftChecklistTitle = $state("");
  let draftTimeSeries = $state<daedalus.TimeSeries | null>(null);

  // Speculative ID for the next card (current max + 1).
  let nextCardId = $derived($maxCardId + 1);

  // Closes the draft modal and clears all draft state.
  function close(): void {
    draftTitle = "";
    draftBody = "";
    draftUrl = "";
    editingUri = false;
    editingBody = false;
    draftLabels = [];
    draftIcon = "";
    draftDue = null;
    draftRange = null;
    draftEstimate = null;
    draftCounter = null;
    draftChecklist = null;
    draftChecklistTitle = "";
    draftTimeSeries = null;
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
      const trimmedUrl = draftUrl.trim();
      const hasExtraMeta = draftLabels.length > 0 || draftIcon || draftDue || draftRange
        || draftEstimate != null || draftCounter || draftChecklist || draftTimeSeries || trimmedUrl;
      if (hasExtraMeta) {
        const fullBody = `# ${draftTitle.trim()}\n\n${draftBody}`;

        const meta = {
          ...card.metadata,
          labels: draftLabels.length > 0 ? draftLabels : card.metadata.labels,
          icon: draftIcon ?? card.metadata.icon,
          url: trimmedUrl ?? card.metadata.url,
          due: draftDue ?? card.metadata.due,
          range: draftRange ?? card.metadata.range,
          estimate: draftEstimate ?? card.metadata.estimate,
          counter: draftCounter ?? card.metadata.counter,
          checklist: draftChecklist ?? card.metadata.checklist,
          checklist_title: draftChecklistTitle || card.metadata.checklist_title,
          timeseries: draftTimeSeries ?? card.metadata.timeseries,
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

  // Handles keyboard shortcuts: Ctrl/Cmd+Enter saves draft, Escape closes.
  function handleKeydown(e: KeyboardEvent): void {
    if (!$draftListKey) {
      return;
    }

    if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
      e.preventDefault();
      saveDraft();
    } else if (e.key === "Escape") {
      if (editingBody) {
        editingBody = false;
        return;
      }
      if (editingUri) {
        editingUri = false;
        return;
      }
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
          {#if editingUri}
            <div class="uri-row">
              <Icon name="link" size={14} style="color: var(--color-text-muted); flex-shrink: 0" />
              <input class="uri-input" type="text" placeholder="https://..."
                bind:value={draftUrl}
                onblur={() => guardedBlur(() => { editingUri = false; })}
                oncontextmenu={onFieldContextMenu}
                onkeydown={e => e.key === 'Enter' && (e.target as HTMLInputElement).blur()}
                use:autoFocus
              />
              <button class="uri-action-btn remove" title="Remove URI" onmousedown={(e) => { e.preventDefault(); draftUrl = ""; editingUri = false; }}>
                <Icon name="trash" size={12} />
              </button>
            </div>
          {:else if draftUrl.trim()}
            <div class="uri-row">
              <Icon name="link" size={14} style="color: var(--color-text-muted); flex-shrink: 0" />
              <button class="uri-link" title={draftUrl.trim()}
                onclick={(e) => e.button === 0 && window.open(draftUrl.trim(), "_blank")}
              >{draftUrl.trim()}</button>
              <button class="uri-action-btn" title="Copy URI" onclick={() => copyToClipboard(draftUrl.trim(), "URI")}>
                <Icon name="copy" size={12} />
              </button>
              <button class="uri-action-btn" title="Edit URI" onclick={() => { editingUri = true; }}>
                <Icon name="pencil" size={12} />
              </button>
              <button class="uri-action-btn remove" title="Remove URI" onclick={() => { draftUrl = ""; }}>
                <Icon name="trash" size={12} />
              </button>
            </div>
          {/if}

          <div class="section">
            {#if editingBody}
              <textarea class="edit-body-textarea" bind:value={draftBody} placeholder="Card description (markdown)" use:autoFocus></textarea>
              <div class="edit-footer">
                <span>{charCount} chars, {wCount} words</span>
                <button class="save-body-btn" title="Done" onclick={() => { editingBody = false; }}>
                  <Icon name="check" size={12} /> Done
                </button>
              </div>
            {:else if draftBody.trim()}
              <div class="desc-wrapper">
                <div class="desc-actions">
                  <button class="uri-action-btn" title="Edit description" onclick={() => { editingBody = true; }}>
                    <Icon name="pencil" size={12} />
                  </button>
                  <button class="uri-action-btn remove" title="Delete description" onclick={() => { draftBody = ""; }}>
                    <Icon name="trash" size={12} />
                  </button>
                </div>
                <div class="desc-preview" role="button" tabindex="0"
                  onclick={() => { editingBody = true; }}
                  onkeydown={e => e.key === 'Enter' && (editingBody = true)}
                >{draftBody.trim()}</div>
              </div>
            {:else}
              <button class="empty-desc" title="Click to add description" onclick={() => { editingBody = true; }}>
                <Icon name="pencil" size={12} /> Enter description...
              </button>
            {/if}
          </div>
          {#if draftChecklist}
            <ChecklistSection
              checklist={draftChecklist}
              title={draftChecklistTitle}
              ontoggle={(idx) => { draftChecklist = toggleChecklistItem(draftChecklist!, idx); }}
              onadd={(desc) => { draftChecklist = addChecklistItem(draftChecklist || [], desc); }}
              onremove={(idx) => { draftChecklist = removeChecklistItem(draftChecklist!, idx); }}
              onedit={(idx, desc) => { draftChecklist = editChecklistItem(draftChecklist!, idx, desc); }}
              onreorder={(from, to) => { draftChecklist = reorderChecklistItem(draftChecklist!, from, to); }}
              ondelete={() => { draftChecklist = null; }}
              ontitlechange={(t) => { draftChecklistTitle = t; }}
            />
          {/if}
          {#if draftTimeSeries}
            <TimeSeriesSection
              timeseries={draftTimeSeries}
              onsave={(ts) => { draftTimeSeries = ts; }}
            />
          {/if}
        </div>
        <DraftSidebar
          {nextCardId}
          hasUrl={!!draftUrl.trim()}
          draftUrl={draftUrl.trim()}
          onstartedituri={() => { editingUri = true; }}
          onremoveuri={() => { draftUrl = ""; editingUri = false; }}
          {editingUri}
          draftListKey={$draftListKey || ""}
          draftPosition={$draftPosition}
          onlistkeychange={(key) => { draftListKey.set(key); }}
          onpositionchange={(pos) => { draftPosition.set(pos); }}
          bind:draftLabels
          bind:draftIcon
          bind:draftDue
          bind:draftRange
          bind:draftEstimate
          bind:draftCounter
          bind:draftChecklist
          bind:draftChecklistTitle
          bind:draftTimeSeries
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

  /* Description */
  .section {
    margin-bottom: 0;
  }

  .desc-preview {
    font-size: 0.82rem;
    color: var(--color-text-secondary);
    white-space: pre-wrap;
    word-break: break-word;
    padding: 6px 8px;
    border-radius: 4px;
    cursor: pointer;
    background: var(--overlay-subtle);
    border: 1px solid var(--color-border);

    &:hover {
      border-color: var(--color-text-tertiary);
    }
  }

  .save-body-btn {
    background: none;
    color: var(--color-text-muted);
    font-family: inherit;
    font-size: 0.72rem;
    font-weight: 600;
    padding: 2px 6px;
    border-radius: 3px;

    &:hover {
      color: var(--color-text-primary);
      background: var(--overlay-hover-light);
    }
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
