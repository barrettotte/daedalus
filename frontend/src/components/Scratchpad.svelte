<script lang="ts">
  // Board-level markdown scratchpad modal. Persistent notes stored in scratchpad.md alongside board.yaml.

  import { onMount } from "svelte";
  import { GetScratchpad, SaveScratchpad, OpenFileExternal } from "../../wailsjs/go/main/App";
  import { marked } from "marked";
  import { backdropClose, wordCount, resolveWikiLinks } from "../lib/utils";
  import { addToast, saveWithToast, boardPath, boardData, selectedCard } from "../stores/board";
  import Icon from "./Icon.svelte";

  let { onclose }: { onclose: () => void } = $props();

  let content = $state("");
  let bodyHtml = $state("");
  let editing = $state(false);
  let editText = $state("");
  let loading = $state(true);

  let charCount = $derived(editing ? editText.length : 0);
  let wCount = $derived(editing ? wordCount(editText) : 0);

  onMount(async () => {
    try {
      content = await GetScratchpad();
      bodyHtml = resolveWikiLinks(marked.parse(content, { async: false }) as string, $boardData);
    } catch (e) {
      addToast(`Failed to load scratchpad: ${e}`);
    }
    loading = false;
    startEdit();
  });

  function startEdit(): void {
    editText = content;
    editing = true;
  }

  async function save(): Promise<void> {
    if (editText === content) {
      return;
    }
    content = editText;
    bodyHtml = resolveWikiLinks(marked.parse(content, { async: false }) as string, $boardData);
    await saveWithToast(SaveScratchpad(content), "save scratchpad");
  }

  async function blurEditor(): Promise<void> {
    await save();
    editing = false;
  }

  function handleBodyClick(e: MouseEvent): void {
    const target = e.target as HTMLElement;
    const anchor = target.closest("a");
    if (anchor) {
      const cardId = anchor.dataset.cardId;
      if (cardId) {
        e.preventDefault();
        e.stopPropagation();

        const id = Number(cardId);
        for (const cards of Object.values($boardData)) {
          const found = cards.find(c => c.metadata.id === id);
          if (found) {
            onclose();
            selectedCard.set(found);
            return;
          }
        }
        addToast(`Card #${id} not found`);
        return;
      }
    }
    startEdit();
  }

  function openExternal(): void {
    saveWithToast(OpenFileExternal($boardPath + "/scratchpad.md"), "open file");
  }

  function handleKeydown(e: KeyboardEvent): void {
    if (e.key === "Escape") {
      if (editing) {
        editing = false;
      } else {
        onclose();
      }
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="modal-backdrop centered" role="presentation" use:backdropClose={onclose}>
  <div class="modal-dialog size-lg" role="dialog">
    <div class="modal-header">
      <span class="scratchpad-title">
        <Icon name="notepad" size={16} />
        Scratchpad
      </span>
      <div class="header-btns">
        {#if !loading && !editing}
          <button class="modal-close" onclick={startEdit} title="Edit">
            <Icon name="pencil" size={16} />
          </button>
        {/if}
        <button class="modal-close" onclick={openExternal} title="Open in editor">
          <Icon name="external-link" size={16} />
        </button>
        <button class="modal-close" onclick={onclose} title="Close">
          <Icon name="close" size={16} />
        </button>
      </div>
    </div>

    <div class="scratchpad-body">
      {#if editing}
        <textarea class="edit-body-textarea scratchpad-textarea" bind:value={editText} onblur={blurEditor} placeholder="Write notes here (markdown supported)..."></textarea>
        <div class="edit-footer">
          <span>{charCount} chars, {wCount} words</span>
          <button class="save-body-btn" title="Save" onmousedown={e => { e.preventDefault(); blurEditor(); }}>
            <Icon name="check" size={12} /> Save
          </button>
        </div>
      {:else if loading}
        <p class="loading-text">Loading...</p>
      {:else if bodyHtml.trim()}
        <div class="markdown-body clickable" role="button" tabindex="0" onclick={handleBodyClick}
          onkeydown={e => e.key === 'Enter' && startEdit()}
        >{@html bodyHtml}</div>
      {:else}
        <button class="empty-desc" onclick={startEdit}>Click to add notes...</button>
      {/if}
    </div>
  </div>
</div>

<style lang="scss">
  .scratchpad-title {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 1rem;
    font-weight: 600;
    color: var(--color-text-primary);
  }

  .scratchpad-body {
    padding: 16px 20px 20px 20px;
  }

  .scratchpad-textarea {
    min-height: 400px;
  }

  .loading-text {
    color: var(--color-text-muted);
    font-size: 0.85rem;
  }

  .empty-desc {
    all: unset;
    color: var(--color-text-muted);
    font-size: 0.85rem;
    font-style: italic;
    cursor: pointer;

    &:hover {
      color: var(--color-text-secondary);
    }
  }


</style>
