<script lang="ts">
  // Sidebar widget for editing a card's icon. Selects from previously uploaded
  // file icons (SVG/PNG) managed by the backend.

  import { getIconNames } from "../lib/icons";
  import Icon from "./Icon.svelte";
  import CardIcon from "./CardIcon.svelte";

  let {
    icon,
    onchange,
  }: {
    icon: string;
    onchange: (icon: string) => void;
  } = $props();

  let editorOpen = $state(false);
  let iconFileNames: string[] = $state([]);

  function openEditor(): void {
    editorOpen = true;
    getIconNames().then(names => { iconFileNames = names; });
  }

  function closeEditor(): void {
    editorOpen = false;
  }

  function selectIcon(name: string): void {
    onchange(name);
    editorOpen = false;
  }

  function removeIcon(): void {
    onchange("");
    editorOpen = false;
  }
</script>

{#if icon && !editorOpen}
  <div class="sidebar-section">
    <div class="section-header">
      <h4 class="sidebar-title">Icon</h4>
      <span class="sidebar-inline-detail">
        <span class="sidebar-inline-sep">-</span>
        <button class="icon-display-btn" onclick={openEditor} title="Change icon">
          <CardIcon name={icon} size={14} />
        </button>
      </span>
      <div class="section-header-actions">
        <button class="counter-header-btn" title="Change icon" onclick={openEditor}>
          <Icon name="pencil" size={12} />
        </button>
        <button class="counter-header-btn remove" title="Remove icon" onclick={removeIcon}>
          <Icon name="trash" size={12} />
        </button>
      </div>
    </div>
  </div>
{:else if editorOpen}
  <div class="sidebar-section">
    <div class="section-header">
      <h4 class="sidebar-title">Icon</h4>
      <div class="section-header-actions">
        <button class="counter-header-btn save" title="Done" onclick={closeEditor}>
          <Icon name="check" size={12} />
        </button>
        <button class="counter-header-btn remove" title="Remove icon" onclick={removeIcon}>
          <Icon name="trash" size={12} />
        </button>
      </div>
    </div>
    {#if iconFileNames.length > 0}
      <div class="icon-editor-body">
        <div class="icon-grid">
          {#each iconFileNames as name}
            <button class="icon-grid-option" class:active={name === icon} title={name} onclick={() => selectIcon(name)}>
              <CardIcon name={name} size={16} />
            </button>
          {/each}
        </div>
      </div>
    {:else}
      <p class="no-icons-text">No icons available. Add SVG or PNG files to the board's icons folder.</p>
    {/if}
  </div>
{:else}
  <div class="sidebar-section">
    <button class="add-counter-btn" onclick={openEditor}>+ Add icon</button>
  </div>
{/if}

<style lang="scss">
  .icon-display-btn {
    all: unset;
    display: inline-flex;
    font-size: 1rem;
    line-height: 1;
    color: var(--color-text-secondary);
    cursor: pointer;
    border-radius: 4px;
    padding: 2px 4px;

    &:hover {
      background: var(--overlay-hover);
    }
  }

  .icon-editor-body {
    margin-top: 4px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .no-icons-text {
    font-size: 0.72rem;
    color: var(--color-text-muted);
    padding: 4px 0;
  }
</style>
