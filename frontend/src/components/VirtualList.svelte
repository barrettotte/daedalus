<!--
  VirtualList.svelte — Only renders the cards visible in the viewport.

  WHY THIS EXISTS:
  The board can have hundreds of cards per list. Rendering all of them as real DOM
  elements would be slow and use a lot of memory. Instead, this component only creates
  DOM elements for the ~10-15 cards you can actually see, plus a small buffer.

  HOW IT WORKS (step by step):
  1. We keep a "prefix sum" array where prefixSums[i] = total pixel height of items 0..i-1.
     This lets us instantly look up "where does item #N start?" without adding up heights.

  2. When the user scrolls, we binary search the prefix sums to find the first visible item.
     Binary search is O(log n) — fast even with thousands of items.

  3. We render only those visible items (plus 2 extra above/below as buffer).

  4. The outer div is set to the TOTAL height of all items (so the scrollbar is correct),
     but only the visible slice is actually in the DOM. A padding-top on the inner wrapper
     pushes the visible items to the right scroll position.

  5. After each render, we measure the actual DOM heights of visible cards (they vary based
     on title length, labels, etc.) and store them. Next time we rebuild prefix sums, we
     use real heights instead of the estimate, so positioning gets more accurate over time.

  DIAGRAM:
    ┌─────────────────────────┐
    │  (empty space = topPad) │  <- padding-top pushes content down
    │                         │
    ├─────────────────────────┤  <- scrollTop (top of viewport)
    │  Card A  (visible)      │
    │  Card B  (visible)      │  <- only these items exist in the DOM
    │  Card C  (visible)      │
    ├─────────────────────────┤  <- scrollTop + containerHeight (bottom of viewport)
    │                         │
    │  (empty space)          │  <- totalHeight ensures correct scrollbar size
    └─────────────────────────┘
-->
<script>
  import { onMount, onDestroy, afterUpdate } from "svelte";

  export let items = [];
  export let estimatedHeight = 90;
  export let component;
  export let listKey = "";

  let container;
  let scrollTop = 0;
  let containerHeight = 800;

  let measuredHeights = {};  // card ID -> actual pixel height
  let itemElements = {};     // card ID -> DOM element ref

  // prefixSums[i] = pixel offset where item i starts. Last entry = total height.
  // Example: heights [90, 100, 80] -> prefixSums = [0, 90, 190, 270]
  let prefixSums = [];

  // Returns measured height for an item, falling back to the estimate if not yet measured.
  function heightOf(index) {
    const id = items[index]?.metadata?.id;
    return (id != null && measuredHeights[id]) || estimatedHeight;
  }

  // Rebuilds the prefix sum array from all item heights.
  function buildPrefixSums() {
    prefixSums = new Array(items.length + 1);
    prefixSums[0] = 0;

    for (let i = 0; i < items.length; i++) {
      prefixSums[i + 1] = prefixSums[i] + heightOf(i);
    }
  }

  // Binary searches prefix sums to find visible items, then walks forward to the last one.
  function getVisibleRange() {
    let lo = 0, hi = items.length;
    while (lo < hi) {
      const mid = (lo + hi) >>> 1;

      if (prefixSums[mid + 1] <= scrollTop) {
        lo = mid + 1;
      } else {
        hi = mid;
      }
    }

    let start = lo;
    let end = start;
    const bottom = scrollTop + containerHeight;
    while (end < items.length && prefixSums[end] < bottom) {
      end++;
    }

    // Buffer 2 extra items above/below to prevent flicker during fast scrolling
    start = Math.max(0, start - 2);
    end = Math.min(items.length, end + 2);
    return { start, end };
  }

  // Measures actual DOM heights of visible items so prefix sums become accurate over time.
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
      measuredHeights = measuredHeights; // reassign to trigger Svelte reactivity
    }

    // Clean up off-screen DOM refs
    const visibleIds = new Set(visibleItems.map(i => i.metadata.id));
    for (const id in itemElements) {
      if (!visibleIds.has(Number(id))) {
        delete itemElements[id];
      }
    }
  }

  // Throttles scroll events to one update per animation frame.
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

  let resizeObserver;

  onMount(() => {
    if (container) {
      containerHeight = container.offsetHeight;
      resizeObserver = new ResizeObserver(entries => {
        for (const entry of entries) {
          containerHeight = entry.contentRect.height;
        }
      });
      resizeObserver.observe(container);
    }
  });

  onDestroy(() => {
    if (resizeObserver) {
      resizeObserver.disconnect();
    }
    if (scrollRaf) {
      cancelAnimationFrame(scrollRaf);
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
      <div class="item-slot" data-card-id={item.metadata.id} bind:this={itemElements[item.metadata.id]}>
        <svelte:component this={component} card={item} {listKey} />
      </div>
    {/each}
  </div>
</div>

<style>
  .virtual-scroll-container {
    height: 100%;
    overflow-y: auto;
    contain: strict;
  }
  .scroll-wrapper {
    box-sizing: border-box;
  }
  .item-slot {
    padding: 0 0 6px 0;
    box-sizing: border-box;
  }
</style>
