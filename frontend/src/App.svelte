<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { SvelteSet } from "svelte/reactivity";
  import {
    LoadBoard, SaveLabelsExpanded,
    SaveCollapsedLists, DeleteCard,
  } from "../wailsjs/go/main/App";
  import {
    boardData, boardConfig, sortedListKeys, isLoaded, selectedCard,
    draftListKey, draftPosition, showMetrics, labelsExpanded,
    dragState, dropTarget, focusedCard, openInEditMode,
    removeCardFromBoard, addToast, isAtLimit,
  } from "./stores/board";
  import { labelColor, getDisplayTitle, getCountDisplay } from "./lib/utils";
  import {
    dragPos, setBoardContainer, clearDropIndicators,
    handleDragEnter, handleDragOver, handleDragLeave, handleDrop,
    handleFooterDragOver, stopAutoScroll,
  } from "./lib/drag";
  import VirtualList from "./components/VirtualList.svelte";
  import Card from "./components/Card.svelte";
  import CardDetail from "./components/CardDetail.svelte";
  import DraftCard from "./components/DraftCard.svelte";
  import ListHeader from "./components/ListHeader.svelte";
  import Metrics from "./components/Metrics.svelte";
  import Toast from "./components/Toast.svelte";
  import KeyboardHelp from "./components/KeyboardHelp.svelte";

  let error = $state("");
  let collapsedLists = $state(new SvelteSet<string>());
  let showKeyboardHelp = $state(false);
  let confirmingFocusDelete = $state(false);
  let boardContainerEl: HTMLDivElement | undefined = $state(undefined);

  // Toggles a list between collapsed and expanded, persisting to board.yaml.
  function toggleCollapse(listKey: string): void {
    if (collapsedLists.has(listKey)) {
      collapsedLists.delete(listKey);
    } else {
      collapsedLists.add(listKey);
    }
    SaveCollapsedLists([...collapsedLists]).catch(e => addToast(`Failed to save collapsed state: ${e}`));
  }

  // Loads the board from the backend and unpacks lists + config into separate stores.
  async function initBoard(): Promise<void> {
    error = "";
    try {
      const response = await LoadBoard("");
      boardData.set(response.lists);
      boardConfig.set(response.config?.lists || {} as Record<string, any>);

      if (response.config?.labelsExpanded !== undefined && response.config.labelsExpanded !== null) {
        labelsExpanded.set(response.config.labelsExpanded);
      }
      if (response.config?.collapsedLists) {
        collapsedLists = new SvelteSet(response.config.collapsedLists);
      }
      isLoaded.set(true);
    } catch (e) {
      error = (e as Error).toString();
    }
  }

  // Opens the draft-creation modal for the given list, defaulting to top placement.
  function createCard(listKey: string): void {
    if (isAtLimit(listKey, $boardData, $boardConfig)) {
      addToast("List is at its card limit");
      return;
    }
    draftPosition.set("top");
    draftListKey.set(listKey);
  }

  // Opens the draft-creation modal for the given list, defaulting to bottom placement.
  function createCardBottom(listKey: string): void {
    if (isAtLimit(listKey, $boardData, $boardConfig)) {
      addToast("List is at its card limit");
      return;
    }
    draftPosition.set("bottom");
    draftListKey.set(listKey);
  }

  // Scrolls the board container to make the focused list column visible.
  function scrollListIntoView(listKey: string): void {
    if (!boardContainerEl) {
      return;
    }
    const columns = boardContainerEl.querySelectorAll('.list-column');
    const keys = sortedListKeys($boardData);
    const idx = keys.indexOf(listKey);
    if (idx >= 0 && columns[idx]) {
      columns[idx].scrollIntoView({ block: 'nearest', inline: 'nearest' });
    }
  }

  // Deletes the currently focused card after confirmation.
  async function deleteFocusedCard(): Promise<void> {
    const focus = $focusedCard;
    if (!focus) {
      return;
    }

    const cards = $boardData[focus.listKey] || [];
    const card = cards[focus.cardIndex];
    if (!card) {
      return;
    }

    try {
      await DeleteCard(card.filePath);
      removeCardFromBoard(card.filePath);

      // Move focus to next card, or previous, or clear
      const remaining = ($boardData[focus.listKey] || []).length;
      if (remaining === 0) {
        focusedCard.set(null);
      } else {
        focusedCard.set({ listKey: focus.listKey, cardIndex: Math.min(focus.cardIndex, remaining - 1) });
      }
    } catch (err) {
      addToast(`Failed to delete card: ${err}`);
    }

    confirmingFocusDelete = false;
  }

  // Handles all global keyboard shortcuts for board navigation and actions.
  function handleGlobalKeydown(e: KeyboardEvent): void {
    const tag = (e.target as HTMLElement).tagName;
    const isTyping = tag === "INPUT" || tag === "TEXTAREA" || (e.target as HTMLElement).isContentEditable;

    // Escape closes help overlay first; all other keys ignored while it's open.
    if (showKeyboardHelp) {
      if (e.key === "Escape") {
        e.preventDefault();
        showKeyboardHelp = false;
      }
      return;
    }

    // Escape cancels delete confirmation
    if (e.key === "Escape" && confirmingFocusDelete) {
      e.preventDefault();
      confirmingFocusDelete = false;
      return;
    }

    // Confirm delete with Enter when confirming
    if (e.key === "Enter" && confirmingFocusDelete) {
      e.preventDefault();
      deleteFocusedCard();
      return;
    }

    // Skip all shortcuts when typing in inputs
    if (isTyping) {
      return;
    }

    // Skip when modal is open (CardDetail handles its own keys)
    if ($selectedCard || $draftListKey) {
      return;
    }

    const keys = sortedListKeys($boardData);
    if (keys.length === 0) {
      return;
    }

    // ? - Toggle keyboard help
    if (e.key === "?") {
      e.preventDefault();
      showKeyboardHelp = !showKeyboardHelp;
      return;
    }

    // N - Create new card
    if (e.key === "n" && !e.metaKey && !e.ctrlKey && !e.altKey) {
      e.preventDefault();
      createCard(keys[0]);
      return;
    }

    // Arrow keys - navigate focus (skip if Ctrl/Cmd held)
    if ((e.key === "ArrowUp" || e.key === "ArrowDown" || e.key === "ArrowLeft" || e.key === "ArrowRight") && !e.ctrlKey && !e.metaKey) {
      e.preventDefault();
      confirmingFocusDelete = false;

      const focus = $focusedCard;

      if (!focus) {
        // No current focus - start at first card of first non-collapsed list
        for (const key of keys) {
          if (!collapsedLists.has(key) && ($boardData[key] || []).length > 0) {
            focusedCard.set({ listKey: key, cardIndex: 0 });
            scrollListIntoView(key);
            break;
          }
        }
        return;
      }

      if (e.key === "ArrowUp") {
        if (focus.cardIndex > 0) {
          focusedCard.set({ listKey: focus.listKey, cardIndex: focus.cardIndex - 1 });
        }
      } else if (e.key === "ArrowDown") {
        const cards = $boardData[focus.listKey] || [];
        if (focus.cardIndex < cards.length - 1) {
          focusedCard.set({ listKey: focus.listKey, cardIndex: focus.cardIndex + 1 });
        }
      } else if (e.key === "ArrowLeft" || e.key === "ArrowRight") {
        const listIdx = keys.indexOf(focus.listKey);
        const delta = e.key === "ArrowLeft" ? -1 : 1;
        let targetIdx = listIdx + delta;

        // Skip collapsed or empty lists
        while (targetIdx >= 0 && targetIdx < keys.length) {
          const targetKey = keys[targetIdx];
          if (!collapsedLists.has(targetKey) && ($boardData[targetKey] || []).length > 0) {
            break;
          }
          targetIdx += delta;
        }

        if (targetIdx >= 0 && targetIdx < keys.length) {
          const targetKey = keys[targetIdx];
          const targetCards = $boardData[targetKey] || [];
          const clampedIndex = Math.min(focus.cardIndex, targetCards.length - 1);
          focusedCard.set({ listKey: targetKey, cardIndex: clampedIndex });
          scrollListIntoView(targetKey);
        }
      }
      return;
    }

    // Enter - open focused card
    if (e.key === "Enter" && $focusedCard) {
      e.preventDefault();
      const cards = $boardData[$focusedCard.listKey] || [];
      const card = cards[$focusedCard.cardIndex];
      if (card) {
        selectedCard.set(card);
      }
      return;
    }

    // Escape - clear focus
    if (e.key === "Escape" && $focusedCard) {
      e.preventDefault();
      focusedCard.set(null);
      confirmingFocusDelete = false;
      return;
    }

    // E - open focused card in edit mode
    if (e.key === "e" && !e.ctrlKey && !e.metaKey && $focusedCard) {
      e.preventDefault();
      const cards = $boardData[$focusedCard.listKey] || [];
      const card = cards[$focusedCard.cardIndex];
      if (card) {
        openInEditMode.set(true);
        selectedCard.set(card);
      }
      return;
    }

    // Delete - delete focused card with confirmation
    if (e.key === "Delete" && $focusedCard) {
      e.preventDefault();
      if (confirmingFocusDelete) {
        deleteFocusedCard();
      } else {
        confirmingFocusDelete = true;
      }
      return;
    }
  }

  // Clean up indicators, drop state, and auto-scroll when drag ends (drop or cancel).
  $effect(() => {
    if (!$dragState) {
      clearDropIndicators();
      dropTarget.set(null);
      stopAutoScroll();
    }
  });

  // Sync the board container DOM ref to the drag module for horizontal auto-scroll.
  $effect(() => {
    setBoardContainer(boardContainerEl);
  });

  onMount(initBoard);

  onDestroy(() => {
    stopAutoScroll();
  });

