<script>
  import { onMount, onDestroy } from "svelte";
  import { LoadBoard, SaveListConfig, SaveLabelsExpanded, SaveCollapsedLists, MoveCard } from "../wailsjs/go/main/App";
  import { boardData, boardConfig, sortedListKeys, isLoaded, selectedCard, draftListKey, draftPosition, showMetrics, labelsExpanded, dragState, dropTarget, moveCardInBoard, computeListOrder, addToast } from "./stores/board.js";
  import { formatListName, autoFocus, labelColor } from "./lib/utils.js";
  import VirtualList from "./components/VirtualList.svelte";
  import Card from "./components/Card.svelte";
  import CardDetail from "./components/CardDetail.svelte";
  import Metrics from "./components/Metrics.svelte";
  import Toast from "./components/Toast.svelte";

  let error = "";
  let editingTitle = null;
  let editingLimit = null;
  let editTitleValue = "";
  let editLimitValue = 0;
  let collapsedLists = new Set();
  let boardContainerEl;
  let autoScrollRaf = null;
  let dragX = 0;
  let dragY = 0;
  let activeIndicators = new Set();

  // Clears drop indicator classes from tracked elements (avoids querySelectorAll on every dragover).
  function clearDropIndicators() {
    for (const el of activeIndicators) {
      el.classList.remove('drop-above', 'drop-below', 'drop-top');
    }
    activeIndicators.clear();
  }

  // Adds a drop indicator class to an element and tracks it for efficient cleanup.
  function addIndicator(el, cls) {
    el.classList.add(cls);
    activeIndicators.add(el);
  }

  // Allows the element to be a valid drop target — WebKitGTK requires dragenter prevention in addition to dragover.
  function handleDragEnter(e) {
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
  }

  // Handles dragover on a list body — positions the drop indicator and triggers auto-scroll.
  // preventDefault must be called unconditionally — WebKitGTK requires it on every dragover to allow drops.
  function handleDragOver(e, listKey) {
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    dragX = e.clientX;
    dragY = e.clientY;
    if (!$dragState) {
      return;
    }

    clearDropIndicators();

    // Find the closest item-slot under the cursor
    const slot = e.target.closest('[data-card-id]');
    if (slot) {
      const rect = slot.getBoundingClientRect();
      const midY = rect.top + rect.height / 2;

      if (e.clientY < midY) {
        // Top half — insert before this card
        addIndicator(slot, 'drop-above');
        dropTarget.set({ listKey, cardId: Number(slot.dataset.cardId), position: "above" });
      } else {
        // Bottom half — insert after this card; show indicator on next card for consistent position
        const next = slot.nextElementSibling;

        if (next && next.hasAttribute('data-card-id')) {
          addIndicator(next, 'drop-above');
          dropTarget.set({ listKey, cardId: Number(next.dataset.cardId), position: "above" });
        } else {
          addIndicator(slot, 'drop-below');
          dropTarget.set({ listKey, cardId: Number(slot.dataset.cardId), position: "below" });
        }
      }

    } else {
      // No card under cursor — determine if near top or bottom of the list
      const listBody = e.currentTarget;
      const rect = listBody.getBoundingClientRect();
      const cards = $boardData[listKey] || [];

      if (cards.length > 0 && e.clientY < rect.top + rect.height / 3) {
        dropTarget.set({ listKey, cardId: cards[0].metadata.id, position: "above" });
        addIndicator(listBody, 'drop-top');
      } else {
        dropTarget.set({ listKey, cardId: null, position: "below" });
      }
    }

    // Auto-scroll: vertical (inside the list's scroll container) and horizontal (board)
    handleAutoScroll(e);
  }

  // Handles dragleave — clears visual indicator only when cursor truly exits the list body.
  // Never clears dropTarget here; that's handled by handleDrop and the dragState reactive cleanup.
  function handleDragLeave(e) {
    const related = e.relatedTarget;
    // relatedTarget is null in WebKitGTK — skip cleanup to avoid false positives
    if (related && !e.currentTarget.contains(related)) {
      clearDropIndicators();
    }
  }

  // Handles drop on a list body — computes target index and list_order, then calls MoveCard API.
  async function handleDrop(e, listKey) {
    e.preventDefault();
    clearDropIndicators();

    const drag = $dragState;
    const drop = $dropTarget;
    dragState.set(null);
    dropTarget.set(null);

    if (!drag || !drop) {
      return;
    }

    const cards = $boardData[listKey] || [];
    let targetIndex;

    if (drop.cardId == null) {
      // Drop on empty list or empty area
      targetIndex = cards.length;
    } else {
      const cardIdx = cards.findIndex(c => c.metadata.id === drop.cardId);
      if (cardIdx === -1) {
        targetIndex = cards.length;
      } else {
        targetIndex = drop.position === "above" ? cardIdx : cardIdx + 1;
      }
    }

    // Adjust targetIndex if dragging within the same list and the source is before the target
    const sourceCards = $boardData[drag.sourceListKey] || [];
    const sourceIdx = sourceCards.findIndex(c => c.filePath === drag.card.filePath);

    if (drag.sourceListKey === listKey && sourceIdx !== -1) {
      // No-op if dropping at the same position
      if (targetIndex === sourceIdx || targetIndex === sourceIdx + 1) {
        return;
      }
      // Adjust for removal of source card
      if (sourceIdx < targetIndex) {
        targetIndex--;
      }
    }

    // Build the target cards array without the dragged card for computing list_order
    const targetCards = (drag.sourceListKey === listKey)
      ? cards.filter(c => c.filePath !== drag.card.filePath)
      : cards;

    const newListOrder = computeListOrder(targetCards, targetIndex);

    // Capture original path before optimistic update (needed for the API call)
    const originalPath = drag.card.filePath;
    moveCardInBoard(originalPath, drag.sourceListKey, listKey, targetIndex, newListOrder);

    try {
      const result = await MoveCard(originalPath, listKey, newListOrder);
      // Cross-list moves change the filePath on disk — sync the store with the backend response
      if (result.filePath !== originalPath) {
        boardData.update(lists => {
          const cards = lists[listKey];
          if (cards) {
            const idx = cards.findIndex(c => c.metadata.id === drag.card.metadata.id);
            if (idx !== -1) {
              cards[idx] = { ...cards[idx], filePath: result.filePath, listName: result.listName };
            }
          }
          return lists;
        });
      }
    } catch (err) {
      addToast(`Failed to move card: ${err}`);
      initBoard(); // Reload board to recover consistent state
    }
  }

  // Handles drag over the list header — treats as drop at the top of the list.
  function handleHeaderDragOver(e, listKey) {
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    dragX = e.clientX;
    dragY = e.clientY;

    if (!$dragState) {
      return;
    }

    clearDropIndicators();
    const cards = $boardData[listKey] || [];
    if (cards.length > 0) {
      dropTarget.set({ listKey, cardId: cards[0].metadata.id, position: "above" });
    } else {
      dropTarget.set({ listKey, cardId: null, position: "below" });
    }

    // Show indicator at the top of the list body
    const listCol = e.currentTarget.closest('.list-column');
    if (listCol) {
      const listBody = listCol.querySelector('.list-body');
      if (listBody) {
        addIndicator(listBody, 'drop-top');
      }
    }
  }

  // Handles drop on the footer add-button area — drop at the bottom of the list.
  function handleFooterDragOver(e, listKey) {
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    dragX = e.clientX;
    dragY = e.clientY;
    if (!$dragState) {
      return;
    }

    clearDropIndicators();
    dropTarget.set({ listKey, cardId: null, position: "below" });
    handleAutoScroll(e);
  }

  // Auto-scrolls the board container horizontally and list bodies vertically during drag.
  function handleAutoScroll(e) {
    if (autoScrollRaf) {
      return;
    }
    autoScrollRaf = requestAnimationFrame(() => {
      autoScrollRaf = null;
      const edgeSize = 40;
      const speed = 12;

      // Horizontal auto-scroll on board container
      if (boardContainerEl) {
        const rect = boardContainerEl.getBoundingClientRect();

        if (e.clientX < rect.left + edgeSize) {
          boardContainerEl.scrollLeft -= speed;
        } else if (e.clientX > rect.right - edgeSize) {
          boardContainerEl.scrollLeft += speed;
        }
      }

      // Vertical auto-scroll on the nearest virtual-scroll-container
      const scrollContainer = e.target.closest('.virtual-scroll-container');
      if (scrollContainer) {
        const rect = scrollContainer.getBoundingClientRect();

        if (e.clientY < rect.top + edgeSize) {
          scrollContainer.scrollTop -= speed;
        } else if (e.clientY > rect.bottom - edgeSize) {
          scrollContainer.scrollTop += speed;
        }
      }
    });
  }

  // Toggles a list between collapsed and expanded, persisting to board.yaml.
  function toggleCollapse(listKey) {
    if (collapsedLists.has(listKey)) {
      collapsedLists.delete(listKey);
    } else {
      collapsedLists.add(listKey);
    }
    collapsedLists = collapsedLists;
    SaveCollapsedLists([...collapsedLists]).catch(e => addToast(`Failed to save collapsed state: ${e}`));
  }

  // Loads the board from the backend and unpacks lists + config into separate stores.
  async function initBoard() {
    try {
      const response = await LoadBoard("");
      boardData.set(response.lists);
      boardConfig.set(response.config?.lists || {});

      if (response.config?.labelsExpanded !== undefined && response.config.labelsExpanded !== null) {
        labelsExpanded.set(response.config.labelsExpanded);
      }
      if (response.config?.collapsedLists) {
        collapsedLists = new Set(response.config.collapsedLists);
      }
      isLoaded.set(true);
    } catch (e) {
      error = e.toString();
    }
  }

  // Returns the config title override if set, otherwise the formatted directory name.
  function getDisplayTitle(listKey, config) {
    const cfg = config[listKey];
    if (cfg && cfg.title) {
      return cfg.title;
    }
    return formatListName(listKey);
  }

  // Returns "count/limit" when a limit is set, otherwise just the count.
  function getCountDisplay(listKey, lists, config) {
    const count = lists[listKey]?.length || 0;
    const cfg = config[listKey];
    if (cfg && cfg.limit > 0) {
      return `${count}/${cfg.limit}`;
    }
    return `${count}`;
  }

  // Returns true when the card count exceeds the configured limit.
  function isOverLimit(listKey, lists, config) {
    const cfg = config[listKey];
    if (!cfg || cfg.limit <= 0) {
      return false;
    }
    return (lists[listKey]?.length || 0) > cfg.limit;
  }

  // Starts inline editing of a list title.
  function startEditTitle(listKey) {
    editingTitle = listKey;
    editTitleValue = getDisplayTitle(listKey, $boardConfig);
  }

  // Saves the edited title via backend and updates the config store.
  async function saveTitle(listKey) {
    editingTitle = null;
    const cfg = $boardConfig[listKey] || { title: "", limit: 0 };
    const newTitle = editTitleValue.trim();
    const formatted = formatListName(listKey);

    // If the user typed the same as the formatted default, treat as empty (no override)
    const titleToSave = newTitle === formatted ? "" : newTitle;

    try {
      await SaveListConfig(listKey, titleToSave, cfg.limit || 0);
      boardConfig.update(c => {
        if (titleToSave === "" && (cfg.limit || 0) === 0) {
          delete c[listKey];
        } else {
          c[listKey] = { ...cfg, title: titleToSave };
        }
        return c;
      });
    } catch (e) {
      addToast(`Failed to save list title: ${e}`);
    }
  }

  // Starts inline editing of a list's card limit.
  function startEditLimit(listKey) {
    editingLimit = listKey;
    const cfg = $boardConfig[listKey];
    editLimitValue = cfg?.limit || 0;
  }

  // Saves the edited limit via backend and updates the config store.
  async function saveLimit(listKey) {
    editingLimit = null;
    const cfg = $boardConfig[listKey] || { title: "", limit: 0 };
    const newLimit = Math.max(0, Math.floor(editLimitValue));

    try {
      await SaveListConfig(listKey, cfg.title || "", newLimit);
      boardConfig.update(c => {
        if ((cfg.title || "") === "" && newLimit === 0) {
          delete c[listKey];
        } else {
          c[listKey] = { ...cfg, limit: newLimit };
        }
        return c;
      });
    } catch (e) {
      addToast(`Failed to save list limit: ${e}`);
    }
  }

  // Handles keydown events on the title input — saves on Enter, cancels on Escape.
  function handleTitleKeydown(e, listKey) {
    if (e.key === "Enter") {
      saveTitle(listKey);
    } else if (e.key === "Escape") {
      editingTitle = null;
    }
  }

  // Handles keydown events on the limit input — saves on Enter, cancels on Escape.
  function handleLimitKeydown(e, listKey) {
    if (e.key === "Enter") {
      saveLimit(listKey);
    } else if (e.key === "Escape") {
      editingLimit = null;
    }
  }

  // Opens the draft-creation modal for the given list, defaulting to top placement.
  function createCard(listKey) {
    draftPosition.set("top");
    draftListKey.set(listKey);
  }

  // Opens the draft-creation modal for the given list, defaulting to bottom placement.
  function createCardBottom(listKey) {
    draftPosition.set("bottom");
    draftListKey.set(listKey);
  }

  // Handles global keydown: "n" creates a card in the first list (unless typing or modal is open).
  function handleGlobalKeydown(e) {
    if (e.key !== "n" || e.metaKey || e.ctrlKey || e.altKey) {
      return;
    }

    const tag = e.target.tagName;
    if (tag === "INPUT" || tag === "TEXTAREA" || e.target.isContentEditable) {
      return;
    }

    if ($selectedCard || $draftListKey) {
      return;
    }

    const keys = sortedListKeys($boardData);
    if (keys.length > 0) {
      e.preventDefault();
      createCard(keys[0]);
    }
  }

  // Clean up indicators, drop state, and auto-scroll when drag ends (drop or cancel).
  $: if (!$dragState) {
    clearDropIndicators();
    dropTarget.set(null);
    if (autoScrollRaf) {
      cancelAnimationFrame(autoScrollRaf);
      autoScrollRaf = null;
    }
  }

  onMount(initBoard);

  onDestroy(() => {
    if (autoScrollRaf) {
      cancelAnimationFrame(autoScrollRaf);
      autoScrollRaf = null;
    }
  });

