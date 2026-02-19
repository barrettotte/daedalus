<script lang="ts">
  // Top navigation bar with board title, search, label editor, and toggle buttons for dark mode, metrics, and help.

  import {
    searchQuery, filteredBoardData, boardData, boardTitle, boardPath, showMetrics,
    addToast, saveWithToast, minimalView, toggleMinimalView, labelColors,
  } from "../stores/board";
  import type { daedalus } from "../../wailsjs/go/models";
  import { SaveShowYearProgress, SaveDarkMode, SaveBoardTitle } from "../../wailsjs/go/main/App";
  import { autoFocus, clickOutside, copyToClipboard } from "../lib/utils";
  import Icon from "./Icon.svelte";
  import SearchFilterPopover from "./SearchFilterPopover.svelte";
  import appIcon from "../assets/images/daedalus.svg";

  let {
    searchOpen = $bindable(false),
    showYearProgress = $bindable(false),
    darkMode = $bindable(true),
    showLabelEditor = $bindable(false),
    showIconManager = $bindable(false),
    showTemplateManager = $bindable(false),
    showScratchpad = $bindable(false),
    showBoardStats = $bindable(false),
    showKeyboardHelp = $bindable(false),
    showAbout = $bindable(false),
    showNewList = $bindable(false),
    showWelcome = $bindable(false),
    zoomLevel = 1.0,
    oncreatecard,
    onzoomin,
    onzoomout,
    onzoomreset,
  }: {
    searchOpen: boolean;
    showYearProgress: boolean;
    darkMode: boolean;
    showLabelEditor: boolean;
    showIconManager: boolean;
    showTemplateManager: boolean;
    showScratchpad: boolean;
    showBoardStats: boolean;
    showKeyboardHelp: boolean;
    showAbout: boolean;
    showNewList: boolean;
    showWelcome: boolean;
    zoomLevel: number;
    oncreatecard: () => void;
    onzoomin: () => void;
    onzoomout: () => void;
    onzoomreset: () => void;
  } = $props();

  let searchInputEl: HTMLInputElement | undefined = $state(undefined);
  let filterOpen = $state(false);
  let editingTitle = $state(false);
  let editTitleValue = $state("");
  let menuOpen = $state(false);
  let menuTriggerEl: HTMLButtonElement | undefined = $state(undefined);

  // Closes the overflow menu and returns focus to the trigger button.
  function closeMenu(): void {
    menuOpen = false;
    menuTriggerEl?.focus();
  }

  // Opens the board title for inline editing.
  function startEditTitle(): void {
    editTitleValue = $boardTitle;
    editingTitle = true;
  }

  // Saves the edited board title, defaulting to "Daedalus" if blank.
  function saveTitle(): void {
    editingTitle = false;
    const newTitle = editTitleValue.trim() || "Daedalus";
    boardTitle.set(newTitle);
    saveWithToast(SaveBoardTitle(newTitle === "Daedalus" ? "" : newTitle), "save board title");
  }

  // Handles keydown on the title input.
  function handleTitleKeydown(e: KeyboardEvent): void {
    if (e.key === "Enter") {
      saveTitle();
    } else if (e.key === "Escape") {
      editingTitle = false;
    }
  }

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

  // Auto-focuses the search input when the search bar opens.
  $effect(() => {
    if (searchOpen && searchInputEl) {
      requestAnimationFrame(() => {
        searchInputEl!.focus();
        searchInputEl!.setSelectionRange(searchInputEl!.value.length, searchInputEl!.value.length);
      });
    }
  });

  // Counts total matched cards across all lists for the search badge.
  function matchedCardCount(
    filtered: Record<string, daedalus.KanbanCard[]>,
    raw: Record<string, daedalus.KanbanCard[]>,
  ): { matched: number; total: number } {
    let matched = 0;
    let total = 0;

    for (const key of Object.keys(raw)) {
      total += (raw[key] || []).length;
      matched += (filtered[key] || []).length;
    }
    return { matched, total };
  }

  // Collapses the search bar, clears the query, and blurs the input.
  function closeSearch(): void {
    searchQuery.set("");
    searchOpen = false;
    filterOpen = false;
    searchInputEl?.blur();
  }

  // Inserts a filter prefix into the search query from the popover.
  function insertFilter(prefix: string): void {
    const current = $searchQuery.trim();
    if (current.split(/\s+/).includes(prefix)) {
      filterOpen = false;
      return;
    }
    searchQuery.set(current ? `${current} ${prefix}` : prefix);
    filterOpen = false;
    searchInputEl?.focus();
  }

  // Handles keydown events inside the search input.
  function handleSearchKeydown(e: KeyboardEvent): void {
    if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      if (filterOpen) {
        filterOpen = false;
      } else {
        closeSearch();
      }
    }
  }

  // Toggles year progress bar visibility and persists to board.yaml.
  function toggleYearProgress(): void {
    showYearProgress = !showYearProgress;
    saveWithToast(SaveShowYearProgress(showYearProgress), "save year progress state");
  }


  // Toggles between dark and light mode, applying the CSS class and persisting to board.yaml.
  function toggleDarkMode(): void {
    darkMode = !darkMode;
    document.documentElement.classList.toggle("light", !darkMode);
    saveWithToast(SaveDarkMode(darkMode), "save dark mode state");
  }
