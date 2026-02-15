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
      { key: "#", desc: "Jump to card by ID (#123)" },
      { key: "N", desc: "Create new card" },
      { key: "?", desc: "Toggle this help overlay" },
    ]},
  ];
</script>

<div class="modal-backdrop centered z-high" role="presentation" onclick={handleBackdropClick}>
  <div class="modal-dialog size-md" role="dialog">
    <div class="modal-header">
      <h2 class="modal-title">Keyboard Shortcuts</h2>
      <button class="modal-close" onclick={onclose} title="Close">
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
