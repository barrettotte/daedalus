<script lang="ts">
  import {
    searchQuery, filteredBoardData, boardData, showMetrics, addToast,
  } from "../stores/board";
  import type { daedalus } from "../../wailsjs/go/models";
  import { SaveShowYearProgress, SaveDarkMode } from "../../wailsjs/go/main/App";
  import Icon from "./Icon.svelte";

  let {
    searchOpen = $bindable(false),
    showYearProgress = $bindable(false),
    darkMode = $bindable(true),
    showLabelEditor = $bindable(false),
    showKeyboardHelp = $bindable(false),
    showAbout = $bindable(false),
    oninitboard,
  }: {
    searchOpen: boolean;
    showYearProgress: boolean;
    darkMode: boolean;
    showLabelEditor: boolean;
    showKeyboardHelp: boolean;
    showAbout: boolean;
    oninitboard: () => Promise<void>;
  } = $props();

  let searchInputEl: HTMLInputElement | undefined = $state(undefined);

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
    SaveShowYearProgress(showYearProgress).catch(
      e => addToast(`Failed to save year progress state: ${e}`),
    );
  }

  // Toggles between dark and light mode, applying the CSS class and persisting to board.yaml.
  function toggleDarkMode(): void {
    darkMode = !darkMode;
    document.documentElement.classList.toggle("light", !darkMode);
    SaveDarkMode(darkMode).catch(e => addToast(`Failed to save dark mode state: ${e}`));
  }
</script>

<div class="top-bar">
  <h1>Daedalus</h1>
  <div class="top-bar-actions">
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
      <svg viewBox="0 0 24 24" width="14" height="14">
        <path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z"
          fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
        />
        <line x1="7" y1="7" x2="7.01" y2="7" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
      </svg>
    </button>
    <button class="top-btn" onclick={oninitboard} title="Reload board">
      <svg viewBox="0 0 24 24" width="14" height="14">
        <path d="M23 4v6h-6" fill="none" stroke="currentColor" stroke-width="2"
          stroke-linecap="round" stroke-linejoin="round"
        />
        <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" fill="none" stroke="currentColor"
          stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
        />
      </svg>
    </button>
    <button class="top-btn" class:active={showYearProgress} onclick={toggleYearProgress} title="Year progress">
      <svg viewBox="0 0 24 24" width="14" height="14">
        <path d="M6 2h12v6l-4 4 4 4v6H6v-6l4-4-4-4V2z" fill="none" stroke="currentColor"
          stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
        />
      </svg>
    </button>
    <button class="top-btn" onclick={toggleDarkMode}
      title={darkMode ? "Switch to light mode" : "Switch to dark mode"}
    >
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
          <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"
            fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
          />
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

<style lang="scss">
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
</style>
