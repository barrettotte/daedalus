<script>
  import { onMount, onDestroy } from "svelte";
  import { GetMetrics } from "../../wailsjs/go/main/App";
  import { showMetrics, boardData, addToast } from "../stores/board.js";

  let metrics = null;
  let interval;
  let fps = 0;
  let domNodes = 0;

  // FPS track via requestAnimationFrame
  let frameCount = 0;
  let lastFpsTime = performance.now();
  let rafId;

  // Increments the frame counter each animation frame and calculates FPS every second.
  function countFrame() {
    frameCount++;
    const now = performance.now();

    if (now - lastFpsTime >= 1000) {
      fps = frameCount;
      frameCount = 0;
      lastFpsTime = now;
    }
    rafId = requestAnimationFrame(countFrame);
  }

  // Samples frontend metrics like DOM node count.
  function updateFrontendMetrics() {
    domNodes = document.querySelectorAll('*').length;
  }

  // Fetches backend metrics from Go and updates frontend metrics.
  async function fetchMetrics() {
    try {
      metrics = await GetMetrics();
      updateFrontendMetrics();
    } catch (e) {
      addToast(`Failed to fetch metrics: ${e}`);
    }
  }

  onMount(() => {
    fetchMetrics();
    interval = setInterval(fetchMetrics, 2000);
    rafId = requestAnimationFrame(countFrame);
  });

  onDestroy(() => {
    clearInterval(interval);
    cancelAnimationFrame(rafId);
  });

  // Refresh metrics when board data changes
  $: $boardData, fetchMetrics();

</script>

{#if $showMetrics && metrics}
  <div class="metrics-overlay">
    <div class="metrics-row"><span class="label" title="Frames per second">FPS</span><span>{fps}</span></div>
    <div class="metrics-row"><span class="label" title="Total HTML elements in document">DOM nodes</span><span>{domNodes}</span></div>
    <div class="metrics-divider"></div>
    <div class="metrics-row"><span class="label" title="Go heap memory allocated for live objects">Go heap</span><span>{metrics.heapAlloc.toFixed(1)} MB</span></div>
    <div class="metrics-row"><span class="label" title="Total memory obtained from OS by Go runtime">Go sys</span><span>{metrics.sys.toFixed(1)} MB</span></div>
    <div class="metrics-row"><span class="label" title="Number of completed garbage collection cycles">GC</span><span>{metrics.numGC}</span></div>
    <div class="metrics-row"><span class="label" title="Active goroutines">Goroutines</span><span>{metrics.goroutines}</span></div>
    <div class="metrics-divider"></div>
    <div class="metrics-row"><span class="label" title="Number of lists">Lists</span><span>{metrics.numLists}</span></div>
    <div class="metrics-row"><span class="label" title="Number of cards">Cards</span><span>{metrics.numCards}</span></div>
    <div class="metrics-row"><span class="label" title="Highest card ID">Max ID</span><span>{metrics.maxID}</span></div>
    <div class="metrics-row"><span class="label" title="Total size of all markdown cards">MD size</span><span>{metrics.fileSizeMB.toFixed(1)} MB</span></div>
  </div>
{/if}

<style>
  .metrics-overlay {
    position: fixed;
    bottom: 12px;
    right: 12px;
    background: #181a1f;
    border: 1px solid #333;
    border-radius: 6px;
    padding: 8px 12px;
    font-size: 0.75rem;
    font-family: monospace;
    z-index: 100;
    min-width: 150px;
  }

  .metrics-row {
    display: flex;
    justify-content: space-between;
    gap: 16px;
    padding: 1px 0;
  }

  .label {
    color: #888;
  }

  .metrics-divider {
    border-top: 1px solid #333;
    margin: 3px 0;
  }
</style>
