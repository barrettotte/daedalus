<script>
  import { onMount } from "svelte";
  import { LoadBoard, SaveListConfig } from "../wailsjs/go/main/App";
  import { boardData, boardConfig, sortedListKeys, isLoaded, selectedCard, showMetrics } from "./stores/board.js";
  import VirtualList from "./components/VirtualList.svelte";
  import Card from "./components/Card.svelte";
  import CardDetail from "./components/CardDetail.svelte";
  import Metrics from "./components/Metrics.svelte";

  let error = "";
  let editingTitle = null;
  let editingLimit = null;
  let editTitleValue = "";
  let editLimitValue = 0;

  // Loads the board from the backend and unpacks lists + config into separate stores.
  async function initBoard() {
    try {
      const response = await LoadBoard("");
      boardData.set(response.lists);
      boardConfig.set(response.config?.lists || {});
      isLoaded.set(true);
    } catch (e) {
      error = e.toString();
    }
  }

  // Strips the numeric prefix and underscores from directory names into display titles.
  function formatListName(rawName) {
    const parts = rawName.split('___');
    const name = parts.length > 1 ? parts[1] : rawName;
    return name
      .replace(/_/g, ' ')
      .replace(/\b\w/g, l => l.toUpperCase());
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
      console.error("Failed to save title:", e);
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
      console.error("Failed to save limit:", e);
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

  // Svelte action that focuses and selects the content of an input on mount.
  function autoFocus(node) {
    node.focus();
    node.select();
  }

  onMount(initBoard);

</script>

<main>
  <div class="top-bar">
    <h1>Daedalus</h1>
    <div class="top-bar-actions">
      <button class="top-btn" on:click={initBoard}>
        <svg viewBox="0 0 24 24" width="14" height="14">
          <path d="M23 4v6h-6" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        Refresh
      </button>
      <button class="top-btn" class:active={$showMetrics} on:click={() => showMetrics.update(v => !v)}>
        <svg viewBox="0 0 24 24" width="14" height="14">
          <rect x="18" y="3" width="4" height="18" rx="1" fill="none" stroke="currentColor" stroke-width="2"/><rect x="10" y="8" width="4" height="13" rx="1" fill="none" stroke="currentColor" stroke-width="2"/><rect x="2" y="13" width="4" height="8" rx="1" fill="none" stroke="currentColor" stroke-width="2"/>
        </svg>
        Metrics
      </button>
    </div>
  </div>

  {#if error}
    <div class="error">{error}</div>
  {:else if $isLoaded}
    <div class="board-container" class:modal-open={$selectedCard}>
      {#each sortedListKeys($boardData) as listKey}
        <div class="list-column">
          <div class="list-header">
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
          <div class="list-body">
            <VirtualList items={$boardData[listKey]} component={Card} estimatedHeight={90} />
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="loading">Loading cards...</div>
  {/if}
  <CardDetail />
  <Metrics />
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
  }

  .list-column {
    flex: 0 0 280px;
    display: flex;
    flex-direction: column;
    background: #22252b;
    border-radius: 8px;
    max-height: 100%;
    border: 1px solid #333;
  }

  .list-header {
    padding: 8px 10px;
    border-bottom: 1px solid #333;
    display: flex;
    justify-content: space-between;
    align-items: center;
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
</style>
