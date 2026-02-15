<script lang="ts">
  // Renders a user-uploaded icon (SVG or PNG) by filename.
  // Loads content from the icon cache module asynchronously.

  import { getIconContent } from "../lib/icons";

  let { name, size = 16 }: { name: string; size?: number } = $props();

  let content = $state("");

  $effect(() => {
    const currentName = name;
    if (!currentName) {
      content = "";
      return;
    }
    let stale = false;

    getIconContent(currentName).then(result => {
      if (!stale) {
        content = result;
      }
    }).catch(() => {
      if (!stale) {
        content = "";
      }
    });

    return () => { stale = true; };
  });
</script>

{#if content.includes("<svg")}
  <span class="card-icon svg-icon" style="width: {size}px; height: {size}px;">{@html content}</span>
{:else if content.startsWith("data:")}
  <img class="card-icon" src={content} alt={name} width={size} height={size}/>
{/if}

<style>
  .card-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    vertical-align: middle;
    flex-shrink: 0;
  }

  .svg-icon :global(svg) {
    width: 100%;
    height: 100%;
    display: block;
  }

  img.card-icon {
    object-fit: contain;
  }
</style>
