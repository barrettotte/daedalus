<script lang="ts">
  // Shared icon picker widget for card sidebars. Shows the current icon with
  // edit/remove actions, or an "Add icon" button when no icon is set.

  import { boardPath } from "../stores/board";
  import { clickOutside } from "../lib/utils";
  import Icon from "./Icon.svelte";
  import CardIcon from "./CardIcon.svelte";
  import IconPicker from "./IconPicker.svelte";

  let {
    icon,
    onchange,
  }: {
    icon: string;
    onchange: (icon: string) => void;
  } = $props();

  let iconPickerOpen = $state(false);
</script>

{#if icon || iconPickerOpen}
  <div class="sidebar-section" use:clickOutside={() => { iconPickerOpen = false; }}>
    <div class="section-header">
      <h4 class="sidebar-title">Icon</h4>
      {#if icon && !iconPickerOpen}
        <button class="icon-current" title={`${$boardPath}/assets/icons/${icon}`} onclick={() => iconPickerOpen = true}>
          <CardIcon name={icon} size={14} />
        </button>
      {/if}
      <div class="section-header-actions">
        {#if iconPickerOpen}
          <button class="counter-header-btn save" title="Done" onclick={() => iconPickerOpen = false}>
            <Icon name="check" size={12} />
          </button>
        {:else if icon}
          <button class="counter-header-btn" title="Change icon" onclick={() => iconPickerOpen = true}>
            <Icon name="pencil" size={12} />
          </button>
        {/if}
        <button class="counter-header-btn remove" title="Remove icon" onclick={() => { iconPickerOpen = false; onchange(""); }}>
          <Icon name="trash" size={12} />
        </button>
      </div>
    </div>
    {#if iconPickerOpen}
      <IconPicker currentIcon={icon || ""} onselect={(name) => { iconPickerOpen = false; onchange(name); }}/>
    {/if}
  </div>
{:else}
  <div class="sidebar-section">
    <button class="add-counter-btn" onclick={() => iconPickerOpen = true}>+ Add icon</button>
  </div>
{/if}

<style lang="scss">
  .icon-current {
    all: unset;
    display: inline-flex;
    margin-left: 10px;
    margin-right: auto;
    color: var(--color-text-secondary);
    cursor: pointer;

    &:hover {
      color: var(--color-text-primary);
    }
  }
</style>
