<script lang="ts">
  // Shared label editor widget for card sidebars. Displays current labels as
  // removable chips and provides a dropdown to add labels from the board registry.

  import { labelColors } from "../stores/board";
  import { labelColor, clickOutside } from "../lib/utils";
  import Icon from "./Icon.svelte";

  let {
    labels,
    onchange,
  }: {
    labels: string[];
    onchange: (labels: string[]) => void;
  } = $props();

  let labelDropdownOpen = $state(false);

  // Board labels not currently on this card, sorted alphabetically.
  let availableLabels = $derived.by(() => {
    const current = new Set(labels);
    return Object.keys($labelColors).filter(l => !current.has(l)).sort();
  });

  function removeLabel(label: string): void {
    onchange(labels.filter(l => l !== label));
  }

  function addLabel(label: string): void {
    onchange([...labels, label]);
  }
</script>

{#if labels.length > 0}
  <div class="sidebar-section label-section" use:clickOutside={() => { labelDropdownOpen = false; }}>
    <div class="section-header">
      <h4 class="sidebar-title">Labels</h4>
      <div class="section-header-actions">
        {#if availableLabels.length > 0}
          <button class="counter-header-btn" title="Add label" onclick={() => labelDropdownOpen = !labelDropdownOpen}>
            <Icon name="plus" size={12} />
          </button>
        {/if}
        <button class="counter-header-btn remove" title="Remove all labels" onclick={() => onchange([])}>
          <Icon name="trash" size={12} />
        </button>
      </div>
    </div>
    <div class="sidebar-labels">
      {#each [...labels].sort() as label}
        <button class="label label-removable" title="Remove {label}"
          style="background: {labelColor(label, $labelColors)}"
          onclick={() => removeLabel(label)}
        >
          {label}
        </button>
      {/each}
    </div>
    {#if labelDropdownOpen && availableLabels.length > 0}
      <div class="dropdown-menu">
        {#each availableLabels as label}
          <button class="dropdown-option" onclick={() => addLabel(label)}>
            <span class="label-add-swatch" style="background: {$labelColors[label]}"></span>
            {label}
          </button>
        {/each}
      </div>
    {/if}
  </div>
{:else}
  <div class="sidebar-section" use:clickOutside={() => { labelDropdownOpen = false; }}>
    {#if availableLabels.length > 0}
      <div class="label-add-wrapper">
        <button class="add-counter-btn" onclick={() => labelDropdownOpen = !labelDropdownOpen}>+ Add label</button>
        {#if labelDropdownOpen}
          <div class="dropdown-menu">
            {#each availableLabels as label}
              <button class="dropdown-option" onclick={() => addLabel(label)}>
                <span class="label-add-swatch" style="background: {$labelColors[label]}"></span>
                {label}
              </button>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </div>
{/if}

<style lang="scss">
  .label-section {
    position: relative;
  }

  .sidebar-labels {
    display: flex;
    gap: 4px;
    flex-wrap: wrap;
    margin-top: 2px;
  }

  .label {
    font-size: 0.7rem;
    font-weight: 600;
    padding: 4px 8px;
    border-radius: 3px;
    color: var(--color-text-inverse);
    text-align: center;
    flex: 0 0 calc(50% - 2px);
    box-sizing: border-box;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .label-removable {
    cursor: pointer;
    border: none;
    text-align: center;

    &:hover {
      opacity: 0.7;
    }
  }

  .label-add-wrapper {
    position: relative;
  }

  .label-section > :global(.dropdown-menu) {
    top: auto;
    margin-top: 4px;
    position: relative;
  }

  .label-section :global(.dropdown-option),
  .label-add-wrapper :global(.dropdown-option) {
    gap: 6px;
  }

  .label-add-swatch {
    width: 10px;
    height: 10px;
    border-radius: 2px;
    flex-shrink: 0;
  }
</style>
