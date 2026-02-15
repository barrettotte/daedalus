<script lang="ts">
  // Full card editor modal. Inline title/body editing, checklist toggling, and arrow-key navigation.

  import {
    selectedCard, draftListKey, updateCardInBoard,
    removeCardFromBoard, boardData, sortedListKeys, listOrder,
    focusedCard, openInEditMode, addToast,
  } from "../stores/board";
  import {
    GetCardContent, SaveCard, OpenFileExternal, DeleteCard,
  } from "../../wailsjs/go/main/App";
  import { marked } from "marked";
  import { autoFocus, backdropClose } from "../lib/utils";
  import type { daedalus } from "../../wailsjs/go/models";
  import ChecklistSection from "./ChecklistSection.svelte";
  import CardSidebar from "./CardSidebar.svelte";
  import Icon from "./Icon.svelte";

  let bodyHtml = $state("");
  let rawBody = $state("");
  let loading = $state(false);
  let backdropEl: HTMLDivElement | undefined = $state(undefined);
  let prevCardId: number | null = $state(null);

  // Edit state
  let editingTitle = $state(false);
  let editingBody = $state(false);
  let editTitle = $state("");
  let editBody = $state("");

  // Live character and word counts for the body edit textarea.
  let charCount = $derived(editingBody ? editBody.length : 0);
  let wordCount = $derived(editingBody && editBody.trim() ? editBody.trim().split(/\s+/).length : 0);

  // Delete confirmation state
  let confirmingDelete = $state(false);

  // Move-to-list dropdown state
  let moveDropdownOpen = $state(false);

  // Guards against stale async responses when rapidly switching cards.
  let loadGeneration = 0;

  // Closes the card detail modal.
  function close(): void {
    editingTitle = false;
    editingBody = false;
    selectedCard.set(null);
  }

  // Opens both title and body for inline editing.
  function startEditAll(): void {
    editTitle = meta!.title;
    editBody = rawBody;
    editingTitle = true;
    editingBody = true;
  }

  // Opens the title for inline editing.
  function startEditTitle(): void {
    editTitle = meta!.title;
    editingTitle = true;
  }

  // Saves the title on blur, or discards if unchanged. Defaults blank titles to the card number.
  async function blurTitle(): Promise<void> {
    editingTitle = false;

    if (!editTitle || !editTitle.trim()) {
      editTitle = String(meta!.id);
    }
    if (editTitle === meta!.title) {
      return;
    }

    const fullBody = `# ${editTitle}\n\n${rawBody}`;
    const updatedMeta = { ...meta!, title: editTitle } as daedalus.CardMetadata;

    try {
      const result = await SaveCard($selectedCard!.filePath, updatedMeta, fullBody);
      updateCardInBoard(result);
      selectedCard.set(result);
    } catch (e) {
      addToast(`Failed to save title: ${e}`);
    }
  }

  // Opens the description for inline editing.
  function startEditBody(): void {
    editBody = rawBody;
    editingBody = true;
  }

  // Saves the body on blur, or discards if unchanged.
  async function blurBody(): Promise<void> {
    editingBody = false;

    if (editBody === rawBody) {
      return;
    }

    const fullBody = `# ${meta!.title}\n\n${editBody}`;

    try {
      const result = await SaveCard($selectedCard!.filePath, { ...meta! } as daedalus.CardMetadata, fullBody);
      updateCardInBoard(result);
      selectedCard.set(result);
      rawBody = editBody;
      bodyHtml = marked.parse(editBody, { async: false });
    } catch (e) {
      addToast(`Failed to save body: ${e}`);
    }
  }

  // Saves the counter (or removes it when null) and persists to disk.
  async function saveCounter(counter: daedalus.Counter | null): Promise<void> {
    const updatedMeta = { ...meta!, counter: counter ?? undefined } as daedalus.CardMetadata;
    const fullBody = `# ${meta!.title}\n\n${rawBody}`;

    try {
      const result = await SaveCard($selectedCard!.filePath, updatedMeta, fullBody);
      updateCardInBoard(result);
      selectedCard.set(result);
    } catch (e) {
      addToast(`Failed to save counter: ${e}`);
    }
  }

  // Saves or clears the card icon and persists to disk.
  async function saveIcon(icon: string): Promise<void> {
    const updatedMeta = { ...meta!, icon } as daedalus.CardMetadata;
    const fullBody = `# ${meta!.title}\n\n${rawBody}`;

    try {
      const result = await SaveCard($selectedCard!.filePath, updatedMeta, fullBody);
      updateCardInBoard(result);
      selectedCard.set(result);
    } catch (e) {
      addToast(`Failed to save icon: ${e}`);
    }
  }

  // Saves or clears the estimated time and persists to disk.
  async function saveEstimate(estimate: number | null): Promise<void> {
    const updatedMeta = {
      ...meta!,
      estimate: estimate ?? undefined,
    } as daedalus.CardMetadata;
    const fullBody = `# ${meta!.title}\n\n${rawBody}`;

    try {
      const result = await SaveCard($selectedCard!.filePath, updatedMeta, fullBody);
      updateCardInBoard(result);
      selectedCard.set(result);
    } catch (e) {
      addToast(`Failed to save estimate: ${e}`);
    }
  }

  // Saves due date and/or date range changes and persists to disk.
  async function saveDates(due: string | null, range: { start: string; end: string } | null): Promise<void> {
    const updatedMeta = {
      ...meta!,
      due: due ?? undefined,
      range: range ?? undefined,
    } as daedalus.CardMetadata;

    const fullBody = `# ${meta!.title}\n\n${rawBody}`;

    try {
      const result = await SaveCard($selectedCard!.filePath, updatedMeta, fullBody);
      updateCardInBoard(result);
      selectedCard.set(result);
    } catch (e) {
      addToast(`Failed to save dates: ${e}`);
    }
  }

  // Toggles a checklist item's done state and saves immediately.
  async function toggleCheckItem(idx: number): Promise<void> {
    const updatedChecklist = meta!.checklist!.map((item, i) => {
      if (i === idx) {
        return { ...item, done: !item.done };
      }
      return { ...item };
    });

    const updatedMeta = { ...meta!, checklist: updatedChecklist } as daedalus.CardMetadata;
    const fullBody = `# ${meta!.title}\n\n${rawBody}`;

    try {
      const result = await SaveCard($selectedCard!.filePath, updatedMeta, fullBody);
      updateCardInBoard(result);
      selectedCard.set(result);
    } catch (e) {
      addToast(`Failed to toggle checklist item: ${e}`);
    }
  }

  // Navigates to prev/next card in the same list while the detail modal is open.
  function navigateCard(delta: number): void {
    const focus = $focusedCard;
    if (!focus) {
      return;
    }

    const cards = $boardData[focus.listKey] || [];
    const newIndex = focus.cardIndex + delta;

    if (newIndex < 0 || newIndex >= cards.length) {
      return;
    }

    focusedCard.set({ listKey: focus.listKey, cardIndex: newIndex });
    selectedCard.set(cards[newIndex]);
  }

  // Navigates to the same-index card in an adjacent list, skipping empty lists.
  function navigateList(delta: number): void {
    const focus = $focusedCard;
    if (!focus) {
      return;
    }

    const keys = sortedListKeys($boardData, $listOrder);
    const listIdx = keys.indexOf(focus.listKey);
    let targetIdx = listIdx + delta;

    // Skip empty lists
    while (targetIdx >= 0 && targetIdx < keys.length) {
      if (($boardData[keys[targetIdx]] || []).length > 0) {
        break;
      }
      targetIdx += delta;
    }

    if (targetIdx < 0 || targetIdx >= keys.length) {
      return;
    }

    const targetKey = keys[targetIdx];
    const targetCards = $boardData[targetKey] || [];
    const clampedIndex = Math.min(focus.cardIndex, targetCards.length - 1);

    focusedCard.set({ listKey: targetKey, cardIndex: clampedIndex });
    selectedCard.set(targetCards[clampedIndex]);
  }

  // Handles keyboard shortcuts: Escape closes/cancels, arrows navigate.
  function handleKeydown(e: KeyboardEvent): void {
    // Only handle keys when the card detail modal is open (not during draft).
    if (!$selectedCard || $draftListKey) {
      return;
    }

    // Arrow navigation only when not editing
    if (!editingTitle && !editingBody && !confirmingDelete) {
      if (e.key === "ArrowUp") {
        e.preventDefault();
        navigateCard(-1);
        return;
      }
      if (e.key === "ArrowDown") {
        e.preventDefault();
        navigateCard(1);
        return;
      }
      if (e.key === "ArrowLeft") {
        e.preventDefault();
        navigateList(-1);
        return;
      }
      if (e.key === "ArrowRight") {
        e.preventDefault();
        navigateList(1);
        return;
      }
    }

    if (e.key === "Escape") {
      if (moveDropdownOpen) {
        moveDropdownOpen = false;
      } else if (editingTitle) {
        editingTitle = false;
      } else if (editingBody) {
        editingBody = false;
      } else if (confirmingDelete) {
        confirmingDelete = false;
      } else {
        close();
      }
    }
  }

  // Loads card content when a different card is selected. Skips reload when
  // the same card is updated (e.g. checklist toggle, counter change).
  $effect(() => {
    if (!$selectedCard) {
      prevCardId = null;
      return;
    }

    const cardId = $selectedCard.metadata.id;
    if (cardId === prevCardId) {
      return;
    }

    // Scroll to top when opening a new card
    if (backdropEl) {
      backdropEl.scrollTop = 0;
    }
    prevCardId = cardId;

    const gen = ++loadGeneration;
    loading = true;
    bodyHtml = "";
    rawBody = "";
    editingTitle = false;
    editingBody = false;
    confirmingDelete = false;
    moveDropdownOpen = false;

    const shouldEdit = $openInEditMode;
    if (shouldEdit) {
      openInEditMode.set(false);
    }

    GetCardContent($selectedCard.filePath).then(content => {
      if (gen !== loadGeneration) {
        return;
      }

      // Strip leading h1 since title is already in the modal header
      const body = (content || "").replace(/^#\s+.*\n*/, "");
      rawBody = body;
      bodyHtml = marked.parse(body, { async: false });
      loading = false;

      // If opened via E shortcut, start in edit mode
      if (shouldEdit) {
        editTitle = meta!.title;
        editBody = body;
        editingTitle = true;
        editingBody = true;
      }

    }).catch(() => {
      if (gen !== loadGeneration) {
        return;
      }
      bodyHtml = "<p><em>Could not load card content.</em></p>";
      rawBody = "";
      loading = false;
    });
  });

  // Opens the card's markdown file in the system default editor.
  function openExternal(): void {
    OpenFileExternal($selectedCard!.filePath).catch(e => addToast(`Failed to open file: ${e}`));
  }

  // Deletes the card from disk and removes it from the board store.
  async function deleteCard(): Promise<void> {
    const filePath = $selectedCard!.filePath;
    try {
      await DeleteCard(filePath);
      removeCardFromBoard(filePath);
      close();
    } catch (e) {
      addToast(`Failed to delete card: ${e}`);
    }
  }

  let meta = $derived($selectedCard ? $selectedCard.metadata : null);

</script>

<svelte:window onkeydown={handleKeydown} />

{#if $selectedCard && meta}
  <div class="modal-backdrop scrollable" bind:this={backdropEl} role="presentation" use:backdropClose={close} onkeydown={handleKeydown}>
    <div class="modal-dialog size-lg card-detail-dialog" role="dialog">
      <div class="modal-header card-detail-header">
        {#if editingTitle}
          <input class="edit-title-input" type="text" bind:value={editTitle} onblur={blurTitle}
            onkeydown={e => e.key === 'Enter' && (e.target as HTMLInputElement).blur()} use:autoFocus
          />
        {:else}
          <button class="card-title clickable" title="Click to edit title" onclick={startEditTitle}>{meta.title}</button>
        {/if}
        <div class="header-btns">
          {#if !loading}
            <button class="modal-close" onclick={startEditAll} title="Edit">
              <Icon name="pencil" size={16} />
            </button>
          {/if}
          <button class="modal-close" onclick={openExternal} title="Open in editor">
            <Icon name="external-link" size={16} />
          </button>
          <button class="modal-close delete-icon" onclick={() => confirmingDelete = true} title="Delete card">
            <Icon name="trash" size={12} />
          </button>
          <button class="modal-close" onclick={close} title="Close">
            <Icon name="close" size={16} />
          </button>
        </div>
      </div>

      {#if confirmingDelete}
        <div class="confirm-delete">
          <span class="confirm-delete-text">Delete this card?</span>
          <div class="confirm-delete-btns">
            <button class="delete-btn" onclick={deleteCard}>Delete</button>
            <button class="cancel-btn" onclick={() => confirmingDelete = false}>Cancel</button>
          </div>
        </div>
      {:else}
      <div class="modal-body">
        <div class="main-col">

          <!-- Description -->
          <div class="section">
            {#if editingBody}
              <textarea class="edit-body-textarea" bind:value={editBody} onblur={blurBody} placeholder="Card description (markdown)" use:autoFocus></textarea>
              <div class="edit-footer">{charCount} chars, {wordCount} words</div>
            {:else if loading}
              <p class="loading-text">Loading...</p>
            {:else if bodyHtml.trim()}
              <div class="markdown-body clickable" role="button" tabindex="0" title="Click to edit description"
                onclick={startEditBody} onkeydown={e => e.key === 'Enter' && startEditBody()}
              >{@html bodyHtml}</div>
            {:else}
              <button class="empty-desc clickable" title="Click to add description" onclick={startEditBody}>Enter description...</button>
            {/if}
          </div>

          <!-- Checklist -->
          {#if meta.checklist && meta.checklist.length > 0}
            <div class="section">
              <ChecklistSection checklist={meta.checklist} ontoggle={toggleCheckItem}/>
            </div>
          {/if}
        </div>

        <CardSidebar {meta} bind:moveDropdownOpen
          onsavecounter={saveCounter} onsavedates={saveDates}
          onsaveestimate={saveEstimate} onsaveicon={saveIcon}
        />
      </div>
      {/if}
    </div>
  </div>
{/if}

<style lang="scss">
  .card-detail-dialog {
    margin-bottom: 48px;
    position: relative;
  }

  .card-detail-header {
    align-items: flex-start;
    gap: 12px;
    border-bottom: none;
    padding: 16px 16px 12px 20px;
  }

  .card-title {
    all: unset;
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--color-text-primary);
    line-height: 1.3;
    word-break: break-word;
    flex: 1;
    min-width: 0;
    padding-top: 4px;
    text-align: left;
  }

  .clickable {
    cursor: pointer;

    &:hover {
      outline: 1px solid var(--overlay-hover-medium);
      outline-offset: 4px;
      border-radius: 4px;
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

  .main-col {
    flex: 1;
    min-width: 0;
  }

  /* Sections */
  .section {
    margin-bottom: 20px;
  }

  .edit-footer {
    text-align: right;
    color: var(--color-text-muted);
    font-size: 0.75rem;
    padding: 4px 12px;
  }

  /* Markdown body */
  .loading-text {
    color: var(--color-text-muted);
    font-size: 0.85rem;
  }

  .empty-desc {
    all: unset;
    color: var(--color-text-muted);
    font-size: 0.85rem;
    font-style: italic;
    text-align: left;
  }

  .markdown-body {
    line-height: 1.6;
    font-size: 0.9rem;
    color: var(--color-text-secondary);
    background: var(--overlay-subtle);
    border-radius: 6px;
    padding: 12px 16px;
    overflow-x: auto;
    white-space: nowrap;

    :global(h1),
    :global(h2),
    :global(h3) {
      color: var(--color-text-primary);
      margin-top: 16px;
      margin-bottom: 8px;
    }

    :global(h1) {
      font-size: 1.2rem;
    }
    :global(h2) {
      font-size: 1.05rem;
    }
    :global(h3) {
      font-size: 0.95rem;
    }

    :global(a) {
      color: var(--color-accent);
    }

    :global(a:hover) {
      text-decoration: underline;
    }

    :global(code) {
      background: var(--color-bg-base);
      padding: 2px 6px;
      border-radius: 3px;
      font-size: 0.85em;
    }

    :global(pre) {
      background: var(--color-bg-base);
      padding: 12px;
      border-radius: 6px;
      overflow-x: auto;

      :global(code) {
        padding: 0;
        background: none;
      }
    }

    :global(blockquote) {
      border-left: 3px solid var(--color-accent);
      margin: 8px 0;
      padding: 4px 12px;
      color: var(--color-text-tertiary);
    }

    :global(ul), :global(ol) {
      padding-left: 20px;
    }

    :global(li) {
      margin-bottom: 2px;
    }

    :global(img) {
      max-width: 100%;
      border-radius: 4px;
    }

    :global(table) {
      border-collapse: collapse;
      width: 100%;
      margin: 8px 0;
    }

    :global(th), :global(td) {
      border: 1px solid var(--color-border);
      padding: 6px 10px;
      text-align: left;
    }

    :global(th) {
      background: var(--color-bg-base);
    }

    :global(hr) {
      border: none;
      border-top: 1px solid var(--color-border);
      margin: 16px 0;
    }
  }

  /* Delete confirmation */
  .delete-icon:hover {
    color: var(--color-error);
  }

  .confirm-delete {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 20px;
    gap: 12px;
  }

  .confirm-delete-text {
    font-size: 0.95rem;
    color: var(--color-text-primary);
    font-weight: 600;
  }

  .confirm-delete-btns {
    display: flex;
    gap: 8px;
  }

  .delete-btn {
    background: var(--color-error-dark);
    color: #fff;
    border: none;
    padding: 8px 20px;
    border-radius: 4px;
    font-weight: 600;
    font-size: 0.85rem;
    cursor: pointer;

    &:hover {
      background: var(--color-error);
    }
  }
</style>
