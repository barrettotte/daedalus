<!--
  Only renders the cards visible in the viewport.

  WHY THIS EXISTS:
  The board can have hundreds of cards per list. Rendering all of them as real DOM
  elements would be slow and use a lot of memory. Instead, this component only creates
  DOM elements for the ~10-15 cards you can actually see, plus a small buffer.

  HOW IT WORKS (step by step):
  1. We keep a "prefix sum" array where prefixSums[i] = total pixel height of items 0..i-1.
     This lets us instantly look up "where does item #N start?" without adding up heights.

  2. When the user scrolls, we binary search the prefix sums to find the first visible item.
     Binary search is O(log n) - fast even with thousands of items.

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
<script lang="ts">
  import { onMount, onDestroy, untrack } from "svelte";
  import type { Component } from "svelte";
  import type { daedalus } from "../../wailsjs/go/models";

  let { items = [], estimatedHeight = 90, component, listKey = "", focusIndex = -1 }: {
    items: daedalus.KanbanCard[];
    estimatedHeight?: number;
    component: Component<any>;
    listKey?: string;
    focusIndex?: number;
  } = $props();

  let container: HTMLDivElement | undefined = $state(undefined);
  let scrollTop = $state(0);
  let containerHeight = $state(800);

  // Plain objects - not reactive. measuredHeights is a cache; measureVersion is the reactive signal.
  let measuredHeights: Record<number, number> = {};  // card ID -> actual pixel height
  let itemElements: Record<number, HTMLDivElement> = {};  // card ID -> DOM element ref

  // Bumped when measurements change - the single reactive signal that invalidates prefixSums.
  let measureVersion = $state(0);

  // Returns measured height for an item, falling back to the estimate if not yet measured.
  function heightOf(index: number): number {
    const id = items[index]?.metadata?.id;
    if (id != null && id in measuredHeights) {
      return measuredHeights[id];
    }
    return estimatedHeight;
  }

  // prefixSums[i] = pixel offset where item i starts. Last entry = total height.
  // Example: heights [90, 100, 80] -> prefixSums = [0, 90, 190, 270]
  // Synchronous derived - no async cycle possible.
  let prefixSums = $derived.by(() => {
    void measureVersion; // re-derive when measurements update
    const sums = new Array(items.length + 1);
    sums[0] = 0;
    for (let i = 0; i < items.length; i++) {
      sums[i + 1] = sums[i] + heightOf(i);
    }
    return sums;
  });

  // Binary searches prefix sums to find visible items, then walks forward to the last one.
  function getVisibleRange(): { start: number; end: number } {
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
  function measureItems(): void {
    let changed = false;
    const currentVisible = untrack(() => visibleItems);

    for (const item of currentVisible) {
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
      measureVersion++;
    }

    // Clean up off-screen DOM refs
    const visibleIds = new Set(currentVisible.map(i => i.metadata.id));
    for (const id in itemElements) {
      if (!visibleIds.has(Number(id))) {
        delete itemElements[id];
      }
    }
  }

  // Throttles scroll events to one update per animation frame.
  let scrollRaf: number | null = null;
  function handleScroll(e: Event): void {
    if (scrollRaf) {
      return;
    }
    scrollRaf = requestAnimationFrame(() => {
      scrollTop = (e.target as HTMLDivElement).scrollTop;
      scrollRaf = null;
    });
  }

  let resizeObserver: ResizeObserver;

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

  let range = $derived(items.length > 0 ? getVisibleRange() : { start: 0, end: 0 });
  let visibleItems = $derived(items.slice(range.start, range.end));
  let topPadding = $derived(prefixSums[range.start] || 0);
  let totalHeight = $derived(prefixSums[items.length] || 0);

  // Measure items after render when visible items change
  $effect(() => {
    void visibleItems;
    measureItems();
  });

  // Scrolls the container so the focused item is visible.
  $effect(() => {
    if (focusIndex >= 0 && focusIndex < items.length && container) {
      const top = prefixSums[focusIndex];
      const bottom = prefixSums[focusIndex + 1];

      if (top < container.scrollTop) {
        container.scrollTop = top;
      } else if (bottom > container.scrollTop + containerHeight) {
        container.scrollTop = bottom - containerHeight;
      }
    }
  });
</script>

<div class="virtual-scroll-container" bind:this={container} onscroll={handleScroll}>
  <div class="scroll-wrapper" style="height: {totalHeight}px; padding-top: {topPadding}px">
    {#each visibleItems as item, i (item.metadata.id)}
      {@const CardComponent = component}
      {@const globalIndex = range.start + i}
      <div class="item-slot" data-card-id={item.metadata.id} bind:this={itemElements[item.metadata.id]}>
        <CardComponent card={item} {listKey} focused={focusIndex === globalIndex} />
      </div>
    {/each}
  </div>
</div>

<style lang="scss">
  .virtual-scroll-container {
    height: 100%;
    overflow-y: auto;
    contain: strict;
    padding-bottom: 6px;

    &::-webkit-scrollbar {
      width: 6px;
    }

    &::-webkit-scrollbar-track {
      background: transparent;
    }

    &::-webkit-scrollbar-thumb {
      background: rgba(255, 255, 255, 0.15);
      border-radius: 3px;
    }

    &::-webkit-scrollbar-thumb:hover {
      background: rgba(255, 255, 255, 0.25);
    }
  }

  .scroll-wrapper {
    box-sizing: border-box;
    padding-left: 6px;
  }

  .item-slot {
    padding: 0 0 6px 0;
    box-sizing: border-box;
  }
</style>
