<script lang="ts">
  // Main board view. Renders list columns, handles collapse/pin/lock state, and coordinates modals.

  import { onMount, onDestroy } from "svelte";
  import { SvelteSet } from "svelte/reactivity";
  import { WindowSetTitle, EventsOn, EventsOff } from "../wailsjs/runtime/runtime";
  import {
    LoadBoard, SaveCollapsedLists, SaveHalfCollapsedLists, SaveLockedLists,
    SavePinnedLists, DeleteList, DeleteAllCards, MoveAllCards,
    SaveLabelColors, SaveListOrder, SaveZoom, GetAppConfig,
  } from "../wailsjs/go/main/App";
  import { get } from "svelte/store";
  import {
    boardData, boardTitle, boardConfig, boardPath, sortedListKeys, isLoaded,
    selectedCard, draftListKey, draftPosition,
    labelsExpanded, minimalView, labelColors, templates, dragState, dropTarget, focusedCard, openInEditMode,
    addToast, saveWithToast, isAtLimit, isLocked, listOrder, loadProfile, toggleMinimalView,
    searchQuery, filteredBoardData, maxCardId,
  } from "./stores/board";
  import type { daedalus } from "../wailsjs/go/models";
  import { main } from "../wailsjs/go/models";
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
  import TemplateManager from "./components/TemplateManager.svelte";
  import CardIcon from "./components/CardIcon.svelte";
  import BoardStats from "./components/BoardStats.svelte";
  import Scratchpad from "./components/Scratchpad.svelte";
  import NewListModal from "./components/NewListModal.svelte";
  import TopBar from "./components/TopBar.svelte";
  import Icon from "./components/Icon.svelte";
  import CardContextMenu from "./components/CardContextMenu.svelte";
  import WelcomeModal from "./components/WelcomeModal.svelte";

  let collapsedLists = $state(new SvelteSet<string>());
  let halfCollapsedLists = $state(new SvelteSet<string>());
  let lockedLists = $state(new SvelteSet<string>());
  let pinnedLeftLists = $state(new SvelteSet<string>());
  let pinnedRightLists = $state(new SvelteSet<string>());
  type ActiveModal = null | 'keyboardHelp' | 'about' | 'labelEditor' | 'iconManager'
    | 'templateManager' | 'boardStats' | 'scratchpad' | 'newList' | 'welcome';
  let activeModal: ActiveModal = $state(null);
  let showKeyboardHelp = $derived(activeModal === 'keyboardHelp');
  let showAbout = $derived(activeModal === 'about');
  let showLabelEditor = $derived(activeModal === 'labelEditor');
  let showIconManager = $derived(activeModal === 'iconManager');
  let showTemplateManager = $derived(activeModal === 'templateManager');
  let showBoardStats = $derived(activeModal === 'boardStats');
  let showScratchpad = $derived(activeModal === 'scratchpad');
  let showNewList = $derived(activeModal === 'newList');
  let showWelcome = $derived(activeModal === 'welcome');
  let showYearProgress = $state(false);
  let darkMode = $state(true);
  let zoomLevel = $state(1.0);

  let boardContainerEl: HTMLDivElement | undefined = $state(undefined);
  let searchOpen = $state(false);

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
    if ($selectedCard?.filePath.includes(listKey) && new RegExp('[/\\\\]' + listKey + '[/\\\\]').test($selectedCard.filePath)) {
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
        lists[targetKey] = [...moved, ...(lists[targetKey] || [])];
        lists[sourceKey] = [];
        return lists;
      });
      addToast(`Moved all cards to ${getDisplayTitle(targetKey, $boardConfig)}`, "success");

    } catch (err) {
      addToast(`Failed to move cards: ${err}`);
      const response = await LoadBoard($boardPath);
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
      const response = await LoadBoard($boardPath);
      boardData.update(() => response.lists);
    }
  }

  // Unpacks ListEntry array into list order, board config, and collapse/lock/pin sets.
  function unpackListConfig(entries: daedalus.ListEntry[]): void {
    listOrder.set(entries.map((e) => e.dir));

    const configMap: Record<string, { title: string; limit: number; locked: boolean; color: string; icon: string }> = {};
    for (const entry of entries) {
      configMap[entry.dir] = {
        title: entry.title || '',
        limit: entry.limit || 0,
        locked: entry.locked || false,
        color: entry.color || '',
        icon: entry.icon || '',
      };
    }
    boardConfig.set(configMap);

    collapsedLists = new SvelteSet(entries.filter((e) => e.collapsed).map((e) => e.dir));
    halfCollapsedLists = new SvelteSet(entries.filter((e) => e.halfCollapsed).map((e) => e.dir));
    lockedLists = new SvelteSet(entries.filter((e) => e.locked).map((e) => e.dir));
    pinnedLeftLists = new SvelteSet(entries.filter((e) => e.pinned === "left").map((e) => e.dir));
    pinnedRightLists = new SvelteSet(entries.filter((e) => e.pinned === "right").map((e) => e.dir));
  }

  // Scans cards for labels not yet in the color registry, backfills with deterministic colors, and saves if dirty.
  function backfillLabelColors(lists: Record<string, daedalus.KanbanCard[]>, loadedColors: Record<string, string>): void {
    let dirty = false;
    for (const cards of Object.values(lists)) {
      for (const card of cards) {
        if (card.metadata.labels) {
          for (const label of card.metadata.labels) {
            if (!loadedColors[label]) {
              loadedColors[label] = labelColor(label);
              dirty = true;
            }
          }
        }
      }
    }
    labelColors.set(loadedColors);

    if (dirty) {
      SaveLabelColors(loadedColors).catch(e => console.error("Failed to backfill label colors:", e));
    }
  }

  // Applies user preferences from board config to local state and stores.
  function unpackUserPreferences(config: daedalus.BoardConfig | undefined): void {
    templates.set(config?.templates || []);

    if (config?.labelsExpanded !== undefined && config.labelsExpanded !== null) {
      labelsExpanded.set(config.labelsExpanded);
    }
    if (config?.minimalView !== undefined && config.minimalView !== null) {
      minimalView.set(config.minimalView);
    }
    if (config?.showYearProgress !== undefined && config.showYearProgress !== null) {
      showYearProgress = config.showYearProgress;
    }
    if (config?.darkMode !== undefined && config.darkMode !== null) {
      darkMode = config.darkMode;
    }
    if (config?.zoom !== undefined && config.zoom !== null) {
      zoomLevel = config.zoom;
    }
    document.documentElement.classList.toggle("light", !darkMode);
  }

  // Unpacks a LoadBoard response into all the relevant stores.
  function unpackBoardResponse(response: main.BoardResponse, roundTripMs: number = 0): void {
    boardData.set(response.lists);
    boardPath.set(response.boardPath || "");

    let maxId = 0;
    for (const cards of Object.values(response.lists)) {
      for (const card of cards) {
        if (card.metadata.id > maxId) {
          maxId = card.metadata.id;
        }
      }
    }
    maxCardId.set(maxId);

    if (response.boardPath) {
      WindowSetTitle(`Daedalus - ${response.boardPath}`);
    }

    unpackListConfig(response.config?.lists || []);
    boardTitle.set(response.config?.title || "Daedalus");
    backfillLabelColors(response.lists, response.config?.labelColors || {});
    unpackUserPreferences(response.config);

    const p = response.profile;
    loadProfile.set({
      configMs: p?.configMs ?? 0,
      scanMs: p?.scanMs ?? 0,
      mergeMs: p?.mergeMs ?? 0,
      totalMs: p?.totalMs ?? 0,
      transferMs: Math.max(0, roundTripMs - (p?.totalMs ?? 0)),
    });

    isLoaded.set(true);
  }

  // Loads the board from the backend and unpacks the response into stores.
  async function initBoard(boardPathArg: string = ""): Promise<void> {
    const path = boardPathArg || $boardPath;
    if (!path) {
      return;
    }
    try {
      const t0 = performance.now();
      const response = await LoadBoard(path);
      const roundTripMs = performance.now() - t0;
      unpackBoardResponse(response, roundTripMs);
    } catch (e) {
      addToast(`Failed to load board: ${e}`);
    }
  }

  // Checks app config and either auto-loads the default board or shows the welcome modal.
  async function startApp(): Promise<void> {
    try {
      const cfg = await GetAppConfig();
      if (cfg.defaultBoard) {
        await initBoard(cfg.defaultBoard);
      } else {
        activeModal = 'welcome';
      }
    } catch (e) {
      activeModal = 'welcome';
    }
  }

  // Handles a board being selected from the welcome modal.
  function handleWelcomeBoard(response: main.BoardResponse): void {
    activeModal = null;
    unpackBoardResponse(response);
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

  // Opens the draft-creation modal for the given list at the specified position.
  function createCard(listKey: string, position: string = "top"): void {
    if (isAtLimit(listKey, $boardData, $boardConfig)) {
      addToast("List is at its card limit");
      return;
    }
    draftPosition.set(position);
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

  const ZOOM_MIN = 0.5;
  const ZOOM_MAX = 1.5;
  const ZOOM_STEP = 0.1;
  const ZOOM_DEFAULT = 1.0;

  function zoomIn(): void {
    zoomLevel = Math.min(ZOOM_MAX, Math.round((zoomLevel + ZOOM_STEP) * 10) / 10);
    saveWithToast(SaveZoom(zoomLevel), "save zoom level");
  }

  function zoomOut(): void {
    zoomLevel = Math.max(ZOOM_MIN, Math.round((zoomLevel - ZOOM_STEP) * 10) / 10);
    saveWithToast(SaveZoom(zoomLevel), "save zoom level");
  }

  function zoomReset(): void {
    zoomLevel = ZOOM_DEFAULT;
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
      setShowAbout: v => { activeModal = v ? 'about' : null; },
      setShowLabelEditor: v => { activeModal = v ? 'labelEditor' : null; },
      setShowKeyboardHelp: v => { activeModal = v ? 'keyboardHelp' : null; },
      setFocusedCard: v => focusedCard.set(v),
      openCard: card => selectedCard.set(card),
      openCardEdit: card => { openInEditMode.set(true); selectedCard.set(card); },
      openSearch,
      createCard,
      createCardDefault,
      toggleMinimalView,
      zoomIn,
      zoomOut,
      zoomReset,
      scrollListIntoView,
      openWelcome: () => { activeModal = 'welcome'; },
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

  // Builds the common ListHeader prop object for a given list key.
  function listHeaderProps(listKey: string) {
    return {
      listKey,
      locked: lockedLists.has(listKey),
      color: $boardConfig[listKey]?.color || '',
      icon: $boardConfig[listKey]?.icon || '',
      pinState: getPinState(listKey),
      hasLeftPin: pinnedLeftLists.size > 0,
      hasRightPin: pinnedRightLists.size > 0,
      isLastList: listKey === visualKeys[visualKeys.length - 1],
      oncreatecard: () => createCard(listKey),
      onfullcollapse: () => toggleFullCollapse(listKey),
      onhalfcollapse: () => toggleHalfCollapse(listKey),
      onlock: () => toggleLock(listKey),
      onpinleft: () => pinList(listKey, "left"),
      onpinright: () => pinList(listKey, "right"),
      onunpin: () => unpinList(listKey),
      onreload: initBoard,
      onlistdragstart: () => startListDrag(listKey, !!getPinState(listKey)),
      onlistdragend: endListDrag,
      onmoveallcards: (target: string) => moveAllCards(listKey, target),
      ondeleteallcards: () => deleteAllCards(listKey),
      ondelete: () => deleteList(listKey),
    };
  }

  onMount(() => {
    document.addEventListener("dragover", handleGlobalDragOver, true);
    startApp();

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
    {zoomLevel} onzoomin={zoomIn} onzoomout={zoomOut} onzoomreset={zoomReset}
    oncreatecard={createCardDefault}
    onopenmodal={(name) => { activeModal = name as ActiveModal; }}
  />

  {#snippet renderList(listKey: string)}
    {#if collapsedLists.has(listKey)}
      {@const collapsedColor = $boardConfig[listKey]?.color || ''}
      {@const collapsedIcon = $boardConfig[listKey]?.icon || ''}
      <div class="list-column collapsed" data-list-key={listKey}
        role="button" tabindex="0" title="Expand list"
        class:pinned-left={getPinState(listKey) === 'left'}
        class:pinned-right={getPinState(listKey) === 'right'}
        class:list-dragging={$listDragging === listKey}
        style={collapsedColor ? `border-top: 3px solid ${collapsedColor}` : ''}
        onclick={() => toggleFullCollapse(listKey)}
        onkeydown={e => e.key === 'Enter' && toggleFullCollapse(listKey)}
      >
        <span class="collapsed-count">{getCountDisplay(listKey, $boardData, $boardConfig)}</span>
        {#if collapsedIcon}
          <span class="collapsed-icon">
            <CardIcon name={collapsedIcon} size={14} />
          </span>
        {/if}
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
        <ListHeader {...listHeaderProps(listKey)} />
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
        class:list-full={
          $dragState
          && $dragState.sourceListKey !== listKey
          && isAtLimit(listKey, $boardData, $boardConfig)
        }
        class:list-locked={$dragState && locked}
        class:list-dragging={$listDragging === listKey}
      >
        <ListHeader {...listHeaderProps(listKey)} />
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
          onclick={() => createCard(listKey, "bottom")}
          ondragenter={handleDragEnter}
          ondragover={(e) => handleFooterDragOver(e, listKey)}
          ondrop={(e) => handleDrop(e, listKey, initBoard)}
        >
          <Icon name="plus" size={12} />
        </button>
      </div>
    {/if}
  {/snippet}

  {#if $isLoaded}
    <div class="board-container" role="group" class:modal-open={$selectedCard || $draftListKey}
      style="zoom: {zoomLevel}"
      bind:this={boardContainerEl}
      ondragover={(e) => { if ($listDragging && boardContainerEl) { computeListDragOver(e, boardContainerEl, scrollableKeys); } }}
      ondrop={(e) => { if ($listDragging) { handleListDrop(e); } }}
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
      {#if ($dragState.card.metadata.labels?.length ?? 0) > 0}
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
    <KeyboardHelp onclose={() => activeModal = null} />
  {/if}
  {#if showAbout}
    <About onclose={() => activeModal = null} />
  {/if}
  {#if showLabelEditor}
    <LabelColorEditor onclose={() => activeModal = null} onreload={initBoard} />
  {/if}
  {#if showIconManager}
    <IconManager onclose={() => activeModal = null} onreload={initBoard} />
  {/if}
  {#if showTemplateManager}
    <TemplateManager onclose={() => activeModal = null} />
  {/if}
  {#if showBoardStats}
    <BoardStats onclose={() => activeModal = null} />
  {/if}
  {#if showScratchpad}
    <Scratchpad onclose={() => activeModal = null} />
  {/if}
  {#if showNewList}
    <NewListModal onclose={() => activeModal = null} onreload={initBoard} />
  {/if}
  {#if showWelcome}
    <WelcomeModal isOverlay={$isLoaded} onclose={() => activeModal = null} onboard={handleWelcomeBoard} />
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
    padding: 24px 0;
    gap: 10px;

    &::before,
    &::after {
      content: '';
      flex-shrink: 0;
      width: 14px;
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

  .collapsed-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    font-size: 0.9rem;
    line-height: 1;
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


</style>
