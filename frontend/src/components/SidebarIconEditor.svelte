<script lang="ts">
  // Sidebar widget for editing a card's icon. Supports freeform emoji input
  // or selecting a previously uploaded file icon by name.

  import { getIconNames } from "../lib/icons";
  import { isFileIcon as checkFileIcon } from "../lib/utils";
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

  let isFileIcon = $derived(checkFileIcon(icon));

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
        <span class="sidebar-inline-sep">-</span>
        <button class="icon-display-btn" onclick={openEditor} title="Change icon">
          {#if isFileIcon}
            <CardIcon name={icon} size={14} />
          {:else}
            {icon}
          {/if}
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
    <div class="icon-editor-body">
      <div class="emoji-row">
        <input class="form-input emoji-input" type="text" placeholder="Type emoji..." bind:value={emojiValue} onkeydown={e => e.key === 'Enter' && commitEmoji()}/>
        <button class="emoji-save-btn" onclick={commitEmoji}>Set</button>
      </div>
      {#if iconFileNames.length > 0}
        <div class="icon-grid">
          {#each iconFileNames as name}
            <button class="icon-grid-option" class:active={name === icon} title={name} onclick={() => selectIcon(name)}>
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
</style>