</script>

<svelte:window on:keydown={handleGlobalKeydown} />

<main>
  <div class="top-bar">
    <h1>Daedalus</h1>
    <div class="top-bar-actions">
      <button class="top-btn" on:click={initBoard}>
        <svg viewBox="0 0 24 24" width="14" height="14">
          <path d="M23 4v6h-6" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </button>
      <button class="top-btn" class:active={$showMetrics} on:click={() => showMetrics.update(v => !v)}>
        <svg viewBox="0 0 24 24" width="14" height="14">
          <rect x="18" y="3" width="4" height="18" rx="1" fill="none" stroke="currentColor" stroke-width="2"/>
          <rect x="10" y="8" width="4" height="13" rx="1" fill="none" stroke="currentColor" stroke-width="2"/>
          <rect x="2" y="13" width="4" height="8" rx="1" fill="none" stroke="currentColor" stroke-width="2"/>
        </svg>
      </button>
    </div>
  </div>

  {#if error}
    <div class="error">{error}</div>
  {:else if $isLoaded}
    <div class="board-container" class:modal-open={$selectedCard || $draftListKey} bind:this={boardContainerEl}>
      {#each sortedListKeys($boardData) as listKey}
        {#if collapsedLists.has(listKey)}
          <div class="list-column collapsed" on:click={() => toggleCollapse(listKey)} on:keydown={e => e.key === 'Enter' && toggleCollapse(listKey)}>
            <span class="collapsed-title">{getDisplayTitle(listKey, $boardConfig)}</span>
            <span class="collapsed-count">{getCountDisplay(listKey, $boardData, $boardConfig)}</span>
          </div>
        {:else}
          <div class="list-column">
            <div class="list-header" on:dragenter={handleDragEnter} on:dragover={(e) => handleHeaderDragOver(e, listKey)} on:drop={(e) => handleDrop(e, listKey)}>
              {#if editingTitle === listKey}
                <input
                  class="edit-title-input"
                  type="text"
                  bind:value={editTitleValue}
                  on:blur={() => saveTitle(listKey)}
                  on:keydown={(e) => handleTitleKeydown(e, listKey)}
                  use:autoFocus
                />
              {:else}
                <button class="list-title-btn" on:click={() => startEditTitle(listKey)}>{getDisplayTitle(listKey, $boardConfig)}</button>
              {/if}
              <div class="header-right">
                <button class="collapse-btn" on:click={() => createCard(listKey)} title="Add card">
                  <svg viewBox="0 0 24 24" width="12" height="12">
                    <line x1="12" y1="5" x2="12" y2="19" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
                    <line x1="5" y1="12" x2="19" y2="12" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
                  </svg>
                </button>
                <button class="collapse-btn" on:click={() => toggleCollapse(listKey)} title="Collapse list">
                  <svg viewBox="0 0 24 24" width="12" height="12">
                    <polyline points="6 9 12 15 18 9" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                  </svg>
                </button>
                {#if editingLimit === listKey}
                  <input
                    class="edit-limit-input"
                    type="number"
                    min="0"
                    bind:value={editLimitValue}
                    on:blur={() => saveLimit(listKey)}
                    on:keydown={(e) => handleLimitKeydown(e, listKey)}
                    use:autoFocus
                  />
                {:else}
                  <button
                    class="count-btn"
                    class:over-limit={isOverLimit(listKey, $boardData, $boardConfig)}
                    on:click={() => startEditLimit(listKey)}
                  >{getCountDisplay(listKey, $boardData, $boardConfig)}</button>
                {/if}
              </div>
            </div>
            <div class="list-body" on:dragenter={handleDragEnter} on:dragover={(e) => handleDragOver(e, listKey)} on:dragleave={handleDragLeave} on:drop={(e) => handleDrop(e, listKey)}>
              <VirtualList items={$boardData[listKey]} component={Card} estimatedHeight={90} listKey={listKey} />
            </div>
            <button class="list-footer-add" on:click={() => createCardBottom(listKey)} on:dragenter={handleDragEnter} on:dragover={(e) => handleFooterDragOver(e, listKey)} on:drop={(e) => handleDrop(e, listKey)} title="Add card to bottom">
              <svg viewBox="0 0 24 24" width="12" height="12">
                <line x1="12" y1="5" x2="12" y2="19" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
                <line x1="5" y1="12" x2="19" y2="12" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
              </svg>
            </button>
          </div>
        {/if}
      {/each}
    </div>
  {:else}
    <div class="loading">Loading cards...</div>
  {/if}
  {#if $dragState}
    <div class="drag-ghost" style="left: {dragX + 10}px; top: {dragY - 20}px">
      {#if $dragState.card.metadata.labels?.length > 0}
        <div class="ghost-labels">
          {#each $dragState.card.metadata.labels as label}
            <span class="ghost-label" style="background: {labelColor(label)}"></span>
          {/each}
        </div>
      {/if}
      <div class="ghost-title">{$dragState.card.metadata.title}</div>
    </div>
  {/if}
  <CardDetail />
  <Metrics />
  <Toast />
</main>

<style>
  :global(body) {
    margin: 0;
    background-color: #1e2128;
    color: white;
    font-family: sans-serif;
    overflow: hidden;
  }

  main {
    display: flex;
    flex-direction: column;
    height: 100vh;
  }

  .top-bar {
    height: 50px;
    background: #181a1f;
    display: flex;
    align-items: center;
    padding: 0 20px;
    border-bottom: 1px solid #000;
  }

  .top-bar-actions {
    display: flex;
    gap: 6px;
    margin-left: auto;
  }

  .top-btn {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid transparent;
    color: #9fadbc;
    font-size: 0.78rem;
    font-weight: 500;
    padding: 5px 10px;
    border-radius: 4px;
    cursor: pointer;
    transition: background 0.15s, color 0.15s;
  }

  .top-btn:hover {
    background: rgba(255, 255, 255, 0.12);
    color: #c7d1db;
  }

  .top-btn.active {
    background: rgba(87, 157, 255, 0.15);
    color: #579dff;
    border-color: rgba(87, 157, 255, 0.3);
  }

  .top-btn.active:hover {
    background: rgba(87, 157, 255, 0.22);
  }

  .board-container {
    flex: 1;
    display: flex;
    overflow-x: auto;
    padding: 10px;
    gap: 10px;
    will-change: scroll-position;
  }

  .list-column {
    flex: 0 0 280px;
    display: flex;
    flex-direction: column;
    background: #22252b;
    border-radius: 8px;
    max-height: 100%;
    border: 1px solid #333;
    contain: layout style paint;
  }

  .list-column.collapsed {
    flex: 0 0 36px;
    cursor: pointer;
    align-items: center;
    padding: 12px 0;
    gap: 10px;
    transition: background 0.1s;
  }

  .list-column.collapsed:hover {
    background: #2a2d34;
  }

  .collapsed-title {
    writing-mode: vertical-rl;
    text-orientation: mixed;
    font-size: 0.8rem;
    font-weight: 600;
    color: #9fadbc;
    white-space: nowrap;
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .collapsed-count {
    writing-mode: vertical-rl;
    font-size: 0.7rem;
    color: #6b7a8d;
    flex-shrink: 0;
  }

  .list-header {
    padding: 8px 10px;
    border-bottom: 1px solid #333;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 4px;
    flex-shrink: 0;
  }

  .collapse-btn {
    all: unset;
    cursor: pointer;
    color: #6b7a8d;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    border-radius: 4px;
  }

  .collapse-btn:hover {
    color: #9fadbc;
    background: rgba(255, 255, 255, 0.08);
  }

  .list-title-btn {
    all: unset;
    font-size: 0.85rem;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    cursor: pointer;
    color: inherit;
  }

  .list-body {
    flex: 1;
    overflow: hidden;
  }

  .list-footer-add {
    all: unset;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    padding: 6px 0;
    cursor: pointer;
    color: #6b7a8d;
    border-top: 1px solid #333;
    box-sizing: border-box;
  }

  .list-footer-add:hover {
    color: #9fadbc;
    background: rgba(255, 255, 255, 0.04);
  }

  .count-btn {
    all: unset;
    background: #333;
    padding: 2px 8px;
    border-radius: 10px;
    font-size: 0.8rem;
    cursor: pointer;
    flex-shrink: 0;
    color: inherit;
  }

  .count-btn.over-limit {
    background: rgba(220, 50, 50, 0.3);
    color: #ff6b6b;
  }

  .edit-title-input {
    background: #181a1f;
    border: 1px solid #579dff;
    color: white;
    font-size: 0.85rem;
    font-weight: 600;
    padding: 2px 6px;
    border-radius: 4px;
    outline: none;
    width: 100%;
    min-width: 0;
    margin-right: 8px;
  }

  .edit-limit-input {
    background: #181a1f;
    border: 1px solid #579dff;
    color: white;
    font-size: 0.8rem;
    padding: 2px 6px;
    border-radius: 10px;
    outline: none;
    width: 60px;
    text-align: center;
    -moz-appearance: textfield;
    flex-shrink: 0;
  }

  .edit-limit-input::-webkit-inner-spin-button,
  .edit-limit-input::-webkit-outer-spin-button {
    -webkit-appearance: none;
    margin: 0;
  }

  .board-container.modal-open :global(.virtual-scroll-container) {
    overflow-y: hidden;
  }

  :global(.list-body.drop-top) {
    position: relative;
  }

  :global(.list-body.drop-top::before) {
    content: '';
    position: absolute;
    top: 0;
    left: 6px;
    right: 6px;
    height: 2px;
    background: #579dff;
    z-index: 2;
    border-radius: 1px;
  }

  :global(.item-slot.drop-above),
  :global(.item-slot.drop-below) {
    position: relative;
    z-index: 1;
  }

  :global(.item-slot.drop-above) {
    box-shadow: 0 -2px 0 0 #579dff;
  }

  :global(.item-slot.drop-below) {
    box-shadow: 0 2px 0 0 #579dff;
  }

  .drag-ghost {
    position: fixed;
    width: 250px;
    background: #2b303b;
    border-radius: 4px;
    padding: 8px 10px;
    color: #c7d1db;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, sans-serif;
    pointer-events: none;
    z-index: 10000;
    opacity: 0.9;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5);
    transform: rotate(3deg);
    border: 1px solid rgba(87, 157, 255, 0.4);
  }

  .ghost-labels {
    display: flex;
    gap: 4px;
    margin-bottom: 6px;
  }

  .ghost-label {
    min-width: 28px;
    height: 8px;
    border-radius: 3px;
  }

  .ghost-title {
    font-size: 0.85rem;
    font-weight: 400;
    line-height: 1.3;
    word-break: break-word;
    overflow: hidden;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
  }
</style>
