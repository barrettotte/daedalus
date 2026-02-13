<script lang="ts">
  // Keyboard shortcut help overlay, toggled by the ? key.
  let { onclose }: { onclose: () => void } = $props();

  // Closes the overlay when clicking the backdrop.
  function handleBackdropClick(e: MouseEvent): void {
    if (e.target === e.currentTarget) {
      onclose();
    }
  }

  const sections = [
    { title: "Board Navigation", items: [
      { key: "Arrow Up / Down", desc: "Move focus to prev/next card" },
      { key: "Arrow Left / Right", desc: "Move focus to adjacent list" },
      { key: "Enter", desc: "Open focused card" },
      { key: "Escape", desc: "Clear focus" },
    ]},
    { title: "Card Actions", items: [
      { key: "E", desc: "Open card in edit mode" },
      { key: "Delete", desc: "Delete focused card (press twice)" },
    ]},
    { title: "Modal", items: [
      { key: "Arrow Up / Down", desc: "Navigate to prev/next card" },
      { key: "Arrow Left / Right", desc: "Navigate to adjacent list" },
      { key: "Escape", desc: "Close modal / cancel edit" },
    ]},
    { title: "General", items: [
      { key: "/", desc: "Focus search bar" },
      { key: "N", desc: "Create new card" },
      { key: "?", desc: "Toggle this help overlay" },
    ]},
  ];
</script>

<div class="help-backdrop" role="presentation" onclick={handleBackdropClick}>
  <div class="help-modal" role="dialog">
    <div class="help-header">
      <h2 class="help-title">Keyboard Shortcuts</h2>
      <button class="help-close" onclick={onclose} title="Close">
        <svg viewBox="0 0 24 24" width="16" height="16">
          <line x1="18" y1="6" x2="6" y2="18" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          <line x1="6" y1="6" x2="18" y2="18" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
        </svg>
      </button>
    </div>
    <div class="help-body">
      {#each sections as section, i}
        <h3 class="section-heading" class:first={i === 0}>{section.title}</h3>
        {#each section.items as item}
          <kbd class="key">{item.key}</kbd>
          <span class="desc">{item.desc}</span>
        {/each}
      {/each}
    </div>
  </div>
</div>

<style lang="scss">
  .help-backdrop {
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

  .help-modal {
    background: var(--color-bg-elevated);
    border-radius: 8px;
    max-width: 480px;
    width: 90%;
    color: #b6c2d1;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, sans-serif;
    text-align: left;
  }

  .help-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px 12px 20px;
    border-bottom: 1px solid var(--color-border);
  }

  .help-title {
    margin: 0;
    font-size: 1rem;
    font-weight: 600;
    color: var(--color-text-primary);
  }

  .help-close {
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
      color: #fff;
    }
  }

  .help-body {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 6px 10px;
    align-items: center;
    padding: 12px 20px 20px 20px;
    max-height: 60vh;
    overflow-y: auto;
  }

  .section-heading {
    grid-column: 1 / -1;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-text-primary);
    margin: 0;
    padding-top: 14px;
    padding-bottom: 4px;
    margin-top: 8px;
    border-top: 1px solid var(--color-border);

    &.first {
      padding-top: 0;
      margin-top: 0;
      border-top: none;
    }
  }

  .key {
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    border-radius: 4px;
    padding: 3px 8px;
    font-size: 0.78rem;
    font-family: monospace;
    color: var(--color-text-primary);
    white-space: nowrap;
    justify-self: start;
  }

  .desc {
    font-size: 0.85rem;
    color: var(--color-text-secondary);
  }
</style>
