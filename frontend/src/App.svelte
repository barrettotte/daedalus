<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { SvelteSet } from "svelte/reactivity";
  import { WindowSetTitle } from "../wailsjs/runtime/runtime";
  import {
    LoadBoard, SaveLabelsExpanded, SaveShowYearProgress,
    SaveCollapsedLists, SaveHalfCollapsedLists, SaveDarkMode, DeleteCard,
  } from "../wailsjs/go/main/App";
  import {
    boardData, boardConfig, boardPath, sortedListKeys, isLoaded,
    selectedCard, draftListKey, draftPosition, showMetrics,
    labelsExpanded, dragState, dropTarget, focusedCard, openInEditMode,
    removeCardFromBoard, addToast, isAtLimit,
    searchQuery, filteredBoardData,
  } from "./stores/board";
  import type { daedalus } from "../wailsjs/go/models";
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
  import About from "./components/About.svelte";

  let error = $state("");
  let collapsedLists = $state(new SvelteSet<string>());
  let halfCollapsedLists = $state(new SvelteSet<string>());
  let showKeyboardHelp = $state(false);
  let showAbout = $state(false);
  let showYearProgress = $state(false);
  let darkMode = $state(true);
  let confirmingFocusDelete = $state(false);
  let boardContainerEl: HTMLDivElement | undefined = $state(undefined);
  let searchInputEl: HTMLInputElement | undefined = $state(undefined);
  let searchOpen = $state(false);

  // Computes year progress percentage, day of year, and remaining time from a timestamp.
  function computeYearInfo(now: Date): { pct: string; remaining: string; dayOfYear: number } {
    const year = now.getFullYear();
    const start = new Date(year, 0, 1).getTime();
    const end = new Date(year + 1, 0, 1).getTime();

    const pct = ((now.getTime() - start) / (end - start) * 100).toFixed(5);
    const dayOfYear = Math.ceil((now.getTime() - start) / 86400000);
    const leftMs = end - now.getTime();
    const totalSec = Math.max(0, Math.floor(leftMs / 1000));

    const h = Math.floor(totalSec / 3600);
    const m = Math.floor((totalSec % 3600) / 60);
    const s = totalSec % 60;
    return { pct, remaining: `${h}h ${m}m ${s}s`, dayOfYear };
  }

  let yearInfo = $state(computeYearInfo(new Date()));
  let yearTimer: ReturnType<typeof setInterval> | null = null;

  // Starts/stops the 1-second year countdown timer based on bar visibility.
  $effect(() => {
    if (showYearProgress) {
      yearInfo = computeYearInfo(new Date());
      yearTimer = setInterval(() => {
        yearInfo = computeYearInfo(new Date());
      }, 1000);
    } else if (yearTimer) {
      clearInterval(yearTimer);
      yearTimer = null;
    }
    return () => {
      if (yearTimer) {
        clearInterval(yearTimer);
        yearTimer = null;
      }
    };
  });

  // Counts total matched cards across all lists for the search badge.
  function matchedCardCount(filtered: Record<string, any[]>, raw: Record<string, any[]>): { matched: number; total: number } {
    let matched = 0;
    let total = 0;

    for (const key of Object.keys(raw)) {
      total += (raw[key] || []).length;
      matched += (filtered[key] || []).length;
    }
    return { matched, total };
  }

  // Expands the search bar, optionally pre-filling a prefix, and focuses the input on next tick.
  function openSearch(prefill: string = ""): void {
    searchOpen = true;
    if (prefill) {
      searchQuery.set(prefill);
    }

    requestAnimationFrame(() => {
      if (searchInputEl) {
        searchInputEl.focus();
        // Place cursor at end after prefill
        searchInputEl.setSelectionRange(searchInputEl.value.length, searchInputEl.value.length);
      }
    });
  }

  // Collapses the search bar, clears the query, and blurs the input.
  function closeSearch(): void {
    searchQuery.set("");
    searchOpen = false;
    searchInputEl?.blur();
  }

  // Handles keydown events inside the search input.
  function handleSearchKeydown(e: KeyboardEvent): void {
    if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      closeSearch();
    }
  }

  // Toggles year progress bar visibility and persists to board.yaml.
  function toggleYearProgress(): void {
    showYearProgress = !showYearProgress;
    SaveShowYearProgress(showYearProgress).catch(e => addToast(`Failed to save year progress state: ${e}`));
  }

  // Toggles between dark and light mode, applying the CSS class and persisting to board.yaml.
  function toggleDarkMode(): void {
    darkMode = !darkMode;
    document.documentElement.classList.toggle("light", !darkMode);
    SaveDarkMode(darkMode).catch(e => addToast(`Failed to save dark mode state: ${e}`));
  }

  // Cycles a list through expanded -> half-collapsed -> fully collapsed -> expanded.
  function cycleCollapseState(listKey: string): void {
    if (halfCollapsedLists.has(listKey)) {
      // Half-collapsed -> fully collapsed
      halfCollapsedLists.delete(listKey);
      collapsedLists.add(listKey);
    } else if (collapsedLists.has(listKey)) {
      // Fully collapsed -> expanded
      collapsedLists.delete(listKey);
    } else {
      // Expanded -> half-collapsed
      halfCollapsedLists.add(listKey);
    }
    SaveCollapsedLists([...collapsedLists]).catch(e => addToast(`Failed to save collapsed state: ${e}`));
    SaveHalfCollapsedLists([...halfCollapsedLists]).catch(e => addToast(`Failed to save half-collapsed state: ${e}`));
  }

  // Expands a half-collapsed list to full (used by the "Show N more" button).
  function expandFromHalf(listKey: string): void {
    halfCollapsedLists.delete(listKey);
    SaveHalfCollapsedLists([...halfCollapsedLists]).catch(e => addToast(`Failed to save half-collapsed state: ${e}`));
  }

  // Loads the board from the backend and unpacks lists + config into separate stores.
  async function initBoard(): Promise<void> {
    error = "";
    try {
      const response = await LoadBoard("");
      boardData.set(response.lists);
      boardConfig.set(response.config?.lists || {} as Record<string, any>);
      boardPath.set(response.boardPath || "");

      if (response.boardPath) {
        WindowSetTitle(`Daedalus - ${response.boardPath}`);
      }
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
      if (response.config?.collapsedLists) {
        collapsedLists = new SvelteSet(response.config.collapsedLists);
      }
      if (response.config?.halfCollapsedLists) {
        halfCollapsedLists = new SvelteSet(response.config.halfCollapsedLists);
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

    // Escape closes overlays first; all other keys ignored while they're open.
    if (showAbout) {
      if (e.key === "Escape") {
        e.preventDefault();
        showAbout = false;
      }
      return;
    }
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

    // / - Open and focus search bar
    if (e.key === "/") {
      e.preventDefault();
      openSearch();
      return;
    }

    // # - Open search with # prefix for card ID jump
    if (e.key === "#") {
      e.preventDefault();
      openSearch("#");
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
        const maxIndex = halfCollapsedLists.has(focus.listKey) ? Math.min(4, cards.length - 1) : cards.length - 1;

        if (focus.cardIndex < maxIndex) {
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

  // Auto-open a card when search query is exactly #<digits> and a unique match exists.
  $effect(() => {
    const q = $searchQuery.trim();
    if (!/^#\d+$/.test(q)) {
      return;
    }
    const targetId = Number(q.slice(1));
    const lists = $boardData;

    let found: daedalus.KanbanCard | null = null;
    for (const key of Object.keys(lists)) {
      for (const card of lists[key]) {
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

  onMount(initBoard);

  onDestroy(() => {
    stopAutoScroll();
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
  <div class="top-bar">
    <h1>Daedalus</h1>
    <div class="top-bar-actions">
      {#if searchOpen}
        <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
        <div class="search-bar" role="search" onmousedown={(e) => {
          if ((e.target as HTMLElement).tagName !== "INPUT") { e.preventDefault(); }
        }}>
          <svg class="search-icon" viewBox="0 0 24 24" width="14" height="14">
            <circle cx="11" cy="11" r="8" fill="none" stroke="currentColor" stroke-width="2"/>
            <line x1="21" y1="21" x2="16.65" y2="16.65" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          </svg>
          <input type="text" class="search-input" placeholder="Search cards..."
            bind:this={searchInputEl} bind:value={$searchQuery}
            onkeydown={handleSearchKeydown} onblur={closeSearch}
          />
          {#if $searchQuery.trim()}
            {@const counts = matchedCardCount($filteredBoardData, $boardData)}
            <span class="search-count">{counts.matched}/{counts.total}</span>
            <button class="search-clear" onmousedown={() => searchQuery.set("")} title="Clear search">
              <svg viewBox="0 0 24 24" width="12" height="12">
                <line x1="18" y1="6" x2="6" y2="18" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
                <line x1="6" y1="6" x2="18" y2="18" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
              </svg>
            </button>
          {/if}
        </div>
      {:else}
        <button class="top-btn" onclick={() => openSearch()} title="Search (/)">
          <svg viewBox="0 0 24 24" width="14" height="14">
            <circle cx="11" cy="11" r="8" fill="none" stroke="currentColor" stroke-width="2"/>
            <line x1="21" y1="21" x2="16.65" y2="16.65" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          </svg>
        </button>
      {/if}
      <button class="top-btn" onclick={initBoard} title="Reload board">
        <svg viewBox="0 0 24 24" width="14" height="14">
          <path d="M23 4v6h-6" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </button>
      <button class="top-btn" class:active={showYearProgress} onclick={toggleYearProgress} title="Year progress">
        <svg viewBox="0 0 24 24" width="14" height="14">
          <path d="M6 2h12v6l-4 4 4 4v6H6v-6l4-4-4-4V2z" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </button>
      <button class="top-btn" onclick={toggleDarkMode} title={darkMode ? "Switch to light mode" : "Switch to dark mode"}>
        {#if darkMode}
          <svg viewBox="0 0 24 24" width="14" height="14">
            <circle cx="12" cy="12" r="5" fill="none" stroke="currentColor" stroke-width="2"/>
            <line x1="12" y1="1" x2="12" y2="3" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            <line x1="12" y1="21" x2="12" y2="23" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            <line x1="1" y1="12" x2="3" y2="12" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            <line x1="21" y1="12" x2="23" y2="12" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          </svg>
        {:else}
          <svg viewBox="0 0 24 24" width="14" height="14">
            <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        {/if}
      </button>
      <button class="top-btn" class:active={$showMetrics} onclick={() => showMetrics.update(v => !v)} title="Toggle metrics">
        <svg viewBox="0 0 24 24" width="14" height="14">
          <rect x="18" y="3" width="4" height="18" rx="1" fill="none" stroke="currentColor" stroke-width="2"/>
          <rect x="10" y="8" width="4" height="13" rx="1" fill="none" stroke="currentColor" stroke-width="2"/>
          <rect x="2" y="13" width="4" height="8" rx="1" fill="none" stroke="currentColor" stroke-width="2"/>
        </svg>
      </button>
      <button class="top-btn" onclick={() => showKeyboardHelp = true} title="Keyboard shortcuts (?)">
        <svg viewBox="0 0 24 24" width="14" height="14">
          <rect x="2" y="4" width="20" height="16" rx="2" fill="none" stroke="currentColor" stroke-width="2"/>
          <line x1="6" y1="10" x2="6" y2="10.01" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          <line x1="10" y1="10" x2="10" y2="10.01" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          <line x1="14" y1="10" x2="14" y2="10.01" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          <line x1="18" y1="10" x2="18" y2="10.01" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          <line x1="8" y1="16" x2="16" y2="16" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
        </svg>
      </button>
      <button class="top-btn" onclick={() => showAbout = true} title="About">
        <svg viewBox="0 0 24 24" width="14" height="14">
          <circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="2"/>
          <line x1="12" y1="16" x2="12" y2="12" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          <line x1="12" y1="8" x2="12.01" y2="8" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
        </svg>
      </button>
    </div>
  </div>
  {#if showYearProgress}
    <div class="year-bar" title="{yearInfo.pct}% of {new Date().getFullYear()}">
      <span class="year-bar-label">Day {yearInfo.dayOfYear} of {new Date().getFullYear()}</span>
      <div class="year-bar-track">
        <div class="year-bar-fill" style="width: {yearInfo.pct}%"></div>
      </div>
      <span class="year-bar-pct">{yearInfo.pct}%</span>
      <span class="year-bar-remaining">{yearInfo.remaining}</span>
    </div>
  {/if}

  {#if error}
    <div class="error">{error}</div>
  {:else if $isLoaded}
    <div class="board-container" class:modal-open={$selectedCard || $draftListKey} bind:this={boardContainerEl}>
      {#each sortedListKeys($boardData) as listKey}
        {#if collapsedLists.has(listKey)}
          <div class="list-column collapsed" role="button" tabindex="0"
            onclick={() => cycleCollapseState(listKey)}
            onkeydown={e => e.key === 'Enter' && cycleCollapseState(listKey)}
          >
            <span class="collapsed-count">
              {getCountDisplay(listKey, $boardData, $boardConfig)}
            </span>
            <span class="collapsed-title">
              {getDisplayTitle(listKey, $boardConfig)}
            </span>
          </div>
        {:else if halfCollapsedLists.has(listKey)}
          {@const allItems = $filteredBoardData[listKey] || []}
          {@const visibleItems = allItems.slice(0, 5)}
          {@const remaining = allItems.length - 5}
          <div class="list-column half-collapsed">
            <ListHeader {listKey}
              oncreatecard={() => createCard(listKey)}
              oncollapse={() => cycleCollapseState(listKey)}
              onreload={initBoard}
            />
            <div class="list-body half-collapsed-body" role="list">
              <VirtualList items={visibleItems} component={Card} estimatedHeight={90} listKey={listKey}
                focusIndex={$focusedCard?.listKey === listKey ? $focusedCard.cardIndex : -1}
              />
            </div>
            {#if remaining > 0}
              <button class="show-more-bar" onclick={() => expandFromHalf(listKey)}>
                Show {remaining} more
              </button>
            {/if}
          </div>
        {:else}
          <div class="list-column"
            class:list-full={$dragState
              && $dragState.sourceListKey !== listKey
              && isAtLimit(listKey, $boardData, $boardConfig)}
          >
            <ListHeader {listKey}
              oncreatecard={() => createCard(listKey)}
              oncollapse={() => cycleCollapseState(listKey)}
              onreload={initBoard}
            />
            <div class="list-body" role="list"
              ondragenter={handleDragEnter}
              ondragover={(e) => handleDragOver(e, listKey)}
              ondragleave={handleDragLeave}
              ondrop={(e) => handleDrop(e, listKey, initBoard)}
            >
              <VirtualList items={$filteredBoardData[listKey]} component={Card} estimatedHeight={90} listKey={listKey}
                focusIndex={$focusedCard?.listKey === listKey ? $focusedCard.cardIndex : -1}
              />
            </div>
            <button class="list-footer-add" title="Add card to bottom"
              onclick={() => createCardBottom(listKey)}
              ondragenter={handleDragEnter}
              ondragover={(e) => handleFooterDragOver(e, listKey)}
              ondrop={(e) => handleDrop(e, listKey, initBoard)}
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
  {#if showAbout}
    <About onclose={() => showAbout = false} />
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

  .top-bar {
    height: 50px;
    background: var(--color-bg-inset);
    display: flex;
    align-items: center;
    padding: 0 20px;
    border-bottom: 1px solid #000;
  }

  .year-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 4px 20px;
    background: var(--color-bg-inset);
    border-bottom: 1px solid #000;
  }

  .year-bar-label {
    font-size: 0.68rem;
    font-weight: 700;
    color: var(--color-text-muted);
    flex-shrink: 0;
    margin-right: 8px;
  }

  .year-bar-track {
    flex: 1;
    height: 6px;
    background: var(--color-border);
    border-radius: 3px;
    overflow: hidden;
  }

  .year-bar-fill {
    height: 100%;
    background: var(--color-accent);
    border-radius: 3px;
  }

  .year-bar-pct {
    font-size: 0.68rem;
    color: var(--color-text-muted);
    flex-shrink: 0;
  }

  .year-bar-remaining {
    font-size: 0.68rem;
    font-family: monospace;
    color: var(--color-text-muted);
    flex-shrink: 0;
    margin-left: 8px;
  }

  .search-bar {
    display: flex;
    align-items: center;
    width: 280px;
    background: var(--overlay-hover-light);
    border: 1px solid var(--color-border-medium);
    border-radius: 4px;
    padding: 0 8px;
    height: 30px;
    gap: 6px;
    transition: border-color 0.15s;

    &:focus-within {
      border-color: var(--color-accent);
    }
  }

  .search-icon {
    flex-shrink: 0;
    color: var(--color-text-muted);
  }

  .search-input {
    all: unset;
    flex: 1;
    font-size: 0.8rem;
    color: var(--color-text-primary);
    min-width: 0;
    text-align: left;

    &::placeholder {
      color: var(--color-text-muted);
    }
  }

  .search-count {
    flex-shrink: 0;
    font-size: 0.7rem;
    color: var(--color-text-muted);
    padding: 1px 6px;
    background: var(--overlay-hover-medium);
    border-radius: 3px;
  }

  .search-clear {
    all: unset;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    cursor: pointer;
    color: var(--color-text-muted);
    border-radius: 3px;

    &:hover {
      color: var(--color-text-primary);
      background: var(--overlay-hover-medium);
    }
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
