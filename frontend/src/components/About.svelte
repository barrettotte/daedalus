<script lang="ts">
  // About modal showing app information - version, stack, and project links.
  import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";

  let { onclose }: { onclose: () => void } = $props();

  // Closes the modal when clicking the backdrop.
  function handleBackdropClick(e: MouseEvent): void {
    if (e.target === e.currentTarget) {
      onclose();
    }
  }

  const info = [
    { label: "Repository", value: "github.com/barrettotte/daedalus", href: "https://github.com/barrettotte/daedalus" },
    { label: "Version", value: "0.1.0" },
    { label: "Backend", value: "Go 1.23, Wails v2" },
    { label: "Frontend", value: "Svelte 5, TypeScript, SCSS" },
  ];
</script>

<div class="about-backdrop" role="presentation" onclick={handleBackdropClick}>
  <div class="about-modal" role="dialog">
    <div class="about-header">
      <h2 class="about-title">About</h2>
      <button class="about-close" onclick={onclose} title="Close">
        <svg viewBox="0 0 24 24" width="16" height="16">
          <line x1="18" y1="6" x2="6" y2="18" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          <line x1="6" y1="6" x2="18" y2="18" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
        </svg>
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
  .about-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: var(--overlay-backdrop);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 2000;
  }

  .about-modal {
    background: var(--color-bg-elevated);
    border-radius: 8px;
    max-width: 420px;
    width: 90%;
    color: var(--color-text-secondary);
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, sans-serif;
    text-align: left;
    user-select: text;
  }

  .about-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px 12px 20px;
    border-bottom: 1px solid var(--color-border);
  }

  .about-title {
    margin: 0;
    font-size: 1rem;
    font-weight: 600;
    color: var(--color-text-primary);
  }

  .about-close {
    background: var(--overlay-hover);
    border: none;
    color: var(--color-text-secondary);
    cursor: pointer;
    width: 32px;
    height: 32px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;

    &:hover {
      background: var(--overlay-hover-strong);
      color: var(--color-text-primary);
    }
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
