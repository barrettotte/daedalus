<script lang="ts">
  // Shared estimate editor widget for card sidebars. Toggles between display
  // mode (clickable value) and edit mode (number input with auto-focus).

  import { autoFocus } from "../lib/utils";
  import Icon from "./Icon.svelte";

  let {
    estimate,
    onchange,
  }: {
    estimate: number | null;
    onchange: (estimate: number | null) => void;
  } = $props();

  let editingEstimate = $state(false);
  let estimateInput = $state("");

  function startEditEstimate(): void {
    estimateInput = estimate != null ? String(estimate) : "";
    editingEstimate = true;
  }

  function blurEstimate(): void {
    editingEstimate = false;
    const val = parseFloat(estimateInput);
    if (isNaN(val) || val <= 0) {
      if (estimate != null) {
        onchange(null);
      }
    } else if (val !== estimate) {
      onchange(val);
    }
  }
</script>

{#if estimate != null || editingEstimate}
  <div class="sidebar-section">
    <div class="section-header">
      <h4 class="sidebar-title">Estimate</h4>
      {#if editingEstimate}
        <span class="sidebar-inline-detail">
          <span class="sidebar-inline-sep">-</span>
          <input class="estimate-input" type="number" step="0.5" min="0"
            bind:value={estimateInput} onblur={blurEstimate} onkeydown={e => e.key === 'Enter' && (e.target as HTMLInputElement).blur()} use:autoFocus/>
        </span>
        <div class="section-header-actions">
          <button class="counter-header-btn save" title="Confirm" onclick={() => (document.querySelector('.estimate-input') as HTMLInputElement)?.blur()}>
            <Icon name="check" size={12} />
          </button>
          <button class="counter-header-btn remove" title="Remove estimate" onclick={() => onchange(null)}>
            <Icon name="trash" size={12} />
          </button>
        </div>
      {:else}
        <span class="sidebar-inline-detail">
          <span class="sidebar-inline-sep">-</span>
          <button class="estimate-value" onclick={startEditEstimate}>{estimate}h</button>
        </span>
        <div class="section-header-actions">
          <button class="counter-header-btn" title="Edit estimate" onclick={startEditEstimate}>
            <Icon name="pencil" size={12} />
          </button>
          <button class="counter-header-btn remove" title="Remove estimate" onclick={() => onchange(null)}>
            <Icon name="trash" size={12} />
          </button>
        </div>
      {/if}
    </div>
  </div>
{:else}
  <div class="sidebar-section">
    <button class="add-counter-btn" onclick={startEditEstimate}>+ Add estimate</button>
  </div>
{/if}

<style lang="scss">
  .estimate-value {
    all: unset;
    font-family: var(--font-mono);
    font-size: 0.8rem;
    color: var(--color-text-secondary);
    cursor: pointer;

    &:hover {
      color: var(--color-text-primary);
    }
  }

  .estimate-input {
    width: 50px;
    background: var(--color-bg-base);
    border: 1px solid var(--color-accent);
    color: var(--color-text-primary);
    font-family: var(--font-mono);
    font-size: 0.8rem;
    padding: 1px 4px;
    border-radius: 4px;
    outline: none;
    box-sizing: border-box;
  }
</style>
