<script lang="ts">
  import { selectedCard, draftListKey, draftPosition, updateCardInBoard, addCardToBoard, removeCardFromBoard, boardConfig, boardData, sortedListKeys, focusedCard, openInEditMode, addToast } from "../stores/board";
  import { GetCardContent, SaveCard, OpenFileExternal, CreateCard, DeleteCard } from "../../wailsjs/go/main/App";
  import { marked } from "marked";
  import { labelColor, formatDate, formatDateTime, formatListName, autoFocus } from "../lib/utils";
  import type { daedalus } from "../../wailsjs/go/models";

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

  // Delete confirmation state
  let confirmingDelete = $state(false);

  // Guards against stale async responses when rapidly switching cards.
  let loadGeneration = 0;

  // Draft mode state for creating new cards before they exist on disk
  let draftTitle = $state("");
  let draftBody = $state("");
  let saving = $state(false);

  // Scroll reset when selected card changes
  $effect(() => {
    const curId = $selectedCard ? $selectedCard.metadata.id : null;
    if (curId && curId !== prevCardId && backdropEl) {
      backdropEl.scrollTop = 0;
      prevCardId = curId;
    }
    if (!curId) {
      prevCardId = null;
    }
  });

  // Closes whichever modal is active
  function close(): void {
    if ($draftListKey) {
      draftTitle = "";
      draftBody = "";
      saving = false;
      draftListKey.set(null);
    } else {
      editingTitle = false;
      editingBody = false;
      selectedCard.set(null);
    }
  }

  // Validates and saves the draft card to disk, then adds it to the board store.
  async function saveDraft(): Promise<void> {
    if (!draftTitle.trim()) {
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
    } catch (e) {
      addToast(`Failed to create card: ${e}`);
    }
    saving = false;
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

    const keys = sortedListKeys($boardData);
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

  // Handles keyboard shortcuts: Ctrl/Cmd+Enter saves draft, Escape closes/cancels, arrows navigate.
  function handleKeydown(e: KeyboardEvent): void {
    // Only handle keys when the modal is actually open.
    if (!$selectedCard) {
      return;
    }

    if ($draftListKey) {
      if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
        e.preventDefault();
        saveDraft();
      } else if (e.key === "Escape") {
        close();
      }
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
      } else if (editingBody) {
        editingBody = false;
      } else if (confirmingDelete) {
        confirmingDelete = false;
      } else {
        close();
      }
    }
  }

  // Closes the modal when clicking the backdrop area outside the modal content.
  function handleBackdropClick(e: MouseEvent): void {
    if (e.target === e.currentTarget) {
      close();
    }
  }

  // Resets draft fields only on transition from null → non-null (new draft started).
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

  // Loads card content when a card is selected.
  $effect(() => {
    if ($selectedCard) {
      const gen = ++loadGeneration;
      loading = true;
      bodyHtml = "";
      rawBody = "";
      editingTitle = false;
      editingBody = false;
      confirmingDelete = false;

      const shouldEdit = $openInEditMode;
      if (shouldEdit) {
        openInEditMode.set(false);
      }

      GetCardContent($selectedCard.filePath).then(content => {
        if (gen !== loadGeneration) { return; }
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
        if (gen !== loadGeneration) { return; }
        bodyHtml = "<p><em>Could not load card content.</em></p>";
        rawBody = "";
        loading = false;
      });
    }
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
  let isOverdue = $derived(meta && meta.due ? new Date(meta.due) < new Date() : false);

  let checkedCount = $derived(meta && meta.checklist
    ? meta.checklist.filter(i => i.done).length : 0);

  let counterPct = $derived(meta && meta.counter && meta.counter.max > 0
    ? (meta.counter.current / meta.counter.max) * 100 : 0);

  let checkPct = $derived(meta && meta.checklist && meta.checklist.length > 0
    ? (checkedCount / meta.checklist.length) * 100 : 0);

  // Returns true if the string is a URL.
  function isUrl(str: string): boolean {
    return /^https?:\/\/\S+$/.test(str);
  }

  // Derives the list display name from the config title or formatted directory name.
  let listDisplayName = $derived.by(() => {
    if (!$selectedCard) {
      return "";
    }

    const parts = $selectedCard.filePath.split("/");
    const dirName = parts[parts.length - 2] || "";
    const cfg = $boardConfig[dirName];

    if (cfg && cfg.title) {
      return cfg.title;
    }
    return formatListName(dirName);
  });

  // Derives the card's 1-based position and list size from boardData.
  let cardPosition = $derived.by(() => {
    if (!$selectedCard) {
      return "";
    }

    const parts = $selectedCard.filePath.split("/");
    const dirName = parts[parts.length - 2] || "";
    const cards = $boardData[dirName];
    if (!cards) {
      return "";
    }

    const idx = cards.findIndex(c => c.filePath === $selectedCard!.filePath);
    if (idx === -1) {
      return "";
    }
    return `${idx + 1} / ${cards.length}`;
  });
</script>

<svelte:window onkeydown={handleKeydown} />

{#if $draftListKey}
  <div class="backdrop" role="presentation" onclick={handleBackdropClick} onkeydown={handleKeydown}>
    <div class="modal" role="dialog">
      <div class="modal-header">
        <input
          class="edit-title-input"
          type="text"
          bind:value={draftTitle}
          placeholder="Card title"
          onkeydown={e => e.key === 'Enter' && saveDraft()}
          use:autoFocus
        />
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
            <textarea
              class="edit-body-textarea"
              bind:value={draftBody}
              placeholder="Card description (markdown)"
            ></textarea>
          </div>
          <div class="draft-actions">
            <div class="position-toggle">
              <button class="pos-btn" class:active={$draftPosition === 'top'} onclick={() => draftPosition.set('top')}>Top</button>
              <button class="pos-btn" class:active={$draftPosition === 'bottom'} onclick={() => draftPosition.set('bottom')}>Bottom</button>
            </div>
            <button class="save-btn" onclick={saveDraft} disabled={saving || !draftTitle.trim()}>
              {saving ? "Saving..." : "Save"}
            </button>
            <button class="cancel-btn" onclick={close}>Cancel</button>
          </div>
        </div>
      </div>
    </div>
  </div>
{:else if $selectedCard && meta}
  <div class="backdrop" bind:this={backdropEl} role="presentation" onclick={handleBackdropClick} onkeydown={handleKeydown}>
    <div class="modal" role="dialog">
      <div class="modal-header">
        {#if editingTitle}
          <input
            class="edit-title-input"
            type="text"
            bind:value={editTitle}
            onblur={blurTitle}
            onkeydown={e => e.key === 'Enter' && (e.target as HTMLInputElement).blur()}
            use:autoFocus
          />
        {:else}
          <button class="card-title clickable" onclick={startEditTitle}>{meta.title}</button>
        {/if}
        <div class="header-btns">
          {#if !loading}
            <button class="header-btn" onclick={startEditAll} title="Edit">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path d="M17 3a2.83 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                <path d="m15 5 4 4" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </button>
          {/if}
          <button class="header-btn" onclick={openExternal} title="Open in editor">
            <svg viewBox="0 0 24 24" width="16" height="16">
              <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              <polyline points="15 3 21 3 21 9" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              <line x1="10" y1="14" x2="21" y2="3" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
          <button class="header-btn delete-icon" onclick={() => confirmingDelete = true} title="Delete card">
            <svg viewBox="0 0 24 24" width="16" height="16">
              <polyline points="3 6 5 6 21 6" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
          <button class="header-btn" onclick={close} title="Close">
            <svg viewBox="0 0 24 24" width="16" height="16">
              <line x1="18" y1="6" x2="6" y2="18" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
              <line x1="6" y1="6" x2="18" y2="18" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            </svg>
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
              <textarea class="edit-body-textarea" bind:value={editBody} onblur={blurBody}
                placeholder="Card description (markdown)" use:autoFocus
              ></textarea>
            {:else if loading}
              <p class="loading-text">Loading...</p>
            {:else if bodyHtml.trim()}
              <div class="markdown-body clickable" role="button" tabindex="0" onclick={startEditBody} onkeydown={e => e.key === 'Enter' && startEditBody()}>{@html bodyHtml}</div>
            {:else}
              <button class="empty-desc clickable" onclick={startEditBody}>Enter description...</button>
            {/if}
          </div>

          <!-- Checklist -->
          {#if meta.checklist && meta.checklist.length > 0}
            <div class="section">
              <div class="section-header">
                <svg class="section-icon" viewBox="0 0 24 24">
                  <polyline points="9 11 12 14 22 4" fill="none" stroke="currentColor" stroke-width="2"/>
                  <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" fill="none" stroke="currentColor" stroke-width="2"/>
                </svg>
                <h3 class="section-title">Checklist</h3>
                <div class="checklist-bar">
                  <div class="progress-fill" class:complete={checkPct === 100} style="width: {checkPct}%"></div>
                </div>
                <span class="checklist-count">{checkedCount}/{meta.checklist.length}</span>
              </div>
              <ul class="checklist">
                {#each meta.checklist as item, idx}
                  <li class:done={item.done}>
                    <button class="checkbox-btn" onclick={() => toggleCheckItem(idx)}>
                      <span class="checkbox" class:checked={item.done}>
                        {#if item.done}
                          <svg viewBox="0 0 16 16">
                            <rect x="1" y="1" width="14" height="14" rx="2" fill="currentColor"/>
                            <polyline points="4 8 7 11 12 5" fill="none" stroke="#22252b" stroke-width="2"/>
                          </svg>
                        {:else}
                          <svg viewBox="0 0 16 16">
                            <rect x="1" y="1" width="14" height="14" rx="2" fill="none" stroke="currentColor" stroke-width="1.5"/>
                          </svg>
                        {/if}
                      </span>
                    </button>
                    <span class="check-text">{#if isUrl(item.desc)}<a href={item.desc} target="_blank" rel="noopener noreferrer" onclick={(e: MouseEvent) => e.stopPropagation()}>{item.desc}</a>{:else}{item.desc}{/if}</span>
                  </li>
                {/each}
              </ul>
            </div>
          {/if}
        </div>

        <!-- Sidebar -->
        <div class="sidebar">
          <div class="sidebar-section">
            <h4 class="sidebar-title">Card</h4>
            <div class="sidebar-value">#{meta.id}</div>
          </div>

          <div class="sidebar-section">
            <h4 class="sidebar-title">List</h4>
            <div class="sidebar-value">{listDisplayName}</div>
          </div>

          {#if cardPosition}
            <div class="sidebar-section">
              <h4 class="sidebar-title">Position</h4>
              <div class="sidebar-value">{cardPosition}</div>
            </div>
          {/if}

          {#if meta.labels && meta.labels.length > 0}
            <div class="sidebar-section">
              <h4 class="sidebar-title">Labels</h4>
              <div class="sidebar-labels">
                {#each meta.labels as label}
                  <span class="label" style="background: {labelColor(label)}">{label}</span>
                {/each}
              </div>
            </div>
          {/if}

          {#if meta.due}
            <div class="sidebar-section">
              <h4 class="sidebar-title">Due Date</h4>
              <div class="sidebar-badge" class:overdue={isOverdue} class:on-time={!isOverdue}>
                <svg class="badge-icon" viewBox="0 0 24 24">
                  <circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="2"/>
                  <polyline points="12 6 12 12 16 14" fill="none" stroke="currentColor" stroke-width="2"/>
                </svg>
                {formatDate(meta.due)}
              </div>
            </div>
          {/if}

          {#if meta.range}
            <div class="sidebar-section">
              <h4 class="sidebar-title">Date Range</h4>
              <div class="sidebar-value">{formatDate(meta.range.start)} &ndash; {formatDate(meta.range.end)}</div>
            </div>
          {/if}

          {#if meta.counter}
            <div class="sidebar-section">
              <h4 class="sidebar-title">{meta.counter.label || "Counter"}</h4>
              <div class="counter-value">{meta.counter.current} / {meta.counter.max}</div>
              <div class="progress-bar sidebar-progress">
                <div class="progress-fill" class:complete={counterPct === 100} style="width: {counterPct}%"></div>
              </div>
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
      </div>
      {/if}
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
    color: #b6c2d1;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, sans-serif;
    margin-bottom: 48px;
    text-align: left;
  }

  /* Header */
  .modal-header {
    display: flex;
    align-items: flex-start;
    gap: 12px;
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
      color: #fff;
    }
  }

  .clickable {
    cursor: pointer;

    &:hover {
      outline: 1px solid rgba(255, 255, 255, 0.1);
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

  /* Body layout */
  .modal-body {
    display: flex;
    gap: 16px;
    padding: 0 20px 20px 20px;
  }

  .main-col {
    flex: 1;
    min-width: 0;
  }

  .sidebar {
    flex: 0 0 168px;
  }

  /* Sections */
  .section {
    margin-bottom: 20px;
  }

  .section-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
  }

  .section-icon {
    width: 20px;
    height: 20px;
    color: var(--color-text-secondary);
    flex-shrink: 0;
  }

  .section-title {
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--color-text-primary);
    margin: 0;
  }

  .checklist-bar {
    flex: 1;
    height: 6px;
    background: var(--color-border);
    border-radius: 3px;
    overflow: hidden;
    margin: 0 8px;
  }

  .checklist-count {
    font-size: 0.75rem;
    color: var(--color-text-tertiary);
    flex-shrink: 0;
  }

  /* Edit textarea */
  .edit-body-textarea {
    width: 100%;
    min-height: 200px;
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    color: #b6c2d1;
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

  /* Labels */
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

  /* Progress bars */
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

  /* Checklist */
  .checklist {
    list-style: none;
    padding: 0;
    margin: 0;
    max-height: 400px;
    overflow-y: auto;

    li {
      padding: 6px 8px;
      font-size: 0.85rem;
      display: flex;
      gap: 8px;
      align-items: flex-start;
      border-radius: 4px;

      &:hover {
        background: var(--overlay-hover-faint);
      }

      &.done .check-text {
        text-decoration: line-through;
        color: var(--color-text-muted);
      }
    }
  }

  .checkbox-btn {
    all: unset;
    cursor: pointer;
    display: flex;
    align-items: center;
    flex-shrink: 0;
  }

  .checkbox {
    flex-shrink: 0;
    width: 16px;
    height: 16px;
    margin-top: 1px;
    color: var(--color-text-secondary);

    &.checked {
      color: var(--color-accent);
    }

    svg {
      width: 16px;
      height: 16px;
    }
  }

  .check-text {
    line-height: 1.3;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;

    a {
      color: var(--color-accent);
      text-decoration: none;
      line-height: inherit;
      display: inline;

      &:hover {
        text-decoration: underline;
      }
    }
  }

  /* Sidebar */
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
    color: #b6c2d1;
  }

  .sidebar-badge {
    font-size: 0.8rem;
    font-weight: 600;
    padding: 4px 8px;
    border-radius: 3px;
    display: inline-flex;
    align-items: center;
    gap: 6px;

    &.on-time {
      background: var(--overlay-success-strong);
      color: var(--color-success);
    }

    &.overdue {
      background: var(--overlay-error-strong);
      color: var(--color-error);
    }
  }

  .badge-icon {
    width: 14px;
    height: 14px;
  }

  .counter-value {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--color-text-primary);
    margin-bottom: 6px;
  }

  .sidebar-progress {
    margin-bottom: 0;
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
    color: #b6c2d1;
    background: var(--overlay-subtle);
    border-radius: 6px;
    padding: 12px 16px;
    overflow-x: auto;

    :global(h1),
    :global(h2),
    :global(h3) {
      color: var(--color-text-primary);
      margin-top: 16px;
      margin-bottom: 8px;
    }

    :global(h1) { font-size: 1.2rem; }
    :global(h2) { font-size: 1.05rem; }
    :global(h3) { font-size: 0.95rem; }

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

    :global(ul),
    :global(ol) {
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

    :global(th),
    :global(td) {
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

  /* Draft mode actions */
  .draft-actions {
    display: flex;
    gap: 8px;
    align-items: center;
    justify-content: flex-end;
  }

  .position-toggle {
    display: flex;
    border-radius: 4px;
    overflow: hidden;
    border: 1px solid var(--color-border);
    margin-right: auto;
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
      color: #fff;
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
