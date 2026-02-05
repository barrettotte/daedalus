<script>
  import { onMount } from "svelte";

  export let items = [];
  export let itemHeight = 100;
  export let component;

  let container;
  let scrollTop = 0;
  let containerHeight = 800;

  $: totalHeight = items.length * itemHeight;
  $: startIndex = Math.floor(scrollTop / itemHeight);
  $: endIndex = Math.min(
    items.length,
    Math.floor((scrollTop + containerHeight) / itemHeight) + 2,
  );
  $: visibleItems = items.slice(startIndex, endIndex);
  $: topPadding = startIndex * itemHeight;
  $: bottomPadding = (items.length - endIndex) * itemHeight;

  function handleScroll(e) {
    scrollTop = e.target.scrollTop;
  }

  onMount(() => {
    if (container) {
      containerHeight = container.offsetHeight;
    }
  });
</script>

<div
  class="virtual-scroll-container"
  bind:this={container}
  on:scroll={handleScroll}
>
  <div
    class="scroll-wrapper"
    style="height: {totalHeight}px; padding-top: {topPadding}px"
  >
    {#each visibleItems as item (item.metadata.id)}
      <div style="height: {itemHeight}px;">
        <svelte:component this={component} card={item} />
      </div>
    {/each}
  </div>
</div>

<style>
  .virtual-scroll-container {
    height: 100%;
    overflow-y: auto;
    scrollbar-width: thin;
    scrollbar-color: #555 #222;
  }
  .scroll-wrapper {
    box-sizing: border-box;
  }
</style>
