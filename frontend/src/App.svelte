<script lang="ts">
  // Main board view. Renders list columns, handles collapse/pin/lock state, and coordinates modals.

  import { onMount, onDestroy } from "svelte";
  import { SvelteSet } from "svelte/reactivity";
  import { WindowSetTitle } from "../wailsjs/runtime/runtime";
  import {
    LoadBoard, SaveCollapsedLists, SaveHalfCollapsedLists, SaveLockedLists,
    SavePinnedLists, DeleteCard, SaveListOrder, DeleteList,
  } from "../wailsjs/go/main/App";
  import { get } from "svelte/store";
  import {
    boardData, boardTitle, boardConfig, boardPath, sortedListKeys, isLoaded,
    selectedCard, draftListKey, draftPosition,
    labelsExpanded, labelColors, dragState, dropTarget, focusedCard, openInEditMode,
    removeCardFromBoard, addToast, isAtLimit, isLocked, listOrder,
    searchQuery, filteredBoardData,
  } from "./stores/board";
  import type { daedalus } from "../wailsjs/go/models";
  import { labelColor, getDisplayTitle, getCountDisplay, HALF_COLLAPSED_CARD_LIMIT } from "./lib/utils";
  import { handleBoardKeydown } from "./lib/keyboard";
  import {
    dragPos, setBoardContainer, clearDropIndicators,
    handleDragEnter, handleDragOver, handleDragLeave, handleDrop,
    handleFooterDragOver, handleAutoScroll, stopAutoScroll,
  } from "./lib/drag";
  import VirtualList from "./components/VirtualList.svelte";
  import Card from "./components/Card.svelte";
  import CardDetail from "./components/CardDetail.svelte";
  import DraftCard from "./components/DraftCard.svelte";
  import ListHeader from "./components/ListHeader.svelte";
  import Metrics from "./components/Metrics.svelte";
  import Toast from "./components/Toast.svelte";
  import KeyboardHelp from "./components/KeyboardHelp.svelte";
  import About from "./components/About.svelte";
  import LabelColorEditor from "./components/LabelColorEditor.svelte";
  import TopBar from "./components/TopBar.svelte";
  import Icon from "./components/Icon.svelte";

  let error = $state("");
  let collapsedLists = $state(new SvelteSet<string>());
  let halfCollapsedLists = $state(new SvelteSet<string>());
  let lockedLists = $state(new SvelteSet<string>());
  let pinnedLeftLists = $state(new SvelteSet<string>());
  let pinnedRightLists = $state(new SvelteSet<string>());
  let showKeyboardHelp = $state(false);
  let showAbout = $state(false);
  let showLabelEditor = $state(false);
  let showYearProgress = $state(false);
  let darkMode = $state(true);
  let confirmingFocusDelete = $state(false);
  let boardContainerEl: HTMLDivElement | undefined = $state(undefined);
  let searchOpen = $state(false);
  let listDragging = $state<string | null>(null);
  let listDropTarget = $state<string | null>(null);
  let listDropSide = $state<'left' | 'right' | null>(null);
  let dropLineX = $state(0);
  let dropLineTop = $state(0);
  let dropLineHeight = $state(0);

  // Opens the search bar with an optional prefill string (used by keyboard handler).
  function openSearch(prefill: string = ""): void {
    if (prefill) {
      searchQuery.set(prefill);
    }
    searchOpen = true;
  }

  // Persists both collapse states to board.yaml.
  function saveCollapseState(): void {
    SaveCollapsedLists([...collapsedLists]).catch(e => addToast(`Failed to save collapsed state: ${e}`));
    SaveHalfCollapsedLists([...halfCollapsedLists]).catch(e => addToast(`Failed to save half-collapsed state: ${e}`));
  }

  // Toggles a list into or out of fully collapsed state.
  function toggleFullCollapse(listKey: string): void {
    halfCollapsedLists.delete(listKey);
    if (collapsedLists.has(listKey)) {
      collapsedLists.delete(listKey);
    } else {
      collapsedLists.add(listKey);
    }
    saveCollapseState();
  }

  // Toggles a list into or out of half-collapsed state.
  function toggleHalfCollapse(listKey: string): void {
    collapsedLists.delete(listKey);
    if (halfCollapsedLists.has(listKey)) {
      halfCollapsedLists.delete(listKey);
    } else {
      halfCollapsedLists.add(listKey);
    }
    saveCollapseState();
  }

  // Expands a half-collapsed list to full (used by the "Show N more" button).
  function expandFromHalf(listKey: string): void {
    halfCollapsedLists.delete(listKey);
    saveCollapseState();
  }

  // Toggles a list's locked state and persists to board.yaml.
  function toggleLock(listKey: string): void {
    if (lockedLists.has(listKey)) {
      lockedLists.delete(listKey);
    } else {
      lockedLists.add(listKey);
    }

    boardConfig.update(c => {
      if (c[listKey]) {
        c[listKey] = { ...c[listKey], locked: lockedLists.has(listKey) };
      }
      return c;
    });

    SaveLockedLists([...lockedLists]).catch(e => addToast(`Failed to save locked state: ${e}`));
  }

  // Pins a list to the left or right edge of the screen.
  function pinList(listKey: string, side: "left" | "right"): void {
    pinnedLeftLists.delete(listKey);
    pinnedRightLists.delete(listKey);

    if (side === "left") {
      pinnedLeftLists.add(listKey);
    } else {
      pinnedRightLists.add(listKey);
    }

    SavePinnedLists([...pinnedLeftLists], [...pinnedRightLists])
      .catch(e => addToast(`Failed to save pinned state: ${e}`));
  }

  // Unpins a list, returning it to the scrollable area.
  function unpinList(listKey: string): void {
    pinnedLeftLists.delete(listKey);
    pinnedRightLists.delete(listKey);

    SavePinnedLists([...pinnedLeftLists], [...pinnedRightLists])
      .catch(e => addToast(`Failed to save pinned state: ${e}`));
  }

  // Returns the pin state of a list.
  function getPinState(listKey: string): "left" | "right" | null {
    if (pinnedLeftLists.has(listKey)) {
      return "left";
    }
    if (pinnedRightLists.has(listKey)) {
      return "right";
    }
    return null;
  }

  // Document-level dragover handler active only during list drags.
  // Updates ghost position, allows drops everywhere, and auto-scrolls at edges.
  function listDragOverHandler(e: DragEvent): void {
    e.preventDefault();
    dragPos.set({ x: e.clientX, y: e.clientY });
    handleAutoScroll(e);
  }

  // Computed key arrays for the three-panel layout.
  let pinnedLeftKeys = $derived(sortedListKeys($boardData, $listOrder).filter(k => pinnedLeftLists.has(k)));
  let pinnedRightKeys = $derived(sortedListKeys($boardData, $listOrder).filter(k => pinnedRightLists.has(k)));
  let scrollableKeys = $derived(
    sortedListKeys($boardData, $listOrder).filter(
      k => !pinnedLeftLists.has(k) && !pinnedRightLists.has(k),
    ),
  );
  let visualKeys = $derived([...pinnedLeftKeys, ...scrollableKeys, ...pinnedRightKeys]);

  // Begins a list column drag operation and attaches a document-level dragover listener.
  function startListDrag(listKey: string): void {
    if (pinnedLeftLists.has(listKey) || pinnedRightLists.has(listKey)) {
      return;
    }
    listDragging = listKey;
    document.addEventListener('dragover', listDragOverHandler as EventListener);
  }

  // Ends a list column drag operation and removes the document-level listener.
  function endListDrag(): void {
    listDragging = null;
    listDropTarget = null;
    listDropSide = null;
    stopAutoScroll();
    document.removeEventListener('dragover', listDragOverHandler as EventListener);
  }

  // Computes drop position from cursor X relative to scrollable column midpoints.
  function handleListDragOver(e: DragEvent): void {
    if (!listDragging || !boardContainerEl) {
      return;
    }

    const keys = scrollableKeys;
    const columns = boardContainerEl.querySelectorAll('.list-column:not(.pinned-left):not(.pinned-right)');
    if (columns.length === 0) {
      return;
    }

    // Find the insertion edge closest to the cursor.
    // Bias thresholds at edges so first/last positions are easier to hit:
    // first column splits at 2/3, last at 1/3, interior at 1/2.
    let targetKey = keys[0];
    let side: 'left' | 'right' = 'left';
    const last = columns.length - 1;

    for (let i = 0; i < columns.length; i++) {
      const rect = columns[i].getBoundingClientRect();
      const bias = i === 0 ? 0.67 : i === last ? 0.33 : 0.5;
      const splitX = rect.left + rect.width * bias;

      if (e.clientX < splitX) {
        targetKey = keys[i];
        side = 'left';
        break;
      }
      // Past this column's split point -- tentatively place after it
      targetKey = keys[i];
      side = 'right';
    }

    listDropTarget = targetKey;
    listDropSide = side;

    // Position the visual drop line at the chosen edge
    const idx = keys.indexOf(targetKey);
    const col = columns[idx] as HTMLElement;
    const rect = col.getBoundingClientRect();

    dropLineX = side === 'left' ? rect.left - 6 : rect.right + 6;
    dropLineTop = rect.top;
    dropLineHeight = rect.height;
  }

  // Handles dropping a list column, reordering and persisting.
  function handleListDrop(e: DragEvent): void {
    e.preventDefault();

    if (!listDragging || !listDropTarget || listDragging === listDropTarget) {
      endListDrag();
      return;
    }

    const allKeys = sortedListKeys($boardData, $listOrder);
    const srcIdx = allKeys.indexOf(listDragging);
    if (srcIdx === -1) {
      endListDrag();
      return;
    }

    const reordered = [...allKeys];
    reordered.splice(srcIdx, 1);

    // Find target position after source removal, then adjust for drop side.
    let insertIdx = reordered.indexOf(listDropTarget);
    if (insertIdx === -1) {
      endListDrag();
      return;
    }

    if (listDropSide === 'right') {
      insertIdx += 1;
    }
    reordered.splice(insertIdx, 0, listDragging);

    listOrder.set(reordered);
    SaveListOrder(reordered).catch(err => addToast(`Failed to save list order: ${err}`));
    endListDrag();
  }

  // Deletes a list after confirmation, clearing any related selection state.
  async function deleteList(listKey: string): Promise<void> {
    if ($selectedCard?.filePath.includes('/' + listKey + '/')) {
      selectedCard.set(null);
    }
    if ($focusedCard?.listKey === listKey) {
      focusedCard.set(null);
    }

    try {
      await DeleteList(listKey);
      await initBoard();
    } catch (err) {
      addToast(`Failed to delete list: ${err}`);
    }
  }

  // Loads the board from the backend and unpacks lists + config into separate stores.
  async function initBoard(): Promise<void> {
    error = "";
    try {
      const response = await LoadBoard("");
      boardData.set(response.lists);
      boardPath.set(response.boardPath || "");

      if (response.boardPath) {
        WindowSetTitle(`Daedalus - ${response.boardPath}`);
      }

      // Unpack ListEntry array into stores.
      const entries: daedalus.ListEntry[] = response.config?.lists || [];
      listOrder.set(entries.map((e) => e.dir));

      const configMap: Record<string, { title: string; limit: number; locked: boolean }> = {};
      for (const entry of entries) {
        configMap[entry.dir] = {
          title: entry.title || '',
          limit: entry.limit || 0,
          locked: entry.locked || false,
        };
      }
      boardConfig.set(configMap);

      collapsedLists = new SvelteSet(entries.filter((e) => e.collapsed).map((e) => e.dir));
      halfCollapsedLists = new SvelteSet(entries.filter((e) => e.halfCollapsed).map((e) => e.dir));
      lockedLists = new SvelteSet(entries.filter((e) => e.locked).map((e) => e.dir));
      pinnedLeftLists = new SvelteSet(entries.filter((e) => e.pinned === "left").map((e) => e.dir));
      pinnedRightLists = new SvelteSet(entries.filter((e) => e.pinned === "right").map((e) => e.dir));

      boardTitle.set(response.config?.title || "Daedalus");
      labelColors.set(response.config?.labelColors || {});

      if (response.config?.labelsExpanded !== undefined && response.config.labelsExpanded !== null) {
        labelsExpanded.set(response.config.labelsExpanded);
      }
      if (response.config?.showYearProgress !== undefined && response.config.showYearProgress !== null) {
        showYearProgress = response.config.showYearProgress;
      }
      if (response.config?.darkMode !== undefined && response.config.darkMode !== null) {
        darkMode = response.config.darkMode;
      }
      document.documentElement.classList.toggle("light", !darkMode);
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
    if (!boardContainerEl || pinnedLeftLists.has(listKey) || pinnedRightLists.has(listKey)) {
      return;
    }
    const col = boardContainerEl.querySelector(`[data-list-key="${listKey}"]`);
    if (col) {
      col.scrollIntoView({ block: 'nearest', inline: 'nearest' });
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

  // Dispatches global keyboard shortcuts to the extracted handler with current state.
  function handleGlobalKeydown(e: KeyboardEvent): void {
    handleBoardKeydown(e, {
      showAbout,
      showLabelEditor,
      showKeyboardHelp,
      draftListKey: $draftListKey,
      selectedCard: $selectedCard,
      focusedCard: $focusedCard,
      confirmingFocusDelete,
      boardData: $boardData,
      sortedKeys: visualKeys,
      collapsedLists,
      halfCollapsedLists,
    }, {
      setShowAbout: v => { showAbout = v; },
      setShowLabelEditor: v => { showLabelEditor = v; },
      setShowKeyboardHelp: v => { showKeyboardHelp = v; },
      setConfirmingFocusDelete: v => { confirmingFocusDelete = v; },
      setFocusedCard: v => focusedCard.set(v),
      openCard: card => selectedCard.set(card),
      openCardEdit: card => { openInEditMode.set(true); selectedCard.set(card); },
      openSearch,
      createCard,
      deleteFocusedCard,
      scrollListIntoView,
    });
  }

  // Clean up indicators, drop state, and auto-scroll when drag ends (drop or cancel).
  $effect(() => {
    if (!$dragState) {
      clearDropIndicators();
      dropTarget.set(null);
      stopAutoScroll();
    }
  });

  // Set keyboard focus to match the selected card so arrow nav works after closing the modal.
  $effect(() => {
    if ($selectedCard) {
      for (const listKey of Object.keys($boardData)) {
        const idx = $boardData[listKey].findIndex(c => c.filePath === $selectedCard!.filePath);
        if (idx !== -1) {
          focusedCard.set({ listKey, cardIndex: idx });
          break;
        }
      }
    } else {
      focusedCard.set(null);
    }
  });

  // Sync the board container DOM ref to the drag module for horizontal auto-scroll.
  $effect(() => {
    setBoardContainer(boardContainerEl);
  });

  // Auto-open a card when search query is exactly #<digits> and a unique match exists.
  $effect(() => {
    const q = $searchQuery.trim();
    if (!/^#\d+$/.test(q)) {
      return;
    }
    const targetId = Number(q.slice(1));

    let found: daedalus.KanbanCard | null = null;
    for (const key of Object.keys($boardData)) {
      for (const card of $boardData[key]) {
        if (card.metadata.id === targetId) {
          found = card;
          break;
        }
      }
      if (found) {
        break;
      }
    }
    if (found) {
      selectedCard.set(found);
      requestAnimationFrame(() => {
        searchQuery.set("");
        searchOpen = false;
      });
    }
  });

  // Tracks the drag ghost position globally via capture phase so stopPropagation cannot block it.
  function handleGlobalDragOver(e: DragEvent): void {
    if (get(dragState)) {
      dragPos.set({ x: e.clientX, y: e.clientY });
    }
  }

  onMount(() => {
    document.addEventListener("dragover", handleGlobalDragOver, true);
    initBoard();
  });

  onDestroy(() => {
    stopAutoScroll();
    document.removeEventListener("dragover", handleGlobalDragOver, true);
    document.removeEventListener('dragover', listDragOverHandler as EventListener);
  });

</script>

<svelte:window onkeydown={handleGlobalKeydown} />
<svelte:document onselectstart={(e) => {
  const target = e.target as HTMLElement;
  const tag = target.tagName;

  if (tag === "INPUT" || tag === "TEXTAREA" || target.isContentEditable) {
    return;
  }
  if (window.getComputedStyle(target).userSelect !== "none") {
    return;
  }
  e.preventDefault();
}} />

<main>
  <TopBar bind:searchOpen bind:showYearProgress bind:darkMode
    bind:showLabelEditor bind:showKeyboardHelp bind:showAbout
    oninitboard={initBoard}
  />

  {#snippet renderList(listKey: string)}
    {#if collapsedLists.has(listKey)}
      <div class="list-column collapsed" data-list-key={listKey}
        role="button" tabindex="0" title="Expand list"
        class:pinned-left={getPinState(listKey) === 'left'}
        class:pinned-right={getPinState(listKey) === 'right'}
        class:list-dragging={listDragging === listKey}
        onclick={() => toggleFullCollapse(listKey)}
        onkeydown={e => e.key === 'Enter' && toggleFullCollapse(listKey)}
      >
        <span class="collapsed-count">{getCountDisplay(listKey, $boardData, $boardConfig)}</span>
        <span class="collapsed-title">{getDisplayTitle(listKey, $boardConfig)}</span>
      </div>
    {:else if halfCollapsedLists.has(listKey)}
      {@const allItems = $filteredBoardData[listKey] || []}
      {@const visibleItems = allItems.slice(0, HALF_COLLAPSED_CARD_LIMIT)}
      {@const remaining = allItems.length - HALF_COLLAPSED_CARD_LIMIT}
      <div class="list-column half-collapsed" data-list-key={listKey}
        role="group"
        class:pinned-left={getPinState(listKey) === 'left'}
        class:pinned-right={getPinState(listKey) === 'right'}
        class:list-dragging={listDragging === listKey}
      >
        <ListHeader {listKey} locked={lockedLists.has(listKey)}
          pinState={getPinState(listKey)}
          hasLeftPin={pinnedLeftLists.size > 0}
          hasRightPin={pinnedRightLists.size > 0}
          isLastList={listKey === visualKeys[visualKeys.length - 1]}
          oncreatecard={() => createCard(listKey)}
          onfullcollapse={() => toggleFullCollapse(listKey)}
          onhalfcollapse={() => toggleHalfCollapse(listKey)}
          onlock={() => toggleLock(listKey)}
          onpinleft={() => pinList(listKey, "left")}
          onpinright={() => pinList(listKey, "right")}
          onunpin={() => unpinList(listKey)}
          onreload={initBoard}
          onlistdragstart={() => startListDrag(listKey)}
          onlistdragend={endListDrag}
          ondelete={() => deleteList(listKey)}
        />
        <div class="list-body half-collapsed-body" role="list">
          <VirtualList items={visibleItems} component={Card} estimatedHeight={90} listKey={listKey}
            focusIndex={$focusedCard?.listKey === listKey ? $focusedCard.cardIndex : -1}
          />
        </div>
        {#if remaining > 0}
          <button class="show-more-bar" title="Expand to show all cards" onclick={() => expandFromHalf(listKey)}>
            Show {remaining} more
          </button>
        {/if}
      </div>
    {:else}
      {@const locked = lockedLists.has(listKey)}
      <div class="list-column" data-list-key={listKey} role="group"
        class:pinned-left={getPinState(listKey) === 'left'}
        class:pinned-right={getPinState(listKey) === 'right'}
        class:list-full={$dragState
          && $dragState.sourceListKey !== listKey
          && isAtLimit(listKey, $boardData, $boardConfig)}
        class:list-locked={$dragState && locked}
        class:list-dragging={listDragging === listKey}
      >
        <ListHeader {listKey} {locked}
          pinState={getPinState(listKey)}
          hasLeftPin={pinnedLeftLists.size > 0}
          hasRightPin={pinnedRightLists.size > 0}
          isLastList={listKey === visualKeys[visualKeys.length - 1]}
          oncreatecard={() => createCard(listKey)}
          onfullcollapse={() => toggleFullCollapse(listKey)}
          onhalfcollapse={() => toggleHalfCollapse(listKey)}
          onlock={() => toggleLock(listKey)}
          onpinleft={() => pinList(listKey, "left")}
          onpinright={() => pinList(listKey, "right")}
          onunpin={() => unpinList(listKey)}
          onreload={initBoard}
          onlistdragstart={() => startListDrag(listKey)}
          onlistdragend={endListDrag}
          ondelete={() => deleteList(listKey)}
        />
        <div class="list-body" role="list"
          ondragenter={handleDragEnter}
          ondragover={(e) => handleDragOver(e, listKey)}
          ondragleave={handleDragLeave}
          ondrop={(e) => handleDrop(e, listKey, initBoard)}
        >
          <VirtualList items={$filteredBoardData[listKey]} component={Card} estimatedHeight={90}
            listKey={listKey} focusIndex={$focusedCard?.listKey === listKey ? $focusedCard.cardIndex : -1}
          />
        </div>
        <button class="list-footer-add" title="Add card to bottom"
          onclick={() => createCardBottom(listKey)}
          ondragenter={handleDragEnter}
          ondragover={(e) => handleFooterDragOver(e, listKey)}
          ondrop={(e) => handleDrop(e, listKey, initBoard)}
        >
          <Icon name="plus" size={12} />
        </button>
      </div>
    {/if}
  {/snippet}

  {#if error}
    <div class="error">{error}</div>
  {:else if $isLoaded}
    <div class="board-container" role="group" class:modal-open={$selectedCard || $draftListKey}
      bind:this={boardContainerEl}
      ondragover={(e) => { if (listDragging) { handleListDragOver(e); } }}
      ondrop={(e) => { if (listDragging) { handleListDrop(e); } }}
    >
      {#each visualKeys as listKey}
        {@render renderList(listKey)}
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
            <span class="ghost-label" style="background: {labelColor(label, $labelColors)}"></span>
          {/each}
        </div>
      {/if}
      <div class="ghost-title">{$dragState.card.metadata.title}</div>
    </div>
  {/if}

  {#if listDragging}
    <div class="drag-ghost list-drag-ghost" style="left: {$dragPos.x + 10}px; top: {$dragPos.y - 20}px">
      {getDisplayTitle(listDragging, $boardConfig)}
    </div>
    {#if listDropTarget}
      <div class="list-drop-line" style="left: {dropLineX}px; top: {dropLineTop}px; height: {dropLineHeight}px"></div>
    {/if}
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
  {#if showAbout}
    <About onclose={() => showAbout = false} />
  {/if}
  {#if showLabelEditor}
    <LabelColorEditor onclose={() => showLabelEditor = false} onreload={initBoard} />
  {/if}

</main>

<style lang="scss">

  :global(body) {
    margin: 0;
    background-color: var(--color-bg-base);
    color: var(--color-text-primary);
    font-family: sans-serif;
    overflow: hidden;
    user-select: none;
  }

  :global(textarea),
  :global(input),
  :global([contenteditable]) {
    user-select: text;
  }

  main {
    display: flex;
    flex-direction: column;
    height: 100vh;
  }

  .board-container {
    flex: 1;
    display: flex;
    overflow-x: auto;
    padding: 10px 0;
    gap: 10px;

    &::before,
    &::after {
      content: '';
      flex-shrink: 0;
      width: 10px;
    }
  }

  .list-column {
    flex: 0 0 280px;
    display: flex;
    flex-direction: column;
    background: var(--color-bg-list);
    border-radius: 8px;
    max-height: 100%;
    border: 1px solid var(--color-border-medium);
    position: relative;
    z-index: 1;

    &:has(:global(.header-menu)) {
      z-index: 10;
    }

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

    &.list-locked {
      opacity: 0.5;
      pointer-events: auto;
    }

    &.list-dragging {
      opacity: 0.4;
    }

    &.pinned-left,
    &.pinned-right {
      position: sticky;
      z-index: 2;
    }

    &.pinned-left {
      left: 0;
      box-shadow: 4px 0 8px rgba(0, 0, 0, 0.25);
    }

    &.pinned-right {
      right: 0;
      box-shadow: -4px 0 8px rgba(0, 0, 0, 0.25);
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

  .half-collapsed-body {
    max-height: 470px;
  }

  .show-more-bar {
    all: unset;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    padding: 6px 0;
    cursor: pointer;
    color: var(--color-text-muted);
    font-size: 0.78rem;
    font-weight: 500;
    border-top: 1px solid var(--color-border-medium);
    box-sizing: border-box;

    &:hover {
      color: var(--color-text-secondary);
      background: var(--overlay-hover-faint);
    }
  }

  .list-body {
    flex: 1;
    overflow: hidden;
    padding-top: 12px;
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

  .list-drag-ghost {
    font-weight: 600;
    font-size: 0.85rem;
  }

  .list-drop-line {
    position: fixed;
    width: 3px;
    background: var(--color-accent);
    border-radius: 2px;
    z-index: 1;
    pointer-events: none;
  }

</style>