</script>

<svelte:window onkeydown={handleGlobalKeydown} />

<main>
  <div class="top-bar">
    <h1>Daedalus</h1>
    <div class="top-bar-actions">
      <button class="top-btn" onclick={initBoard} title="Reload board">
        <svg viewBox="0 0 24 24" width="14" height="14">
          <path d="M23 4v6h-6" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </button>
      <button class="top-btn" class:active={$showMetrics} onclick={() => showMetrics.update(v => !v)} title="Toggle metrics">
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
          <div class="list-column collapsed" role="button" tabindex="0" onclick={() => toggleCollapse(listKey)} onkeydown={e => e.key === 'Enter' && toggleCollapse(listKey)}>
            <span class="collapsed-title">{getDisplayTitle(listKey, $boardConfig)}</span>
            <span class="collapsed-count">{getCountDisplay(listKey, $boardData, $boardConfig)}</span>
          </div>
        {:else}
          <div class="list-column" class:list-full={$dragState && $dragState.sourceListKey !== listKey && isAtLimit(listKey, $boardData, $boardConfig)}>
            <ListHeader {listKey}
              oncreatecard={() => createCard(listKey)}
              oncollapse={() => toggleCollapse(listKey)}
              onreload={initBoard}
            />
            <div class="list-body" role="list"
              ondragenter={handleDragEnter}
              ondragover={(e) => handleDragOver(e, listKey)}
              ondragleave={handleDragLeave}
              ondrop={(e) => handleDrop(e, listKey, initBoard)}
            >
              <VirtualList items={$boardData[listKey]} component={Card} estimatedHeight={90}
                listKey={listKey} focusIndex={$focusedCard?.listKey === listKey ? $focusedCard.cardIndex : -1}
              />
            </div>
            <button
              class="list-footer-add"
              onclick={() => createCardBottom(listKey)}
              ondragenter={handleDragEnter}
              ondragover={(e) => handleFooterDragOver(e, listKey)}
              ondrop={(e) => handleDrop(e, listKey, initBoard)}
              title="Add card to bottom"
            >
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
    <div class="drag-ghost" style="left: {$dragPos.x + 10}px; top: {$dragPos.y - 20}px">
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
  {#if confirmingFocusDelete}
    <div class="focus-delete-confirm">Press Delete again to confirm, Escape to cancel</div>
  {/if}
  <DraftCard />
  <CardDetail />
  <Metrics />
  <Toast />
  {#if showKeyboardHelp}
    <KeyboardHelp onclose={() => showKeyboardHelp = false} />
  {/if}
</main>

<style lang="scss">
  :global(body) {
    margin: 0;
    background-color: var(--color-bg-base);
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
    background: var(--color-bg-inset);
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
    background: var(--overlay-hover-light);
    border: 1px solid transparent;
    color: var(--color-text-secondary);
    font-size: 0.78rem;
    font-weight: 500;
    padding: 5px 10px;
    border-radius: 4px;
    cursor: pointer;
    transition: background 0.15s, color 0.15s;

    &:hover {
      background: var(--overlay-hover-medium);
      color: var(--color-text-primary);
    }

    &.active {
      background: var(--overlay-accent);
      color: var(--color-accent);
      border-color: var(--overlay-accent-border);

      &:hover {
        background: var(--overlay-accent-medium);
      }
    }
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
    background: var(--color-bg-list);
    border-radius: 8px;
    max-height: 100%;
    border: 1px solid var(--color-border-medium);
    contain: layout style paint;

    &.collapsed {
      flex: 0 0 36px;
      cursor: pointer;
      align-items: center;
      padding: 12px 0;
      gap: 10px;
      transition: background 0.1s;

      &:hover {
        background: var(--color-bg-surface-alt);
      }
    }

    &.list-full {
      opacity: 0.5;
      pointer-events: auto;
    }
  }

  .collapsed-title {
    writing-mode: vertical-rl;
    text-orientation: mixed;
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--color-text-secondary);
    white-space: nowrap;
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .collapsed-count {
    writing-mode: vertical-rl;
    font-size: 0.7rem;
    color: var(--color-text-muted);
    flex-shrink: 0;
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
    color: var(--color-text-muted);
    border-top: 1px solid var(--color-border-medium);
    box-sizing: border-box;

    &:hover {
      color: var(--color-text-secondary);
      background: var(--overlay-hover-faint);
    }
  }

  .focus-delete-confirm {
    position: fixed;
    bottom: 20px;
    left: 50%;
    transform: translateX(-50%);
    background: rgba(200, 55, 44, 0.9);
    color: #fff;
    padding: 8px 16px;
    border-radius: 6px;
    font-size: 0.85rem;
    font-weight: 500;
    z-index: 900;
    pointer-events: none;
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
    height: 3px;
    background: var(--color-accent);
    z-index: 2;
  }

  :global(.item-slot.drop-above::before) {
    content: '';
    position: absolute;
    top: 0;
    left: 6px;
    right: 6px;
    height: 3px;
    background: var(--color-accent);
    z-index: 2;
  }

  :global(.item-slot.drop-below::after) {
    content: '';
    position: absolute;
    bottom: 6px;
    left: 6px;
    right: 6px;
    height: 3px;
    background: var(--color-accent);
    z-index: 2;
  }

  :global(.list-body.drop-top),
  :global(.item-slot.drop-above),
  :global(.item-slot.drop-below) {
    position: relative;
    z-index: 1;
  }

  .drag-ghost {
    position: fixed;
    width: 250px;
    background: var(--color-bg-surface);
    border-radius: 4px;
    padding: 8px 10px;
    color: var(--color-text-primary);
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, sans-serif;
    pointer-events: none;
    z-index: 10000;
    opacity: 0.9;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5);
    transform: rotate(3deg);
    border: 1px solid var(--overlay-accent-strong);

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
      line-clamp: 3;
      -webkit-line-clamp: 3;
      -webkit-box-orient: vertical;
    }
  }
</style>
