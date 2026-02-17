<script lang="ts">
  // Full card editor modal. Inline title/body editing, checklist toggling, and arrow-key navigation.

  import {
    selectedCard, draftListKey, updateCardInBoard,
    removeCardFromBoard, boardData, sortedListKeys, listOrder,
    focusedCard, openInEditMode, addToast, saveWithToast,
  } from "../stores/board";
  import {
    GetCardContent, SaveCard, OpenFileExternal, DeleteCard, OpenURI,
  } from "../../wailsjs/go/main/App";
  import { marked } from "marked";
  import { autoFocus, blurOnEnter, backdropClose } from "../lib/utils";

  // Strip title attributes from links to prevent browser tooltips.
  marked.use({
    renderer: {
      link({ href, text }: { href: string; text: string }) {
        return `<a href="${href}" target="_blank" rel="noopener noreferrer">${text}</a>`;
      },
    },
  });
  import type { daedalus } from "../../wailsjs/go/models";
  import {
    toggleChecklistItem, addChecklistItem, editChecklistItem,
    removeChecklistItem, reorderChecklistItem,
  } from "../lib/checklist";
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

  // URI edit state
  let editingUri = $state(false);
  let editUri = $state("");

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

  // Handles clicks on the markdown body -- opens links via system handler, otherwise enters edit mode.
  function handleBodyClick(e: MouseEvent): void {
    const target = e.target as HTMLElement;
    const anchor = target.closest("a");

    if (anchor && anchor.href) {
      e.preventDefault();
      e.stopPropagation();
      saveWithToast(OpenURI(anchor.href), "open link");
      return;
    }
    startEditBody();
  }

  // Saves the body content to disk if changed. Does not close the editor.
  async function saveBody(): Promise<void> {
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
      addToast("Saved", "success");
    } catch (e) {
      addToast(`Failed to save body: ${e}`);
    }
  }

  // Saves the body on blur, then closes the editor.
  async function blurBody(): Promise<void> {
    await saveBody();
    editingBody = false;
  }

  // Persists metadata changes to disk and updates stores.
  async function saveCardMeta(changes: Partial<daedalus.CardMetadata>, errorLabel: string): Promise<void> {
    const updatedMeta = { ...meta!, ...changes } as daedalus.CardMetadata;
    const fullBody = `# ${meta!.title}\n\n${rawBody}`;

    try {
      const result = await SaveCard($selectedCard!.filePath, updatedMeta, fullBody);
      updateCardInBoard(result);
      selectedCard.set(result);
    } catch (e) {
      addToast(`Failed to ${errorLabel}: ${e}`);
    }
  }

  function saveCounter(counter: daedalus.Counter | null): Promise<void> {
    return saveCardMeta({ counter: counter ?? undefined }, "save counter");
  }

  function saveIcon(icon: string): Promise<void> {
    return saveCardMeta({ icon }, "save icon");
  }

  function saveUrl(url: string): Promise<void> {
    return saveCardMeta({ url }, "save URL");
  }

  function startEditUri(): void {
    editUri = meta?.url || "";
    editingUri = true;
  }

  function blurUri(): void {
    editingUri = false;
    const trimmed = editUri.trim();
    if (trimmed !== (meta?.url || "")) {
      saveUrl(trimmed);
    }
  }

  function openUri(): void {
    if (meta?.url) {
      saveWithToast(OpenURI(meta.url), "open URI");
    }
  }

  function removeUri(): void {
    editingUri = false;
    saveUrl("");
  }

  function saveEstimate(estimate: number | null): Promise<void> {
    return saveCardMeta({ estimate: estimate ?? undefined }, "save estimate");
  }

  function saveDates(due: string | null, range: { start: string; end: string } | null): Promise<void> {
    return saveCardMeta(
      { due: due ?? undefined, range: (range as daedalus.DateRange) ?? undefined }, "save dates",
    );
  }

  function saveChecklist(checklist: daedalus.CheckListItem[] | null): Promise<void> {
    return saveCardMeta({
      checklist: checklist ?? undefined,
      checklist_title: checklist ? (meta!.checklist_title || "Checklist") : undefined,
    }, "save checklist");
  }

  function saveChecklistTitle(title: string): Promise<void> {
    return saveCardMeta({ checklist_title: title }, "save checklist title");
  }

  function saveLabels(labels: string[]): Promise<void> {
    return saveCardMeta({ labels }, "save labels");
  }

  function toggleCheckItem(idx: number): Promise<void> {
    return saveCardMeta({ checklist: toggleChecklistItem(meta!.checklist!, idx) }, "toggle checklist item");
  }

  function addCheckItem(desc: string): Promise<void> {
    return saveCardMeta({ checklist: addChecklistItem(meta!.checklist || [], desc) }, "add checklist item");
  }

  function editCheckItem(idx: number, desc: string): Promise<void> {
    return saveCardMeta({ checklist: editChecklistItem(meta!.checklist!, idx, desc) }, "edit checklist item");
  }

  function removeCheckItem(idx: number): Promise<void> {
    return saveCardMeta({ checklist: removeChecklistItem(meta!.checklist!, idx) }, "remove checklist item");
  }

  function reorderCheckItem(fromIdx: number, toIdx: number): Promise<void> {
    return saveCardMeta({ checklist: reorderChecklistItem(meta!.checklist!, fromIdx, toIdx) }, "reorder checklist");
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
      if (editingTitle) {
        editingTitle = false;
      } else if (editingUri) {
        editingUri = false;
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
    editingUri = false;
    confirmingDelete = false;

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
      addToast("Failed to load card content");
    });
  });

  // Opens the card's markdown file in the system default editor.
  function openExternal(): void {
    saveWithToast(OpenFileExternal($selectedCard!.filePath), "open file");
  }

  // Deletes the card from disk and removes it from the board store.
  async function deleteCard(): Promise<void> {
    const filePath = $selectedCard!.filePath;
    try {
      await DeleteCard(filePath);
      removeCardFromBoard(filePath);
      addToast("Card deleted", "success");
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
      <div class="modal-header card-editor">
        {#if editingTitle}
          <input class="edit-title-input" type="text" bind:value={editTitle} onblur={blurTitle} use:blurOnEnter use:autoFocus/>
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

          <!-- Primary URI -->
          <div class="uri-row">
            {#if editingUri}
              <Icon name="link" size={14} style="color: var(--color-text-muted); flex-shrink: 0" />
              <input class="uri-input" type="text" placeholder="https://..."
                bind:value={editUri}
                onblur={blurUri}
                onkeydown={e => e.key === 'Enter' && (e.target as HTMLInputElement).blur()}
                use:autoFocus
              />
              <button class="uri-action-btn remove" title="Remove URI" onclick={removeUri}>
                <Icon name="trash" size={12} />
              </button>
            {:else if meta.url}
              <Icon name="link" size={14} style="color: var(--color-text-muted); flex-shrink: 0" />
              <button class="uri-link" title={meta.url} onclick={openUri}>{meta.url}</button>
              <button class="uri-action-btn" title="Edit URI" onclick={startEditUri}>
                <Icon name="pencil" size={12} />
              </button>
              <button class="uri-action-btn remove" title="Remove URI" onclick={removeUri}>
                <Icon name="trash" size={12} />
              </button>
            {:else}
              <button class="uri-add-btn" onclick={startEditUri}>
                <Icon name="link" size={12} /> Add URI
              </button>
            {/if}
          </div>

          <!-- Description -->
          <div class="section">
            {#if editingBody}
              <textarea class="edit-body-textarea" bind:value={editBody} onblur={blurBody} placeholder="Card description (markdown)" use:autoFocus></textarea>
              <div class="edit-footer">
                <span>{charCount} chars, {wordCount} words</span>
                <button class="save-body-btn" title="Save" onmousedown={e => { e.preventDefault(); blurBody(); }}>
                  <Icon name="check" size={12} /> Save
                </button>
              </div>
            {:else if loading}
              <p class="loading-text">Loading...</p>
            {:else if bodyHtml.trim()}
              <div class="markdown-body clickable" role="button" tabindex="0"
                onclick={handleBodyClick} onkeydown={e => e.key === 'Enter' && startEditBody()}
              >{@html bodyHtml}</div>
            {:else}
              <button class="empty-desc clickable" title="Click to add description" onclick={startEditBody}>Enter description...</button>
            {/if}
          </div>

          <!-- Checklist -->
          {#if meta.checklist_title || (meta.checklist && meta.checklist.length > 0)}
            <div class="section">
              <ChecklistSection
                checklist={meta.checklist || []}
                title={meta.checklist_title || ""}
                ontoggle={toggleCheckItem}
                onadd={addCheckItem}
                onremove={removeCheckItem}
                onedit={editCheckItem}
                ontitlechange={saveChecklistTitle}
                onreorder={reorderCheckItem}
                ondelete={() => saveChecklist(null)}
              />
            </div>
          {/if}
        </div>

        <CardSidebar {meta}
          onsavecounter={saveCounter} onsavedates={saveDates}
          onsaveestimate={saveEstimate} onsaveicon={saveIcon}
          onsavechecklist={saveChecklist} onsavelabels={saveLabels}
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

  .card-title {
    all: unset;
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--color-text-primary);
    line-height: 1.3;
    word-break: break-word;
    overflow: hidden;
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

  .main-col {
    flex: 1;
    min-width: 0;
  }

  /* Primary URI */
  .uri-row {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 12px;
    min-height: 24px;
  }

  .uri-link {
    all: unset;
    font-size: 0.8rem;
    line-height: 1;
    color: var(--color-accent);
    cursor: pointer;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;

    &:hover {
      text-decoration: underline;
    }
  }

  .uri-input {
    flex: 1;
    min-width: 0;
    background: var(--color-bg-base);
    border: 1px solid var(--color-accent);
    color: var(--color-text-primary);
    font-size: 0.8rem;
    padding: 2px 6px;
    border-radius: 4px;
    outline: none;
    box-sizing: border-box;
  }

  .uri-action-btn {
    all: unset;
    display: inline-flex;
    align-items: center;
    color: var(--color-text-muted);
    cursor: pointer;
    flex-shrink: 0;
    padding: 2px;
    border-radius: 3px;

    &:hover {
      color: var(--color-text-primary);
    }

    &.remove:hover {
      color: var(--color-error);
    }
  }

  .uri-add-btn {
    all: unset;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 0.75rem;
    color: var(--color-text-muted);
    cursor: pointer;

    &:hover {
      color: var(--color-text-primary);
    }
  }

  /* Sections */
  .section {
    margin-bottom: 20px;
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
    color: var(--color-text-inverse);
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
