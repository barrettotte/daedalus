<script>
  import { afterUpdate } from "svelte";
  import { selectedCard, updateCardInBoard, boardConfig, boardData, addToast } from "../stores/board.js";
  import { GetCardContent, SaveCard, OpenFileExternal } from "../../wailsjs/go/main/App";
  import { marked } from "marked";
  import { labelColor, formatDate, formatDateTime, formatListName, autoFocus } from "../lib/utils.js";

  let bodyHtml = "";
  let rawBody = "";
  let loading = false;
  let backdropEl;
  let prevCardId = null;

  // Edit state — title and body are independently inline-editable
  let editingTitle = false;
  let editingBody = false;
  let editTitle = "";
  let editBody = "";

  afterUpdate(() => {
    const curId = $selectedCard ? $selectedCard.metadata.id : null;
    if (curId && curId !== prevCardId && backdropEl) {
      backdropEl.scrollTop = 0;
      prevCardId = curId;
    }
    if (!curId) {
      prevCardId = null;
    }
  });

  // Closes the detail modal by clearing the selected card and resetting edit state.
  function close() {
    editingTitle = false;
    editingBody = false;
    selectedCard.set(null);
  }

  // Opens both title and body for inline editing.
  function startEditAll() {
    editTitle = meta.title;
    editBody = rawBody;
    editingTitle = true;
    editingBody = true;
  }

  // Opens the title for inline editing.
  function startEditTitle() {
    editTitle = meta.title;
    editingTitle = true;
  }

  // Saves the title on blur, or discards if unchanged. Defaults blank titles to the card number.
  async function blurTitle() {
    editingTitle = false;
    if (!editTitle || !editTitle.trim()) {
      editTitle = String(meta.id);
    }
    if (editTitle === meta.title) {
      return;
    }

    const fullBody = `# ${editTitle}\n\n${rawBody}`;
    const updatedMeta = { ...meta, title: editTitle };
    try {
      const result = await SaveCard($selectedCard.filePath, updatedMeta, fullBody);
      updateCardInBoard(result);
      selectedCard.set(result);
    } catch (e) {
      addToast(`Failed to save title: ${e}`);
    }
  }

  // Opens the description for inline editing.
  function startEditBody() {
    editBody = rawBody;
    editingBody = true;
  }

  // Saves the body on blur, or discards if unchanged.
  async function blurBody() {
    editingBody = false;
    if (editBody === rawBody) {
      return;
    }
    const fullBody = `# ${meta.title}\n\n${editBody}`;
  
    try {
      const result = await SaveCard($selectedCard.filePath, { ...meta }, fullBody);
      updateCardInBoard(result);
      selectedCard.set(result);
      rawBody = editBody;
      bodyHtml = /** @type {string} */ (marked.parse(editBody));
    } catch (e) {
      addToast(`Failed to save body: ${e}`);
    }
  }

  // Toggles a checklist item's done state and saves immediately.
  async function toggleCheckItem(idx) {
    const updatedChecklist = meta.checklist.map((item, i) => {
      if (i === idx) {
        return { ...item, done: !item.done };
      }
      return { ...item };
    });

    const updatedMeta = { ...meta, checklist: updatedChecklist };
    const fullBody = `# ${meta.title}\n\n${rawBody}`;

    try {
      const result = await SaveCard($selectedCard.filePath, updatedMeta, fullBody);
      updateCardInBoard(result);
      selectedCard.set(result);
    } catch (e) {
      addToast(`Failed to toggle checklist item: ${e}`);
    }
  }

  // Closes the modal when Escape is pressed, cancelling any active inline edit first.
  function handleKeydown(e) {
    if (e.key === "Escape") {
      if (editingTitle) {
        editingTitle = false;
      } else if (editingBody) {
        editingBody = false;
      } else {
        close();
      }
    }
  }

  // Closes the modal when clicking the backdrop area outside the modal content.
  function handleBackdropClick(e) {
    if (e.target === e.currentTarget) {
      close();
    }
  }

  $: if ($selectedCard) {
    loading = true;
    bodyHtml = "";
    rawBody = "";
    editingTitle = false;
    editingBody = false;

    GetCardContent($selectedCard.filePath).then(content => {
      // Strip leading h1 since title is already in the modal header
      const body = (content || "").replace(/^#\s+.*\n*/, "");
      rawBody = body;
      bodyHtml = /** @type {string} */ (marked.parse(body));
      loading = false;
    }).catch(() => {
      bodyHtml = "<p><em>Could not load card content.</em></p>";
      rawBody = "";
      loading = false;
    });
  }

  // Opens the card's markdown file in the system default editor.
  function openExternal() {
    OpenFileExternal($selectedCard.filePath).catch(e => addToast(`Failed to open file: ${e}`));
  }

  $: meta = $selectedCard ? $selectedCard.metadata : null;
  $: isOverdue = meta && meta.due ? new Date(meta.due) < new Date() : false;

  $: checkedCount = meta && meta.checklist
    ? meta.checklist.filter(i => i.done).length : 0;

  $: counterPct = meta && meta.counter && meta.counter.max > 0
    ? (meta.counter.current / meta.counter.max) * 100 : 0;

  $: checkPct = meta && meta.checklist && meta.checklist.length > 0
    ? (checkedCount / meta.checklist.length) * 100 : 0;

  // Returns true if the string is a URL.
  function isUrl(str) {
    return /^https?:\/\/\S+$/.test(str);
  }

  // Derives the list display name from the config title or formatted directory name.
  $: listDisplayName = (() => {
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
  })();

  // Derives the card's 1-based position and list size from boardData.
  $: cardPosition = (() => {
    if (!$selectedCard) {
      return "";
    }

    const parts = $selectedCard.filePath.split("/");
    const dirName = parts[parts.length - 2] || "";
    const cards = $boardData[dirName];
    if (!cards) {
      return "";
    }

    const idx = cards.findIndex(c => c.filePath === $selectedCard.filePath);
    if (idx === -1) {
      return "";
    }
    return `${idx + 1} / ${cards.length}`;
  })();
</script>

<svelte:window on:keydown={handleKeydown} />

{#if $selectedCard && meta}
  <div class="backdrop" bind:this={backdropEl} on:click={handleBackdropClick} on:keydown={handleKeydown}>
    <div class="modal">
      <div class="modal-header">
        {#if editingTitle}
          <input
            class="edit-title-input"
            type="text"
            bind:value={editTitle}
            on:blur={blurTitle}
            on:keydown={e => e.key === 'Enter' && e.target.blur()}
            use:autoFocus
          />
        {:else}
          <h2 class="card-title clickable" on:click={startEditTitle} on:keydown={e => e.key === 'Enter' && startEditTitle()}>{meta.title}</h2>
        {/if}
        <div class="header-btns">
          {#if !loading}
            <button class="header-btn" on:click={startEditAll} title="Edit">
              <svg viewBox="0 0 24 24" width="16" height="16">
                <path d="M17 3a2.83 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                <path d="m15 5 4 4" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </button>
          {/if}
          <button class="header-btn" on:click={openExternal} title="Open in editor">
            <svg viewBox="0 0 24 24" width="16" height="16">
              <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              <polyline points="15 3 21 3 21 9" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              <line x1="10" y1="14" x2="21" y2="3" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
          <button class="header-btn" on:click={close} title="Close">
            <svg viewBox="0 0 24 24" width="16" height="16">
              <line x1="18" y1="6" x2="6" y2="18" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
              <line x1="6" y1="6" x2="18" y2="18" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            </svg>
          </button>
        </div>
      </div>

      <div class="modal-body">
        <div class="main-col">

          <!-- Description -->
          <div class="section">
            {#if editingBody}
              <textarea class="edit-body-textarea" bind:value={editBody} on:blur={blurBody}
                placeholder="Card description (markdown)" use:autoFocus
              ></textarea>
            {:else if loading}
              <p class="loading-text">Loading...</p>
            {:else if bodyHtml.trim()}
              <div class="markdown-body clickable" on:click={startEditBody} on:keydown={e => e.key === 'Enter' && startEditBody()}>{@html bodyHtml}</div>
            {:else}
              <p class="empty-desc clickable" on:click={startEditBody} on:keydown={e => e.key === 'Enter' && startEditBody()}>Enter description...</p>
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
                    <button class="checkbox-btn" on:click={() => toggleCheckItem(idx)}>
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
                    <span class="check-text">{#if isUrl(item.desc)}<a href={item.desc} target="_blank" rel="noopener noreferrer" on:click|stopPropagation>{item.desc}</a>{:else}{item.desc}{/if}</span>
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
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: flex-start;
    justify-content: center;
    z-index: 1000;
    padding-top: 48px;
    overflow-y: auto;
  }

  .modal {
    background: #282c34;
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
    margin: 0;
    font-size: 1.25rem;
    font-weight: 600;
    color: #c7d1db;
    line-height: 1.3;
    word-break: break-word;
    flex: 1;
    min-width: 0;
    padding-top: 4px;
  }
  .header-btns {
    display: flex;
    gap: 4px;
    flex-shrink: 0;
  }
  .header-btn {
    background: rgba(255, 255, 255, 0.08);
    border: none;
    color: #9fadbc;
    cursor: pointer;
    width: 32px;
    height: 32px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .header-btn:hover {
    background: rgba(255, 255, 255, 0.15);
    color: #fff;
  }
  .clickable {
    cursor: pointer;
  }
  .clickable:hover {
    outline: 1px solid rgba(255, 255, 255, 0.1);
    outline-offset: 4px;
    border-radius: 4px;
  }
  .edit-title-input {
    flex: 1;
    min-width: 0;
    background: #1e2128;
    border: 1px solid #579dff;
    color: #c7d1db;
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
    color: #9fadbc;
    flex-shrink: 0;
  }
  .section-title {
    font-size: 0.9rem;
    font-weight: 600;
    color: #c7d1db;
    margin: 0;
  }
  .checklist-bar {
    flex: 1;
    height: 6px;
    background: #3b4048;
    border-radius: 3px;
    overflow: hidden;
    margin: 0 8px;
  }
  .checklist-count {
    font-size: 0.75rem;
    color: #8c9bab;
    flex-shrink: 0;
  }

  /* Edit textarea */
  .edit-body-textarea {
    width: 100%;
    min-height: 200px;
    background: #1e2128;
    border: 1px solid #3b4048;
    color: #b6c2d1;
    font-size: 0.9rem;
    font-family: monospace;
    padding: 12px;
    border-radius: 6px;
    outline: none;
    resize: vertical;
    box-sizing: border-box;
    line-height: 1.5;
  }
  .edit-body-textarea:focus {
    border-color: #579dff;
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
    background: #3b4048;
    border-radius: 3px;
    overflow: hidden;
    margin-bottom: 8px;
    max-width: 100%;
    box-sizing: border-box;
  }
  .progress-fill {
    height: 100%;
    background: #579dff;
    border-radius: 4px;
    transition: width 0.3s;
  }
  .progress-fill.complete {
    background: #4bce97;
  }

  /* Checklist */
  .checklist {
    list-style: none;
    padding: 0;
    margin: 0;
    max-height: 400px;
    overflow-y: auto;
  }
  .checklist li {
    padding: 6px 8px;
    font-size: 0.85rem;
    display: flex;
    gap: 8px;
    align-items: flex-start;
    border-radius: 4px;
  }
  .checklist li:hover {
    background: rgba(255, 255, 255, 0.04);
  }
  .checklist li.done .check-text {
    text-decoration: line-through;
    color: #6b7a8d;
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
    color: #9fadbc;
  }
  .checkbox.checked {
    color: #579dff;
  }
  .checkbox svg {
    width: 16px;
    height: 16px;
  }
  .check-text {
    line-height: 1.3;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .check-text a {
    color: #579dff;
    text-decoration: none;
    line-height: inherit;
    display: inline;
  }
  .check-text a:hover {
    text-decoration: underline;
  }

  /* Sidebar */
  .sidebar-section {
    background: rgba(255, 255, 255, 0.06);
    border-radius: 6px;
    padding: 10px 12px;
    margin-bottom: 8px;
  }
  .sidebar-title {
    font-size: 0.7rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: #8c9bab;
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
  }
  .sidebar-badge.on-time {
    background: rgba(75, 206, 151, 0.2);
    color: #4bce97;
  }
  .sidebar-badge.overdue {
    background: rgba(247, 68, 68, 0.2);
    color: #f87168;
  }
  .badge-icon {
    width: 14px;
    height: 14px;
  }
  .counter-value {
    font-size: 0.85rem;
    font-weight: 600;
    color: #c7d1db;
    margin-bottom: 6px;
  }
  .sidebar-progress {
    margin-bottom: 0;
  }

  /* Markdown body */
  .loading-text {
    color: #6b7a8d;
    font-size: 0.85rem;
  }
  .empty-desc {
    color: #6b7a8d;
    font-size: 0.85rem;
    font-style: italic;
    margin: 0;
  }
  .markdown-body {
    line-height: 1.6;
    font-size: 0.9rem;
    color: #b6c2d1;
    background: rgba(255, 255, 255, 0.03);
    border-radius: 6px;
    padding: 12px 16px;
    overflow-x: auto;
  }
  .markdown-body :global(h1),
  .markdown-body :global(h2),
  .markdown-body :global(h3) {
    color: #c7d1db;
    margin-top: 16px;
    margin-bottom: 8px;
  }
  .markdown-body :global(h1) { font-size: 1.2rem; }
  .markdown-body :global(h2) { font-size: 1.05rem; }
  .markdown-body :global(h3) { font-size: 0.95rem; }
  .markdown-body :global(a) {
    color: #579dff;
  }
  .markdown-body :global(a:hover) {
    text-decoration: underline;
  }
  .markdown-body :global(code) {
    background: #1e2128;
    padding: 2px 6px;
    border-radius: 3px;
    font-size: 0.85em;
  }
  .markdown-body :global(pre) {
    background: #1e2128;
    padding: 12px;
    border-radius: 6px;
    overflow-x: auto;
  }
  .markdown-body :global(pre code) {
    padding: 0;
    background: none;
  }
  .markdown-body :global(blockquote) {
    border-left: 3px solid #579dff;
    margin: 8px 0;
    padding: 4px 12px;
    color: #8c9bab;
  }
  .markdown-body :global(ul),
  .markdown-body :global(ol) {
    padding-left: 20px;
  }
  .markdown-body :global(li) {
    margin-bottom: 2px;
  }
  .markdown-body :global(img) {
    max-width: 100%;
    border-radius: 4px;
  }
  .markdown-body :global(table) {
    border-collapse: collapse;
    width: 100%;
    margin: 8px 0;
  }
  .markdown-body :global(th),
  .markdown-body :global(td) {
    border: 1px solid #3b4048;
    padding: 6px 10px;
    text-align: left;
  }
  .markdown-body :global(th) {
    background: #1e2128;
  }
  .markdown-body :global(hr) {
    border: none;
    border-top: 1px solid #3b4048;
    margin: 16px 0;
  }
</style>
