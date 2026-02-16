<script lang="ts">
  // Shared label editor widget for card sidebars. Displays current labels as
  // removable chips and provides a dropdown to add labels from the board registry.

  import { labelColors } from "../stores/board";
  import { labelColor, clickOutside } from "../lib/utils";

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

<div class="sidebar-section" use:clickOutside={() => { labelDropdownOpen = false; }}>
  <h4 class="sidebar-title">Labels</h4>
  {#if labels.length > 0}
    <div class="sidebar-labels">
      {#each [...labels].sort() as label}
        <button class="label label-removable" title="Remove {label}" style="background: {labelColor(label, $labelColors)}" onclick={() => removeLabel(label)}>
          {label}
        </button>
      {/each}
    </div>
  {/if}
  {#if availableLabels.length > 0}
    <div class="label-add-wrapper">
      <button class="add-counter-btn" onclick={() => labelDropdownOpen = !labelDropdownOpen}>+ Add label</button>
      {#if labelDropdownOpen}
        <div class="label-add-menu">
          {#each availableLabels as label}
            <button class="label-add-option" onclick={() => addLabel(label)}>
              <span class="label-add-swatch" style="background: {$labelColors[label]}"></span>
              {label}
            </button>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>

<style lang="scss">
  .sidebar-labels {
    display: flex;
    gap: 4px;
    flex-wrap: wrap;
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
    margin-top: 4px;
  }

  .label-add-menu {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    right: 0;
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: 4px;
    padding: 4px 0;
    z-index: 10;
    max-height: 200px;
    overflow-y: auto;
  }

  .label-add-option {
    all: unset;
    display: flex;
    align-items: center;
    gap: 6px;
    width: 100%;
    padding: 5px 8px;
    font-size: 0.8rem;
    color: var(--color-text-primary);
    cursor: pointer;
    box-sizing: border-box;

    &:hover {
      background: var(--overlay-hover);
    }
  }

  .label-add-swatch {
    width: 10px;
    height: 10px;
    border-radius: 2px;
    flex-shrink: 0;
  }
</style>
