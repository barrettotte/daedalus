<script>
  import { afterUpdate } from "svelte";
  import { selectedCard } from "../stores/board.js";
  import { GetCardContent } from "../../wailsjs/go/main/App";
  import { marked } from "marked";

  let bodyHtml = "";
  let loading = false;
  let backdropEl;
  let prevCardId = null;

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

  function labelColor(label) {
    let hash = 0;
    for (let i = 0; i < label.length; i++) {
      hash = label.charCodeAt(i) + ((hash << 5) - hash);
    }
    const hue = ((hash % 360) + 360) % 360;
    return `hsl(${hue}, 55%, 45%)`;
  }

  function formatDate(d) {
    if (!d) {
      return "";
    }
    return new Date(d).toLocaleDateString(undefined, {
      year: "numeric", month: "short", day: "numeric"
    });
  }

  function formatDateTime(d) {
    if (!d) {
      return "";
    }
    return new Date(d).toLocaleString(undefined, {
      year: "numeric", month: "short", day: "numeric",
      hour: "2-digit", minute: "2-digit"
    });
  }

  function close() {
    selectedCard.set(null);
  }

  function handleKeydown(e) {
    if (e.key === "Escape") {
      close();
    }
  }

  function handleBackdropClick(e) {
    if (e.target === e.currentTarget) {
      close();
    }
  }

  $: if ($selectedCard) {
    loading = true;
    bodyHtml = "";
    GetCardContent($selectedCard.filePath).then(content => {
      // Strip leading h1 since title is already in the modal header
      const body = (content || "").replace(/^#\s+.*\n*/, "");
      bodyHtml = /** @type {string} */ (marked.parse(body));
      loading = false;
    }).catch(() => {
      bodyHtml = "<p><em>Could not load card content.</em></p>";
      loading = false;
    });
  }

  $: meta = $selectedCard ? $selectedCard.metadata : null;
  $: isOverdue = meta && meta.due ? new Date(meta.due) < new Date() : false;

  $: checkedCount = meta && meta.checklist
    ? meta.checklist.filter(i => i.done).length : 0;

  $: counterPct = meta && meta.counter && meta.counter.max > 0
    ? (meta.counter.current / meta.counter.max) * 100 : 0;

  $: checkPct = meta && meta.checklist && meta.checklist.length > 0
    ? (checkedCount / meta.checklist.length) * 100 : 0;
</script>

<svelte:window on:keydown={handleKeydown} />

{#if $selectedCard && meta}
  <div class="backdrop" bind:this={backdropEl} on:click={handleBackdropClick} on:keydown={handleKeydown}>
    <div class="modal">
      <button class="close-btn" on:click={close}>&times;</button>

      <div class="modal-header">
        <h2 class="card-title">{meta.title}</h2>
      </div>

      <div class="modal-body">
        <div class="main-col">

          <!-- Description -->
          <div class="section">
            <div class="section-header">
              <svg class="section-icon" viewBox="0 0 24 24">
                <line x1="4" y1="6" x2="20" y2="6" stroke="currentColor" stroke-width="2"/>
                <line x1="4" y1="12" x2="16" y2="12" stroke="currentColor" stroke-width="2"/>
                <line x1="4" y1="18" x2="12" y2="18" stroke="currentColor" stroke-width="2"/>
              </svg>
              <h3 class="section-title">Description</h3>
            </div>
            {#if loading}
              <p class="loading-text">Loading...</p>
            {:else if bodyHtml.trim()}
              <div class="markdown-body">{@html bodyHtml}</div>
            {:else}
              <p class="empty-desc">No description provided.</p>
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
                {#each meta.checklist as item}
                  <li class:done={item.done}>
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
                    <span class="check-text">{item.desc}</span>
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
            <div class="sidebar-value">{$selectedCard.listName}</div>
          </div>

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

  .close-btn {
    position: absolute;
    top: 8px;
    right: 8px;
    background: rgba(255, 255, 255, 0.08);
    border: none;
    color: #9fadbc;
    font-size: 1.4rem;
    cursor: pointer;
    line-height: 1;
    width: 32px;
    height: 32px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1;
  }
  .close-btn:hover {
    background: rgba(255, 255, 255, 0.15);
    color: #fff;
  }

  /* Header */
  .modal-header {
    padding: 20px 52px 20px 20px;
  }
  .card-title {
    margin: 0;
    font-size: 1.25rem;
    font-weight: 600;
    color: #c7d1db;
    line-height: 1.3;
    word-break: break-word;
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
    overflow-x: auto;
    white-space: nowrap;
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
