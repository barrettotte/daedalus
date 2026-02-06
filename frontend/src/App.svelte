<script>
  import { onMount } from "svelte";
  import { LoadBoard } from "../wailsjs/go/main/App";
  import { boardData, sortedListKeys, isLoaded, selectedCard, showMetrics } from "./stores/board.js";
  import VirtualList from "./components/VirtualList.svelte";
  import Card from "./components/Card.svelte";
  import CardDetail from "./components/CardDetail.svelte";
  import Metrics from "./components/Metrics.svelte";

  let error = "";

  // Loads the board from the backend and updates stores.
  async function initBoard() {
    try {
      const data = await LoadBoard("");
      boardData.set(data);
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
            <h2>{formatListName(listKey)}</h2>
            <span class="count">{$boardData[listKey].length}</span>
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

  .list-header h2 {
    margin: 0;
    font-size: 0.85rem;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .list-body {
    flex: 1;
    overflow: hidden;
  }

  .count {
    background: #333;
    padding: 2px 8px;
    border-radius: 10px;
    font-size: 0.8rem;
  }

  .board-container.modal-open :global(.virtual-scroll-container) {
    overflow-y: hidden;
  }
</style>