</script>

<div class="top-bar">
  <div class="top-bar-brand">
    <button class="app-icon-btn" title="Copy board path" onclick={() => $boardPath && copyToClipboard($boardPath, "Board path")}>
      <img src={appIcon} alt="" class="app-icon" />
    </button>
    {#if editingTitle}
      <input class="board-title-input" type="text" bind:value={editTitleValue} onblur={saveTitle} onkeydown={handleTitleKeydown} use:autoFocus/>
    {:else}
      <button class="board-title" onclick={startEditTitle} title="Click to edit board title">
        {$boardTitle}
      </button>
    {/if}
  </div>
  <div class="top-bar-actions">
    <button class="top-btn" onclick={oncreatecard} title="New card (N)">
      <Icon name="plus" size={14} />
    </button>
    <button class="top-btn" onclick={() => showNewList = true} title="New list">
      <Icon name="list-plus" size={14} />
    </button>
    {#if searchOpen}
      <div class="search-bar-wrapper">
        <div class="search-bar" role="toolbar" aria-label="Search" tabindex="-1"
          onmousedown={(e) => {
            if ((e.target as HTMLElement).tagName !== "INPUT") {
              e.preventDefault();
            }
          }}
        >
          <span class="search-icon"><Icon name="search" size={14} /></span>
          <input type="text" class="search-input" placeholder="Search cards... (#label, url:, icon:)"
            bind:this={searchInputEl} bind:value={$searchQuery}
            onkeydown={handleSearchKeydown} onblur={() => { if (!filterOpen) { closeSearch(); } }}
          />
          <button class="search-filter-btn" class:active={filterOpen} title="Filter by field" onmousedown={(e) => { e.preventDefault(); filterOpen = !filterOpen; }}>
            <Icon name="filter" size={12} />
          </button>
          {#if $searchQuery.trim()}
            {@const counts = matchedCardCount($filteredBoardData, $boardData)}
            <span class="search-count">{counts.matched}/{counts.total}</span>
            <button class="search-clear" onmousedown={() => searchQuery.set("")} title="Clear search">
              <Icon name="close" size={12} />
            </button>
          {/if}
        </div>
        {#if filterOpen}
          <SearchFilterPopover lists={$boardData} colors={$labelColors} query={$searchQuery} onselect={insertFilter} />
        {/if}
      </div>
    {:else}
      <button class="top-btn" onclick={() => searchOpen = true} title="Search (/)">
        <Icon name="search" size={14} />
      </button>
    {/if}
    <div class="zoom-controls">
      <button class="zoom-btn" onclick={onzoomout} title="Zoom out (-)">
        <Icon name="minus" size={10} />
      </button>
      <button class="zoom-label" onclick={onzoomreset} title="Reset zoom (0)">
        {Math.round(zoomLevel * 100)}%
      </button>
      <button class="zoom-btn" onclick={onzoomin} title="Zoom in (+)">
        <Icon name="plus" size={10} />
      </button>
    </div>
    <button class="top-btn" onclick={() => { showWelcome = true; }} title="Switch board (Ctrl+O)">
      <Icon name="folder" size={14} />
    </button>
    <button class="top-btn" onclick={() => window.location.reload()} title="Reload board">
      <Icon name="refresh" size={14} />
    </button>
    <div class="menu-wrapper" use:clickOutside={() => { menuOpen = false; }}>
      <button class="top-btn" class:active={menuOpen} bind:this={menuTriggerEl} title="More actions"
        onclick={() => { menuOpen = !menuOpen; }}
        onkeydown={(e) => { if (e.key === 'Escape' && menuOpen) { e.stopPropagation(); closeMenu(); } }}
      >
        <Icon name="menu-dots" size={14} />
      </button>
      {#if menuOpen}
        <div class="overflow-menu" role="menu" tabindex="-1" onkeydown={(e) => { if (e.key === 'Escape') { e.stopPropagation(); closeMenu(); } }}>
          <button class="overflow-item" onclick={() => { closeMenu(); showLabelEditor = true; }}>
            <Icon name="tag" size={14} />
            Label manager
          </button>
          <button class="overflow-item" onclick={() => { closeMenu(); showIconManager = true; }}>
            <Icon name="image" size={14} />
            Icon manager
          </button>
          <button class="overflow-item" onclick={() => { closeMenu(); showTemplateManager = true; }}>
            <Icon name="template" size={14} />
            Template manager
          </button>
          <button class="overflow-item" onclick={() => { closeMenu(); showScratchpad = true; }}>
            <Icon name="notepad" size={14} />
            Scratchpad
          </button>
          <button class="overflow-item" onclick={() => { closeMenu(); showBoardStats = true; }}>
            <Icon name="chart-bar" size={14} />
            Board statistics
          </button>
          <div class="overflow-divider"></div>
          <button class="overflow-item" onclick={() => { closeMenu(); toggleYearProgress(); }}>
            <span class="overflow-icon" class:active={showYearProgress}>
              <Icon name="hourglass" size={14} />
            </span>
            Year progress
            {#if showYearProgress}<Icon name="check" size={12} />{/if}
          </button>
          <button class="overflow-item" onclick={() => { closeMenu(); toggleDarkMode(); }}>
            <span class="overflow-icon" class:active={darkMode}>
              <Icon name={darkMode ? "moon" : "sun"} size={14} />
            </span>
            Dark mode
            {#if darkMode}<Icon name="check" size={12} />{/if}
          </button>
          <button class="overflow-item" onclick={() => { closeMenu(); showMetrics.update(v => !v); }}>
            <span class="overflow-icon" class:active={$showMetrics}>
              <Icon name="activity" size={14} />
            </span>
            Metrics
            {#if $showMetrics}<Icon name="check" size={12} />{/if}
          </button>
          <button class="overflow-item" onclick={() => { closeMenu(); toggleMinimalView(); }}>
            <span class="overflow-icon" class:active={$minimalView}>
              <Icon name="list" size={14} />
            </span>
            Minimal view
            {#if $minimalView}<Icon name="check" size={12} />{/if}
          </button>
          <div class="overflow-divider"></div>
          <button class="overflow-item" onclick={() => { closeMenu(); showKeyboardHelp = true; }}>
            <Icon name="keyboard" size={14} />
            Keyboard shortcuts
          </button>
          <button class="overflow-item" onclick={() => { closeMenu(); showAbout = true; }}>
            <Icon name="info" size={14} />
            About
          </button>
        </div>
      {/if}
    </div>
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

<style lang="scss">
  .top-bar {
    min-height: 62px;
    background: var(--color-bg-inset);
    display: flex;
    flex-wrap: nowrap;
    align-items: center;
    padding: 0 16px 0 10px;
    gap: 6px;
    border-bottom: 1px solid var(--color-border);
  }

  .top-bar-brand {
    display: flex;
    align-items: center;
    min-width: 0;
  }

  .app-icon-btn {
    all: unset;
    display: flex;
    align-items: center;
    cursor: pointer;
    flex-shrink: 0;
    margin-right: 6px;
    border-radius: 6px;

    &:hover {
      opacity: 0.8;
    }
  }

  .app-icon {
    height: 42px;
    width: 42px;
  }

  .board-title {
    all: unset;
    font-size: 1.4rem;
    font-weight: 700;
    color: var(--color-text-primary);
    cursor: pointer;
    padding: 2px 6px;
    border-radius: 4px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    flex-shrink: 1;
    min-width: 0;

    &:hover {
      background: var(--overlay-hover);
    }
  }

  .board-title-input {
    background: var(--color-bg-base);
    border: 1px solid var(--color-accent);
    color: var(--color-text-primary);
    font-size: 1.4rem;
    font-weight: 700;
    padding: 2px 6px;
    border-radius: 4px;
    outline: none;
    width: 180px;
  }

  .year-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 4px 20px;
    background: var(--color-bg-inset);
    border-bottom: 1px solid var(--color-border);
  }

  .year-bar-label {
    font-family: var(--font-mono);
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
    font-family: var(--font-mono);
    font-size: 0.68rem;
    color: var(--color-text-muted);
    flex-shrink: 0;
  }

  .year-bar-remaining {
    font-family: var(--font-mono);
    font-size: 0.68rem;
    color: var(--color-text-muted);
    flex-shrink: 0;
    margin-left: 8px;
  }

  .search-bar-wrapper {
    position: relative;
  }

  .search-bar {
    display: flex;
    align-items: center;
    width: 280px;
    max-width: 100%;
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
    font-family: var(--font-mono);
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

  .search-filter-btn {
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

    &:hover, &.active {
      color: var(--color-text-primary);
      background: var(--overlay-hover-medium);
    }
  }

  .zoom-controls {
    display: flex;
    align-items: center;
    background: var(--overlay-hover-light);
    border-radius: 5px;
    border: 1px solid transparent;
    height: 34px;
  }

  .zoom-btn {
    all: unset;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 100%;
    cursor: pointer;
    color: var(--color-text-secondary);
    border-radius: 4px;

    &:hover {
      background: var(--overlay-hover-medium);
      color: var(--color-text-primary);
    }
  }

  .zoom-label {
    all: unset;
    font-family: var(--font-mono);
    font-size: 0.7rem;
    font-weight: 500;
    color: var(--color-text-muted);
    padding: 0 2px;
    cursor: pointer;
    min-width: 34px;
    text-align: center;

    &:hover {
      color: var(--color-text-primary);
    }
  }

  .top-bar-actions {
    display: flex;
    flex-wrap: nowrap;
    gap: 6px;
    align-items: center;
    margin-left: auto;
    justify-content: flex-end;
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
    padding: 9px 11px;
    border-radius: 5px;
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

  .menu-wrapper {
    position: relative;
  }

  .overflow-menu {
    position: absolute;
    top: 100%;
    right: 0;
    margin-top: 4px;
    z-index: var(--z-dropdown);
    background: var(--color-bg-surface);
    border: 1px solid var(--color-border-medium);
    border-radius: 6px;
    padding: 4px 0;
    min-width: 190px;
    box-shadow: var(--shadow-md);
  }

  .overflow-item {
    all: unset;
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 6px 12px;
    font-size: 0.8rem;
    color: var(--color-text-secondary);
    cursor: pointer;
    box-sizing: border-box;
    white-space: nowrap;

    &:hover {
      background: var(--overlay-hover);
      color: var(--color-text-primary);
    }
  }

  .overflow-icon {
    display: flex;
    align-items: center;

    &.active {
      color: var(--color-accent);
    }
  }

  .overflow-divider {
    height: 1px;
    background: var(--color-border);
    margin: 4px 0;
  }
</style>
