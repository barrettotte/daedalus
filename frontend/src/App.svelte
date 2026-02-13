<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { SvelteSet } from "svelte/reactivity";
  import { LoadBoard, SaveListConfig, SaveLabelsExpanded, SaveCollapsedLists, MoveCard, DeleteCard } from "../wailsjs/go/main/App";
  import { boardData, boardConfig, sortedListKeys, isLoaded, selectedCard, draftListKey, draftPosition, showMetrics, labelsExpanded, dragState, dropTarget, focusedCard, openInEditMode, moveCardInBoard, removeCardFromBoard, computeListOrder, addToast, isAtLimit } from "./stores/board";
  import type { BoardLists, BoardConfigMap } from "./stores/board";
  import type { daedalus } from "../wailsjs/go/models";
  import { formatListName, autoFocus, labelColor } from "./lib/utils";
  import VirtualList from "./components/VirtualList.svelte";
  import Card from "./components/Card.svelte";
  import CardDetail from "./components/CardDetail.svelte";
  import Metrics from "./components/Metrics.svelte";
  import Toast from "./components/Toast.svelte";
  import KeyboardHelp from "./components/KeyboardHelp.svelte";

  let error = $state("");
  let editingTitle: string | null = $state(null);
  let editingLimit: string | null = $state(null);
  let editTitleValue = $state("");
  let editLimitValue = $state(0);
  let collapsedLists = $state(new SvelteSet<string>());
  let showKeyboardHelp = $state(false);
  let confirmingFocusDelete = $state(false);
  let boardContainerEl: HTMLDivElement | undefined = $state(undefined);
  let dragX = $state(0);
  let dragY = $state(0);
  let autoScrollRaf: number | null = null;
  let autoScrollVContainer: Element | null = null;
  let autoScrollVSpeed = 0;
  let autoScrollHSpeed = 0;
  let activeIndicators: Set<Element> = new Set();

  // Clears drop indicator classes from tracked elements (avoids querySelectorAll on every dragover).
  function clearDropIndicators(): void {
    for (const el of activeIndicators) {
      el.classList.remove('drop-above', 'drop-below', 'drop-top', 'drop-bottom');
    }
    activeIndicators.clear();
  }

  // Adds a drop indicator class to an element and tracks it for efficient cleanup.
  function addIndicator(el: Element, cls: string): void {
    el.classList.add(cls);
    activeIndicators.add(el);
  }

  // Allows the element to be a valid drop target
  function handleDragEnter(e: DragEvent): void {
    e.preventDefault();
    e.dataTransfer!.dropEffect = "move";
  }

  // Handles dragover on a list body - positions the drop indicator and triggers auto-scroll.
  function handleDragOver(e: DragEvent, listKey: string): void {
    e.preventDefault();
    dragX = e.clientX;
    dragY = e.clientY;
    if (!$dragState) {
      e.dataTransfer!.dropEffect = "move";
      return;
    }

    // Block cross-list drops into lists that are at their card limit.
    if ($dragState.sourceListKey !== listKey && isAtLimit(listKey, $boardData, $boardConfig)) {
      e.dataTransfer!.dropEffect = "none";
      clearDropIndicators();
      return;
    }

    e.dataTransfer!.dropEffect = "move";
    clearDropIndicators();

    // Find the closest item-slot under the cursor
    const slot = (e.target as HTMLElement).closest('[data-card-id]');
    if (slot) {
      const rect = slot.getBoundingClientRect();
      const midY = rect.top + rect.height / 2;

      if (e.clientY < midY) {
        // Top half - insert before this card
        addIndicator(slot, 'drop-above');
        dropTarget.set({ listKey, cardId: Number((slot as HTMLElement).dataset.cardId), position: "above" });
      } else {
        // Bottom half - insert after this card
        const next = slot.nextElementSibling;

        if (next && next.hasAttribute('data-card-id')) {
          addIndicator(next, 'drop-above');
          dropTarget.set({ listKey, cardId: Number((next as HTMLElement).dataset.cardId), position: "above" });
        } else {
          // Last card - show indicator below
          addIndicator(slot, 'drop-below');
          dropTarget.set({ listKey, cardId: Number((slot as HTMLElement).dataset.cardId), position: "below" });
        }
      }

    } else {
      // No card under cursor. Determine if near top or bottom of the list
      const listBody = e.currentTarget as HTMLElement;
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

  // Handles dragleave
  function handleDragLeave(e: DragEvent): void {
    const related = e.relatedTarget as Node | null;
    if (related && !(e.currentTarget as HTMLElement).contains(related)) {
      clearDropIndicators();
    }
  }

  // Handles drop on a list body
  async function handleDrop(e: DragEvent, listKey: string): Promise<void> {
    e.preventDefault();
    clearDropIndicators();

    const drag = $dragState;
    const drop = $dropTarget;
    dragState.set(null);
    dropTarget.set(null);

    if (!drag || !drop) {
      return;
    }

    // Block cross-list moves into lists at their card limit.
    if (drag.sourceListKey !== listKey && isAtLimit(listKey, $boardData, $boardConfig)) {
      addToast("List is at its card limit");
      return;
    }

    const cards = $boardData[listKey] || [];
    let targetIndex: number;

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
      // Cross-list moves change the filePath on disk. Sync the store with the backend response
      if (result.filePath !== originalPath) {
        boardData.update(lists => {
          const listCards = lists[listKey];
          if (listCards) {
            const idx = listCards.findIndex(c => c.metadata.id === drag.card.metadata.id);
            if (idx !== -1) {
              listCards[idx] = { ...listCards[idx], filePath: result.filePath, listName: result.listName } as daedalus.KanbanCard;
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

  // Handles drag over the list header
  function handleHeaderDragOver(e: DragEvent, listKey: string): void {
    e.preventDefault();
    dragX = e.clientX;
    dragY = e.clientY;

    if (!$dragState) {
      e.dataTransfer!.dropEffect = "move";
      return;
    }

    // Block cross-list drops into lists that are at their card limit.
    if ($dragState.sourceListKey !== listKey && isAtLimit(listKey, $boardData, $boardConfig)) {
      e.dataTransfer!.dropEffect = "none";
      clearDropIndicators();
      return;
    }

    e.dataTransfer!.dropEffect = "move";
    clearDropIndicators();
    const cards = $boardData[listKey] || [];
    if (cards.length > 0) {
      dropTarget.set({ listKey, cardId: cards[0].metadata.id, position: "above" });
    } else {
      dropTarget.set({ listKey, cardId: null, position: "below" });
    }

    // Show indicator at the top of the list body
    const listCol = (e.currentTarget as HTMLElement).closest('.list-column');
    if (listCol) {
      const listBody = listCol.querySelector('.list-body');
      if (listBody) {
        addIndicator(listBody, 'drop-top');
      }
    }
  }

  // Handles drop on the footer add-button area
  function handleFooterDragOver(e: DragEvent, listKey: string): void {
    e.preventDefault();
    dragX = e.clientX;
    dragY = e.clientY;
    if (!$dragState) {
      e.dataTransfer!.dropEffect = "move";
      return;
    }

    // Block cross-list drops into lists that are at their card limit.
    if ($dragState.sourceListKey !== listKey && isAtLimit(listKey, $boardData, $boardConfig)) {
      e.dataTransfer!.dropEffect = "none";
      clearDropIndicators();
      return;
    }

    e.dataTransfer!.dropEffect = "move";
    clearDropIndicators();
    dropTarget.set({ listKey, cardId: null, position: "below" });
    handleAutoScroll(e);
  }

  // Updates auto-scroll speeds based on cursor position
  function handleAutoScroll(e: DragEvent): void {
    const edgeSize = 40;
    const speed = 10;

    let hSpeed = 0;
    let vSpeed = 0;
    let vContainer: Element | null = null;

    // Horizontal scroll
    if (boardContainerEl) {
      const rect = boardContainerEl.getBoundingClientRect();
      if (e.clientX < rect.left + edgeSize) {
        hSpeed = -speed;
      } else if (e.clientX > rect.right - edgeSize) {
        hSpeed = speed;
      }
    }

    // Vertical scroll
    const target = e.target as HTMLElement;
    vContainer = target.closest('.virtual-scroll-container')
      || (e.currentTarget as HTMLElement).querySelector('.virtual-scroll-container');
    if (vContainer) {
      const rect = vContainer.getBoundingClientRect();
      if (e.clientY < rect.top + edgeSize) {
        vSpeed = -speed;
      } else if (e.clientY > rect.bottom - edgeSize) {
        vSpeed = speed;
      }
    }

    autoScrollHSpeed = hSpeed;
    autoScrollVSpeed = vSpeed;
    autoScrollVContainer = vContainer;

    // Start the loop if scrolling is needed and not already running
    if ((hSpeed !== 0 || vSpeed !== 0) && !autoScrollRaf) {
      autoScrollTick();
    }
  }

  // Continuous scroll loop
  function autoScrollTick(): void {
    if (autoScrollHSpeed === 0 && autoScrollVSpeed === 0) {
      autoScrollRaf = null;
      return;
    }
    if (autoScrollHSpeed !== 0 && boardContainerEl) {
      boardContainerEl.scrollLeft += autoScrollHSpeed;
    }
    if (autoScrollVSpeed !== 0 && autoScrollVContainer) {
      autoScrollVContainer.scrollTop += autoScrollVSpeed;
    }
    autoScrollRaf = requestAnimationFrame(autoScrollTick);
  }

  // Stops the auto-scroll loop.
  function stopAutoScroll(): void {
    autoScrollHSpeed = 0;
    autoScrollVSpeed = 0;
    autoScrollVContainer = null;
    if (autoScrollRaf) {
      cancelAnimationFrame(autoScrollRaf);
      autoScrollRaf = null;
    }
  }

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

  // Returns the config title override if set, otherwise the formatted directory name.
  function getDisplayTitle(listKey: string, config: BoardConfigMap): string {
    const cfg = config[listKey];
    if (cfg && cfg.title) {
      return cfg.title;
    }
    return formatListName(listKey);
  }

  // Returns "count/limit" when a limit is set, otherwise just the count.
  function getCountDisplay(listKey: string, lists: BoardLists, config: BoardConfigMap): string {
    const count = lists[listKey]?.length || 0;
    const cfg = config[listKey];
    if (cfg && cfg.limit > 0) {
      return `${count}/${cfg.limit}`;
    }
    return `${count}`;
  }

  // Returns true when the card count exceeds the configured limit.
  function isOverLimit(listKey: string, lists: BoardLists, config: BoardConfigMap): boolean {
    const cfg = config[listKey];
    if (!cfg || cfg.limit <= 0) {
      return false;
    }
    return (lists[listKey]?.length || 0) > cfg.limit;
  }

  // Starts inline editing of a list title.
  function startEditTitle(listKey: string): void {
    editingTitle = listKey;
    editTitleValue = getDisplayTitle(listKey, $boardConfig);
  }

  // Saves the edited title via backend and updates the config store.
  async function saveTitle(listKey: string): Promise<void> {
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
  function startEditLimit(listKey: string): void {
    editingLimit = listKey;
    const cfg = $boardConfig[listKey];
    editLimitValue = cfg?.limit || 0;
  }

  // Saves the edited limit via backend and updates the config store.
  async function saveLimit(listKey: string): Promise<void> {
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

  // Handles keydown events on the title input
  function handleTitleKeydown(e: KeyboardEvent, listKey: string): void {
    if (e.key === "Enter") {
      saveTitle(listKey);
    } else if (e.key === "Escape") {
      editingTitle = null;
    }
  }

  // Handles keydown events on the limit input
  function handleLimitKeydown(e: KeyboardEvent, listKey: string): void {
    if (e.key === "Enter") {
      saveLimit(listKey);
    } else if (e.key === "Escape") {
      editingLimit = null;
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
            <div class="list-header" role="group" ondragenter={handleDragEnter} ondragover={(e) => handleHeaderDragOver(e, listKey)} ondrop={(e) => handleDrop(e, listKey)}>
              {#if editingTitle === listKey}
                <input
                  class="edit-title-input"
                  type="text"
                  bind:value={editTitleValue}
                  onblur={() => saveTitle(listKey)}
                  onkeydown={(e) => handleTitleKeydown(e, listKey)}
                  use:autoFocus
                />
              {:else}
                <button class="list-title-btn" onclick={() => startEditTitle(listKey)}>{getDisplayTitle(listKey, $boardConfig)}</button>
              {/if}
              <div class="header-right">
                <button class="collapse-btn" onclick={() => createCard(listKey)} title="Add card">
                  <svg viewBox="0 0 24 24" width="12" height="12">
                    <line x1="12" y1="5" x2="12" y2="19" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
                    <line x1="5" y1="12" x2="19" y2="12" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
                  </svg>
                </button>
                <button class="collapse-btn" onclick={() => toggleCollapse(listKey)} title="Collapse list">
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
                    onblur={() => saveLimit(listKey)}
                    onkeydown={(e) => handleLimitKeydown(e, listKey)}
                    use:autoFocus
                  />
                {:else}
                  <button
                    class="count-btn"
                    class:over-limit={isOverLimit(listKey, $boardData, $boardConfig)}
                    onclick={() => startEditLimit(listKey)}
                  >{getCountDisplay(listKey, $boardData, $boardConfig)}</button>
                {/if}
              </div>
            </div>
            <div class="list-body" role="list" ondragenter={handleDragEnter} ondragover={(e) => handleDragOver(e, listKey)} ondragleave={handleDragLeave} ondrop={(e) => handleDrop(e, listKey)}>
              <VirtualList items={$boardData[listKey]} component={Card} estimatedHeight={90} listKey={listKey} focusIndex={$focusedCard?.listKey === listKey ? $focusedCard.cardIndex : -1} />
            </div>
            <button class="list-footer-add" onclick={() => createCardBottom(listKey)} ondragenter={handleDragEnter} ondragover={(e) => handleFooterDragOver(e, listKey)} ondrop={(e) => handleDrop(e, listKey)} title="Add card to bottom">
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
  {#if confirmingFocusDelete}
    <div class="focus-delete-confirm">Press Delete again to confirm, Escape to cancel</div>
  {/if}
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

  .list-header {
    padding: 8px 10px;
    border-bottom: 1px solid var(--color-border-medium);
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
    color: var(--color-text-muted);
    display: flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    border-radius: 4px;

    &:hover {
      color: var(--color-text-secondary);
      background: var(--overlay-hover);
    }
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
    color: var(--color-text-muted);
    border-top: 1px solid var(--color-border-medium);
    box-sizing: border-box;

    &:hover {
      color: var(--color-text-secondary);
      background: var(--overlay-hover-faint);
    }
  }

  .count-btn {
    all: unset;
    background: var(--color-border-medium);
    padding: 2px 8px;
    border-radius: 10px;
    font-size: 0.8rem;
    cursor: pointer;
    flex-shrink: 0;
    color: inherit;

    &.over-limit {
      background: var(--overlay-error-limit);
      color: #ff6b6b;
    }
  }

  .edit-title-input {
    background: var(--color-bg-inset);
    border: 1px solid var(--color-accent);
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
    background: var(--color-bg-inset);
    border: 1px solid var(--color-accent);
    color: white;
    font-size: 0.8rem;
    padding: 2px 6px;
    border-radius: 10px;
    outline: none;
    width: 60px;
    text-align: center;
    appearance: textfield;
    -moz-appearance: textfield;
    flex-shrink: 0;

    &::-webkit-inner-spin-button,
    &::-webkit-outer-spin-button {
      appearance: none;
      -webkit-appearance: none;
      margin: 0;
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
