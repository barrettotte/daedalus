<script lang="ts">
  // About modal showing app information - version, stack, and project links.

  import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";
  import { GetVersion } from "../../wailsjs/go/main/App";
  import Icon from "./Icon.svelte";
  import { backdropClose } from "../lib/utils";

  let { onclose }: { onclose: () => void } = $props();

  let appVersion = $state("...");
  GetVersion().then((v) => { appVersion = v; });

  const info = $derived([
    { label: "Repository", value: "github.com/barrettotte/daedalus", href: "https://github.com/barrettotte/daedalus" },
    { label: "Version", value: appVersion },
    { label: "Backend", value: "Go 1.23, Wails v2" },
    { label: "Frontend", value: "Svelte 5, TypeScript, SCSS" },
  ]);
</script>

<div class="modal-backdrop centered z-high" role="presentation" use:backdropClose={onclose}>
  <div class="modal-dialog size-sm about-modal" role="dialog">
    <div class="modal-header">
      <h2 class="modal-title">About</h2>
      <button class="modal-close" onclick={onclose} title="Close">
        <Icon name="close" size={16} />
      </button>
    </div>
    <div class="about-body">
      {#each info as item}
        <span class="about-label">{item.label}</span>
        {#if item.href}
          <button class="about-value about-link" onclick={() => BrowserOpenURL(item.href!)}>{item.value}</button>
        {:else}
          <span class="about-value">{item.value}</span>
        {/if}
      {/each}
    </div>
  </div>
</div>

<style lang="scss">
  .about-modal {
    user-select: text;
  }

  .about-body {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 8px 16px;
    align-items: baseline;
    padding: 16px 20px 20px 20px;
  }

  .about-label {
    font-size: 0.78rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-text-muted);
    white-space: nowrap;
  }

  .about-value {
    font-size: 0.88rem;
    color: var(--color-text-secondary);
  }

  .about-link {
    all: unset;
    color: var(--color-accent);
    cursor: pointer;
    font-size: 0.88rem;

    &:hover {
      text-decoration: underline;
    }
  }
</style>
