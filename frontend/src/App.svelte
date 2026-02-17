<script lang="ts">
  // Main board view. Renders list columns, handles collapse/pin/lock state, and coordinates modals.

  import { onMount, onDestroy } from "svelte";
  import { SvelteSet } from "svelte/reactivity";
  import { WindowSetTitle, EventsOn, EventsOff } from "../wailsjs/runtime/runtime";
  import {
    LoadBoard, SaveCollapsedLists, SaveHalfCollapsedLists, SaveLockedLists,
    SavePinnedLists, DeleteList, DeleteAllCards, CreateList, MoveAllCards,
    SaveLabelColors, SaveMinimalView, SaveListOrder, SaveZoom,
  } from "../wailsjs/go/main/App";
  import { get } from "svelte/store";
  import {
    boardData, boardTitle, boardConfig, boardPath, sortedListKeys, isLoaded,
    selectedCard, draftListKey, draftPosition,
    labelsExpanded, minimalView, labelColors, dragState, dropTarget, focusedCard, openInEditMode,
    removeCardFromBoard, addToast, saveWithToast, isAtLimit, isLocked, listOrder, loadProfile,
    searchQuery, filteredBoardData,
  } from "./stores/board";
  import type { daedalus } from "../wailsjs/go/models";
  import { labelColor, getDisplayTitle, getCountDisplay, HALF_COLLAPSED_CARD_LIMIT } from "./lib/utils";
  import { handleBoardKeydown } from "./lib/keyboard";
  import {
    dragPos, setBoardContainer, clearDropIndicators,
    handleDragEnter, handleDragOver, handleDragLeave, handleDrop,
    handleFooterDragOver, stopAutoScroll,
  } from "./lib/drag";
  import {
    listDragging, listDropTarget, dropLineX, dropLineTop, dropLineHeight,
    startListDrag, endListDrag, computeListDragOver, handleListDrop, cleanupListDrag,
  } from "./lib/listDrag";
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
  import IconManager from "./components/IconManager.svelte";
  import BoardStats from "./components/BoardStats.svelte";
  import Scratchpad from "./components/Scratchpad.svelte";
  import TopBar from "./components/TopBar.svelte";
  import Icon from "./components/Icon.svelte";
  import CardContextMenu from "./components/CardContextMenu.svelte";

  let error = $state("");
  let collapsedLists = $state(new SvelteSet<string>());
  let halfCollapsedLists = $state(new SvelteSet<string>());
  let lockedLists = $state(new SvelteSet<string>());
  let pinnedLeftLists = $state(new SvelteSet<string>());
  let pinnedRightLists = $state(new SvelteSet<string>());
  let showKeyboardHelp = $state(false);
  let showAbout = $state(false);
  let showLabelEditor = $state(false);
  let showIconManager = $state(false);
  let showBoardStats = $state(false);
  let showScratchpad = $state(false);
  let showYearProgress = $state(false);
  let darkMode = $state(true);
  let zoomLevel = $state(1.0);

  let boardContainerEl: HTMLDivElement | undefined = $state(undefined);
  let firstLoad = true;
  let searchOpen = $state(false);
  let addingList = $state<'left' | 'right' | false>(false);
  let newListName = $state("");

  // Opens the search bar with an optional prefill string (used by keyboard handler).
  function openSearch(prefill: string = ""): void {
    if (prefill) {
      searchQuery.set(prefill);
    }
    searchOpen = true;
  }

  // Persists both collapse states to board.yaml.
  function saveCollapseState(): void {
    saveWithToast(SaveCollapsedLists([...collapsedLists]), "save collapsed state");
    saveWithToast(SaveHalfCollapsedLists([...halfCollapsedLists]), "save half-collapsed state");
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

    saveWithToast(SaveLockedLists([...lockedLists]), "save locked state");
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

    saveWithToast(SavePinnedLists([...pinnedLeftLists], [...pinnedRightLists]), "save pinned state");
  }

  // Unpins a list, returning it to the scrollable area.
  function unpinList(listKey: string): void {
    pinnedLeftLists.delete(listKey);
    pinnedRightLists.delete(listKey);

    saveWithToast(SavePinnedLists([...pinnedLeftLists], [...pinnedRightLists]), "save pinned state");
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

  // Computed key arrays for the three-panel layout.
  let pinnedLeftKeys = $derived(sortedListKeys($boardData, $listOrder).filter(k => pinnedLeftLists.has(k)));
  let pinnedRightKeys = $derived(sortedListKeys($boardData, $listOrder).filter(k => pinnedRightLists.has(k)));
  let scrollableKeys = $derived(
    sortedListKeys($boardData, $listOrder).filter(
      k => !pinnedLeftLists.has(k) && !pinnedRightLists.has(k),
    ),
  );
  let visualKeys = $derived([...pinnedLeftKeys, ...scrollableKeys, ...pinnedRightKeys]);

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

  // Moves all cards from one list to another via the backend, then updates the store.
  async function moveAllCards(sourceKey: string, targetKey: string): Promise<void> {
    try {
      await MoveAllCards(sourceKey, targetKey);

      boardData.update(lists => {
        const moved = lists[sourceKey] || [];
        lists[targetKey] = [...(lists[targetKey] || []), ...moved];
        lists[sourceKey] = [];
        return lists;
      });
      addToast(`Moved all cards to ${getDisplayTitle(targetKey, $boardConfig)}`, "success");

    } catch (err) {
      addToast(`Failed to move cards: ${err}`);
      const response = await LoadBoard("");
      boardData.update(() => response.lists);
    }
  }

  // Deletes all cards in a list via the backend, then clears the store.
  async function deleteAllCards(listKey: string): Promise<void> {
    try {
      await DeleteAllCards(listKey);

      boardData.update(lists => {
        lists[listKey] = [];
        return lists;
      });
      addToast(`Deleted all cards from ${getDisplayTitle(listKey, $boardConfig)}`, "success");

    } catch (err) {
      addToast(`Failed to delete cards: ${err}`);
      const response = await LoadBoard("");
      boardData.update(() => response.lists);
    }
  }

  // Submits the new list name to the backend, reloads the board on success.
  async function submitNewList(): Promise<void> {
    const name = newListName.trim();
    if (!name) {
      addingList = false;
      newListName = "";
      return;
    }

    const side = addingList;
    try {
      await CreateList(name);
      addingList = false;
      newListName = "";
      await initBoard();

      // If added from the left button, move the new list to the front of the order.
      if (side === 'left') {
        const order = [...$listOrder];
        const idx = order.indexOf(name);

        if (idx > 0) {
          order.splice(idx, 1);
          order.unshift(name);
          listOrder.set(order);
          saveWithToast(SaveListOrder(order), "save list order");
        }
      }
      addToast(`List "${name}" created`, "success");

    } catch (err) {
      addToast(`Failed to create list: ${err}`);
    }
  }

  // Loads the board from the backend and unpacks lists + config into separate stores.
  async function initBoard(): Promise<void> {
    error = "";
    try {
      const t0 = performance.now();
      const response = await LoadBoard("");
      const roundTripMs = performance.now() - t0;
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

      // Backfill: register any card labels not yet in the label_colors registry
      const loadedColors: Record<string, string> = response.config?.labelColors || {};
      let backfillDirty = false;
      for (const cards of Object.values(response.lists)) {
        for (const card of cards) {
          if (card.metadata.labels) {
            for (const label of card.metadata.labels) {
              if (!loadedColors[label]) {
                loadedColors[label] = labelColor(label);
                backfillDirty = true;
              }
            }
          }
        }
      }
      labelColors.set(loadedColors);

      if (backfillDirty) {
        SaveLabelColors(loadedColors).catch(e => console.error("Failed to backfill label colors:", e));
      }

      if (response.config?.labelsExpanded !== undefined && response.config.labelsExpanded !== null) {
        labelsExpanded.set(response.config.labelsExpanded);
      }
      if (response.config?.minimalView !== undefined && response.config.minimalView !== null) {
        minimalView.set(response.config.minimalView);
      }
      if (response.config?.showYearProgress !== undefined && response.config.showYearProgress !== null) {
        showYearProgress = response.config.showYearProgress;
      }
      if (response.config?.darkMode !== undefined && response.config.darkMode !== null) {
        darkMode = response.config.darkMode;
      }
      if (response.config?.zoom !== undefined && response.config.zoom !== null) {
        zoomLevel = response.config.zoom;
      }
      document.documentElement.classList.toggle("light", !darkMode);

      const p = response.profile;
      loadProfile.set({
        configMs: p?.configMs ?? 0,
        scanMs: p?.scanMs ?? 0,
        mergeMs: p?.mergeMs ?? 0,
        totalMs: p?.totalMs ?? 0,
        transferMs: Math.max(0, roundTripMs - (p?.totalMs ?? 0)),
      });

      isLoaded.set(true);

      // On first load, scroll past the left add-list button so it's out of view.
      if (firstLoad) {
        firstLoad = false;
        requestAnimationFrame(() => {
          requestAnimationFrame(() => {
            if (boardContainerEl) {
              boardContainerEl.scrollLeft = 60;
            }
          });
        });
      }

    } catch (e) {
      error = (e as Error).toString();
      addToast(`Failed to load board: ${error}`);
    }
  }

  // Opens the draft-creation modal for the first (leftmost) list.
  function createCardDefault(): void {
    const keys = sortedListKeys($boardData, $listOrder);
    if (keys.length === 0) {
      addToast("No lists on this board");
      return;
    }
    createCard(keys[0]);
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

  // Zoom functions -- clamp between 0.5 and 1.5, step by 0.1.
  function zoomIn(): void {
    zoomLevel = Math.min(1.5, Math.round((zoomLevel + 0.1) * 10) / 10);
    saveWithToast(SaveZoom(zoomLevel), "save zoom level");
  }

  function zoomOut(): void {
    zoomLevel = Math.max(0.5, Math.round((zoomLevel - 0.1) * 10) / 10);
    saveWithToast(SaveZoom(zoomLevel), "save zoom level");
  }

  function zoomReset(): void {
    zoomLevel = 1.0;
    saveWithToast(SaveZoom(zoomLevel), "save zoom level");
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
      boardData: $boardData,
      sortedKeys: visualKeys,
      collapsedLists,
      halfCollapsedLists,
    }, {
      setShowAbout: v => { showAbout = v; },
      setShowLabelEditor: v => { showLabelEditor = v; },
      setShowKeyboardHelp: v => { showKeyboardHelp = v; },
      setFocusedCard: v => focusedCard.set(v),
      openCard: card => selectedCard.set(card),
      openCardEdit: card => { openInEditMode.set(true); selectedCard.set(card); },
      openSearch,
      createCard,
      createCardDefault,
      toggleMinimalView: () => {
        minimalView.update(v => {
          const next = !v;
          saveWithToast(SaveMinimalView(next), "save minimal view");
          return next;
        });
      },
      zoomIn,
      zoomOut,
      zoomReset,
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

  // Svelte action to auto-focus an input element and scroll the board to show it.
  function autofocus(node: HTMLInputElement): void {
    node.focus();
    // Double rAF: first frame inserts into layout, second frame has final scrollWidth.
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        if (boardContainerEl) {
          boardContainerEl.scrollLeft = addingList === 'left' ? 0 : boardContainerEl.scrollWidth;
        }
      });
    });
  }

  onMount(() => {
    document.addEventListener("dragover", handleGlobalDragOver, true);
    initBoard();

    // Reload board when the Go file watcher detects external changes.
    EventsOn("board:reload", () => initBoard());
  });

  onDestroy(() => {
    stopAutoScroll();
    document.removeEventListener("dragover", handleGlobalDragOver, true);
    cleanupListDrag();
    EventsOff("board:reload");
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
    bind:showLabelEditor bind:showIconManager bind:showScratchpad bind:showBoardStats bind:showKeyboardHelp bind:showAbout
    {zoomLevel} onzoomin={zoomIn} onzoomout={zoomOut} onzoomreset={zoomReset}
    oncreatecard={createCardDefault}
  />

  {#snippet renderList(listKey: string)}
    {#if collapsedLists.has(listKey)}
      <div class="list-column collapsed" data-list-key={listKey}
        role="button" tabindex="0" title="Expand list"
        class:pinned-left={getPinState(listKey) === 'left'}
        class:pinned-right={getPinState(listKey) === 'right'}
        class:list-dragging={$listDragging === listKey}
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
        class:list-dragging={$listDragging === listKey}
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
          onlistdragstart={() => startListDrag(listKey, !!getPinState(listKey))}
          onlistdragend={endListDrag}
          onmoveallcards={(target) => moveAllCards(listKey, target)}
          ondeleteallcards={() => deleteAllCards(listKey)}
          ondelete={() => deleteList(listKey)}
        />
        <div class="list-body half-collapsed-body" role="list">
          <VirtualList items={visibleItems} component={Card} estimatedHeight={$minimalView ? 28 : 90} listKey={listKey}
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
        class:list-dragging={$listDragging === listKey}
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
          onlistdragstart={() => startListDrag(listKey, !!getPinState(listKey))}
          onlistdragend={endListDrag}
          onmoveallcards={(target) => moveAllCards(listKey, target)}
          ondeleteallcards={() => deleteAllCards(listKey)}
          ondelete={() => deleteList(listKey)}
        />
        <div class="list-body" role="list"
          ondragenter={handleDragEnter}
          ondragover={(e) => handleDragOver(e, listKey)}
          ondragleave={handleDragLeave}
          ondrop={(e) => handleDrop(e, listKey, initBoard)}
        >
          <VirtualList items={$filteredBoardData[listKey]} component={Card} estimatedHeight={$minimalView ? 28 : 90}
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
      style="zoom: {zoomLevel}"
      bind:this={boardContainerEl}
      ondragover={(e) => { if ($listDragging && boardContainerEl) { computeListDragOver(e, boardContainerEl, scrollableKeys); } }}
      ondrop={(e) => { if ($listDragging) { handleListDrop(e); } }}
    >
      {#if addingList === 'left'}
        <div class="add-list-input">
          <input type="text" placeholder="List name..."
            bind:value={newListName}
            use:autofocus
            onkeydown={(e) => {
              if (e.key === 'Enter') {
                submitNewList();
              }
              if (e.key === 'Escape') {
                addingList = false;
                newListName = "";
              }
            }}
            onblur={() => submitNewList()}
          />
        </div>
      {:else}
        <button class="add-list-btn" title="Add new list to start" onclick={() => { addingList = 'left'; }}>
          <Icon name="plus" size={18} />
        </button>
      {/if}

      {#each visualKeys as listKey}
        {@render renderList(listKey)}
      {/each}

      {#if addingList === 'right'}
        <div class="add-list-input">
          <input type="text" placeholder="List name..."
            bind:value={newListName}
            use:autofocus
            onkeydown={(e) => {
              if (e.key === 'Enter') {
                submitNewList();
              }
              if (e.key === 'Escape') {
                addingList = false;
                newListName = "";
              }
            }}
            onblur={() => submitNewList()}
          />
        </div>
      {:else}
        <button class="add-list-btn" title="Add new list to end" onclick={() => { addingList = 'right'; }}>
          <Icon name="plus" size={18} />
        </button>
      {/if}
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

  {#if $listDragging}
    <div class="drag-ghost list-drag-ghost" style="left: {$dragPos.x + 10}px; top: {$dragPos.y - 20}px">
      {getDisplayTitle($listDragging, $boardConfig)}
    </div>
    {#if $listDropTarget}
      <div class="list-drop-line" style="left: {$dropLineX}px; top: {$dropLineTop}px; height: {$dropLineHeight}px"></div>
    {/if}
  {/if}

  <DraftCard />
  <CardDetail />
  <Metrics />
  <Toast />
  <CardContextMenu />

  {#if showKeyboardHelp}
    <KeyboardHelp onclose={() => showKeyboardHelp = false} />
  {/if}
  {#if showAbout}
    <About onclose={() => showAbout = false} />
  {/if}
  {#if showLabelEditor}
    <LabelColorEditor onclose={() => showLabelEditor = false} onreload={initBoard} />
  {/if}
  {#if showIconManager}
    <IconManager onclose={() => showIconManager = false} onreload={initBoard} />
  {/if}
  {#if showBoardStats}
    <BoardStats onclose={() => showBoardStats = false} />
  {/if}
  {#if showScratchpad}
    <Scratchpad onclose={() => showScratchpad = false} />
  {/if}

</main>

<style lang="scss">

  :global(body) {
    margin: 0;
    background-color: var(--color-bg-base);
    color: var(--color-text-primary);
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
    z-index: var(--z-board);

    &:has(:global(.header-menu)) {
      z-index: var(--z-list-menu-parent);
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
      z-index: var(--z-board-raised);
    }

    &.pinned-left {
      left: 0;
      box-shadow: var(--shadow-side);
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
    z-index: var(--z-board-raised);
  }

  :global(.item-slot.drop-below::after) {
    content: '';
    position: absolute;
    bottom: 6px;
    left: 6px;
    right: 6px;
    height: 3px;
    background: var(--color-accent);
    z-index: var(--z-board-raised);
  }

  :global(.item-slot.drop-above),
  :global(.item-slot.drop-below) {
    position: relative;
    z-index: var(--z-board);
  }

  .drag-ghost {
    position: fixed;
    width: 250px;
    background: var(--color-bg-surface);
    border-radius: 4px;
    padding: 8px 10px;
    color: var(--color-text-primary);
    pointer-events: none;
    z-index: var(--z-drag-ghost);
    opacity: 0.9;
    box-shadow: var(--shadow-lg);
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
    z-index: var(--z-board);
    pointer-events: none;
  }

  .add-list-btn {
    flex: 0 0 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: 2px dashed var(--color-border-medium);
    border-radius: 8px;
    cursor: pointer;
    color: var(--color-text-muted);
    transition: border-color 0.15s, color 0.15s, background 0.15s;
    align-self: stretch;

    &:hover {
      border-color: var(--color-text-secondary);
      color: var(--color-text-secondary);
      background: var(--overlay-hover-faint);
    }
  }

  .add-list-input {
    flex: 0 0 280px;
    display: flex;
    align-items: flex-start;
    padding-top: 6px;

    input {
      width: 100%;
      padding: 8px 10px;
      border: 1px solid var(--color-accent);
      border-radius: 6px;
      background: var(--color-bg-list);
      color: var(--color-text-primary);
      font-size: 0.9rem;
      outline: none;
      box-sizing: border-box;
    }
  }

</style>
