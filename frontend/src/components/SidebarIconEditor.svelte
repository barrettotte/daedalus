<script lang="ts">
  // Sidebar widget for editing a card's icon. Supports freeform emoji input
  // or selecting a previously uploaded file icon by name.

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
  let emojiValue = $state("");
  let iconFileNames: string[] = $state([]);

  let isFileIcon = $derived(icon ? icon.endsWith(".svg") || icon.endsWith(".png") : false);

  function openEditor(): void {
    emojiValue = isFileIcon ? "" : icon;
    editorOpen = true;
    getIconNames().then(names => { iconFileNames = names; });
  }

  function closeEditor(): void {
    editorOpen = false;
  }

  function commitEmoji(): void {
    const trimmed = emojiValue.trim();
    if (trimmed !== icon) {
      onchange(trimmed);
    }
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
        {#if isFileIcon}
          <span class="icon-display"><CardIcon name={icon} size={14} /></span>
        {:else}
          <span class="icon-display">{icon}</span>
        {/if}
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
    <div class="icon-editor-body">
      <div class="emoji-row">
        <input class="emoji-input" type="text" placeholder="Type emoji..." bind:value={emojiValue} onkeydown={e => e.key === 'Enter' && commitEmoji()}/>
        <button class="emoji-save-btn" onclick={commitEmoji}>Set</button>
      </div>
      {#if iconFileNames.length > 0}
        <div class="icon-grid">
          {#each iconFileNames as name}
            <button class="icon-option" class:active={name === icon} title={name} onclick={() => selectIcon(name)}>
              <CardIcon name={name} size={16} />
            </button>
          {/each}
        </div>
      {/if}
    </div>
  </div>
{:else}
  <div class="sidebar-section">
    <button class="add-counter-btn" onclick={openEditor}>+ Add icon</button>
  </div>
{/if}

<style lang="scss">
  .sidebar-inline-detail {
    margin-left: 8px;
  }

  .icon-display {
    display: inline-flex;
    font-size: 1rem;
    line-height: 1;
    color: var(--color-text-secondary);
  }

  .icon-editor-body {
    margin-top: 4px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .emoji-row {
    display: flex;
    gap: 4px;
  }

  .emoji-input {
    flex: 1;
    min-width: 0;
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    color: var(--color-text-primary);
    font-size: 0.8rem;
    padding: 3px 6px;
    border-radius: 4px;
    outline: none;

    &:focus {
      border-color: var(--color-accent);
    }
  }

  .emoji-save-btn {
    all: unset;
    font-size: 0.68rem;
    font-weight: 600;
    color: var(--color-text-secondary);
    background: var(--overlay-hover-light);
    border: 1px solid var(--color-border);
    border-radius: 4px;
    padding: 2px 8px;
    cursor: pointer;

    &:hover {
      background: var(--overlay-hover-medium);
      color: var(--color-text-primary);
    }
  }

  .icon-grid {
    display: grid;
    grid-template-columns: repeat(5, 1fr);
    gap: 2px;
  }

  .icon-option {
    all: unset;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 6px;
    border-radius: 4px;
    color: var(--color-text-tertiary);
    cursor: pointer;

    &:hover {
      background: var(--overlay-hover);
      color: var(--color-text-primary);
    }

    &.active {
      background: var(--overlay-hover-medium);
      color: var(--color-accent);
    }
  }
</style>
