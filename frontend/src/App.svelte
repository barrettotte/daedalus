<script>
  import { onMount } from "svelte";
  import { LoadBoard } from "../wailsjs/go/main/App";
  import { boardData, sortedListKeys, isLoaded, selectedCard } from "./stores/board.js";
  import VirtualList from "./components/VirtualList.svelte";
  import Card from "./components/Card.svelte";
  import CardDetail from "./components/CardDetail.svelte";

  let error = "";

  async function initBoard() {
    try {
      const data = await LoadBoard("");
      boardData.set(data);
      isLoaded.set(true);
    } catch (e) {
      error = e.toString();
    }
  }

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
    <button on:click={initBoard}>Refresh</button>
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
