<script lang="ts">
  // Top navigation bar with board title, search, label editor, and toggle buttons for dark mode, metrics, and help.

  import {
    searchQuery, filteredBoardData, boardData, boardTitle, showMetrics, addToast, saveWithToast, minimalView,
  } from "../stores/board";
  import type { daedalus } from "../../wailsjs/go/models";
  import { SaveShowYearProgress, SaveDarkMode, SaveBoardTitle, SaveMinimalView } from "../../wailsjs/go/main/App";
  import { autoFocus } from "../lib/utils";
  import Icon from "./Icon.svelte";

  let {
    searchOpen = $bindable(false),
    showYearProgress = $bindable(false),
    darkMode = $bindable(true),
    showLabelEditor = $bindable(false),
    showKeyboardHelp = $bindable(false),
    showAbout = $bindable(false),
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
    showKeyboardHelp: boolean;
    showAbout: boolean;
    zoomLevel: number;
    oncreatecard: () => void;
    onzoomin: () => void;
    onzoomout: () => void;
    onzoomreset: () => void;
  } = $props();

  let searchInputEl: HTMLInputElement | undefined = $state(undefined);
  let editingTitle = $state(false);
  let editTitleValue = $state("");

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
    saveWithToast(SaveShowYearProgress(showYearProgress), "save year progress state");
  }

  // Toggles minimal card view and persists to board.yaml.
  function toggleMinimalView(): void {
    minimalView.update(v => {
      const next = !v;
      saveWithToast(SaveMinimalView(next), "save minimal view state");
      return next;
    });
  }

  // Toggles between dark and light mode, applying the CSS class and persisting to board.yaml.
  function toggleDarkMode(): void {
    darkMode = !darkMode;
    document.documentElement.classList.toggle("light", !darkMode);
    saveWithToast(SaveDarkMode(darkMode), "save dark mode state");
  }
</script>

<div class="top-bar">
  {#if editingTitle}
    <input class="board-title-input" type="text" bind:value={editTitleValue} onblur={saveTitle} onkeydown={handleTitleKeydown} use:autoFocus/>
  {:else}
    <button class="board-title" onclick={startEditTitle} title="Click to edit board title">
      {$boardTitle}
    </button>
  {/if}
  <div class="top-bar-actions">
    <button class="top-btn" onclick={oncreatecard} title="New card (N)">
      <Icon name="plus" size={14} />
    </button>
    {#if searchOpen}
      <div class="search-bar" role="toolbar" aria-label="Search" tabindex="-1"
        onmousedown={(e) => {
          if ((e.target as HTMLElement).tagName !== "INPUT") {
            e.preventDefault();
          }
        }}
      >
        <span class="search-icon"><Icon name="search" size={14} /></span>
        <input type="text" class="search-input" placeholder="Search cards..."
          bind:this={searchInputEl} bind:value={$searchQuery}
          onkeydown={handleSearchKeydown} onblur={closeSearch}
        />
        {#if $searchQuery.trim()}
          {@const counts = matchedCardCount($filteredBoardData, $boardData)}
          <span class="search-count">{counts.matched}/{counts.total}</span>
          <button class="search-clear" onmousedown={() => searchQuery.set("")} title="Clear search">
            <Icon name="close" size={12} />
          </button>
        {/if}
      </div>
    {:else}
      <button class="top-btn" onclick={() => searchOpen = true} title="Search (/)">
        <Icon name="search" size={14} />
      </button>
    {/if}
    <button class="top-btn" onclick={() => showLabelEditor = true} title="Label manager">
      <Icon name="tag" size={14} />
    </button>
    <button class="top-btn" class:active={$minimalView} onclick={toggleMinimalView} title="Minimal view (M)">
      <Icon name="list" size={14} />
    </button>
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
    <button class="top-btn" onclick={() => window.location.reload()} title="Reload board">
      <Icon name="refresh" size={14} />
    </button>
    <button class="top-btn" class:active={showYearProgress} onclick={toggleYearProgress} title="Year progress">
      <Icon name="hourglass" size={14} />
    </button>
    <button class="top-btn" onclick={toggleDarkMode} title={darkMode ? "Switch to light mode" : "Switch to dark mode"}>
      <Icon name={darkMode ? "sun" : "moon"} size={14} />
    </button>
    <button class="top-btn" class:active={$showMetrics} onclick={() => showMetrics.update(v => !v)} title="Toggle metrics">
      <Icon name="chart-bar" size={14} />
    </button>
    <button class="top-btn" onclick={() => showKeyboardHelp = true} title="Keyboard shortcuts (?)">
      <Icon name="keyboard" size={14} />
    </button>
    <button class="top-btn" onclick={() => showAbout = true} title="About">
      <Icon name="info" size={14} />
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

<style lang="scss">
  .top-bar {
    height: 50px;
    background: var(--color-bg-inset);
    display: flex;
    align-items: center;
    padding: 0 20px;
    border-bottom: 1px solid var(--color-border);
  }

  .board-title {
    all: unset;
    font-size: 1.1rem;
    font-weight: 700;
    color: var(--color-text-primary);
    cursor: pointer;
    padding: 2px 6px;
    border-radius: 4px;

    &:hover {
      background: var(--overlay-hover);
    }
  }

  .board-title-input {
    background: var(--color-bg-base);
    border: 1px solid var(--color-accent);
    color: var(--color-text-primary);
    font-size: 1.1rem;
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

  .zoom-controls {
    display: flex;
    align-items: center;
    background: var(--overlay-hover-light);
    border-radius: 4px;
    border: 1px solid transparent;
    height: 28px;
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
</style>
