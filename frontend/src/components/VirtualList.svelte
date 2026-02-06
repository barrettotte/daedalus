<script>
  import { onMount, afterUpdate } from "svelte";

  export let items = [];
  export let estimatedHeight = 90;
  export let component;

  let container;
  let scrollTop = 0;
  let containerHeight = 800;

  let measuredHeights = {};
  let itemElements = {};

  // Prefix sums - the total of item heights
  // prefixsums[i] is the pixel offset where item i starts
  let prefixSums = [];

  // Returns measured height for an item, falling back to the estimate if not yet measured.
  function heightOf(index) {
    const id = items[index]?.metadata?.id;
    return (id != null && measuredHeights[id]) || estimatedHeight;
  }

  // Rebuilds the prefix sum array from item heights for O(1) offset lookups.
  function buildPrefixSums() {
    prefixSums = new Array(items.length + 1);
    prefixSums[0] = 0;

    for (let i = 0; i < items.length; i++) {
      prefixSums[i + 1] = prefixSums[i] + heightOf(i);
    }
  }

  // Binary searches the prefix sums to find which items are visible at the current scroll position.
  function getVisibleRange() {
    let lo = 0, hi = items.length;
    while (lo < hi) {
      const mid = (lo + hi) >>> 1;

      if (prefixSums[mid + 1] <= scrollTop){
        lo = mid + 1;
      }
      else {
        hi = mid;
      }
    }

    let start = lo;
    let end = start;
    const bottom = scrollTop + containerHeight;
    while (end < items.length && prefixSums[end] < bottom) {
      end++;
    }

    // Buffer a couple extra items above and below to prevent flickering
    start = Math.max(0, start - 2);
    end = Math.min(items.length, end + 2);
    return { start, end };
  }

  // Reads actual DOM heights of visible items and cleans up refs for off-screen items.
  function measureItems() {
    let changed = false;

    for (const item of visibleItems) {
      const id = item.metadata.id;
      const el = itemElements[id];

      if (el) {
        const h = el.offsetHeight;

        if (h > 0 && measuredHeights[id] !== h) {
          measuredHeights[id] = h;
          changed = true;
        }
      }
    }
    if (changed) {
      measuredHeights = measuredHeights;
    }

    // clean up off-screen refs
    const visibleIds = new Set(visibleItems.map(i => i.metadata.id));
    for (const id in itemElements) {
      if (!visibleIds.has(Number(id))) {
        delete itemElements[id];
      }
    }
  }

  // Throttles scroll events to one update per animation frame to avoid redundant recalculations.
  let scrollRaf = null;
  function handleScroll(e) {
    if (scrollRaf) {
      return;
    }
    scrollRaf = requestAnimationFrame(() => {
      scrollTop = e.target.scrollTop;
      scrollRaf = null;
    });
  }

  onMount(() => {
    if (container) {
      containerHeight = container.offsetHeight;
    }
  });

  afterUpdate(measureItems);

  $: if (items || measuredHeights) buildPrefixSums();
  $: range = items.length > 0 ? getVisibleRange(scrollTop, containerHeight, prefixSums) : { start: 0, end: 0 };
  $: visibleItems = items.slice(range.start, range.end);
  $: topPadding = prefixSums[range.start] || 0;
  $: totalHeight = prefixSums[items.length] || 0;

</script>

<div class="virtual-scroll-container" bind:this={container} on:scroll={handleScroll}>
  <div class="scroll-wrapper" style="height: {totalHeight}px; padding-top: {topPadding}px">
    {#each visibleItems as item (item.metadata.id)}
      <div class="item-slot" bind:this={itemElements[item.metadata.id]}>
        <svelte:component this={component} card={item} />
      </div>
    {/each}
  </div>
</div>

<style>
  .virtual-scroll-container {
    height: 100%;
    overflow-y: auto;
  }
  .scroll-wrapper {
    box-sizing: border-box;
  }
  .item-slot {
    padding: 0 0 6px 0;
    box-sizing: border-box;
  }
</style>
