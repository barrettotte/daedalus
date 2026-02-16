<script lang="ts">
  // Shared checklist summary widget for card sidebars. Shows the checklist title,
  // done/total count, and remove/add actions. Pure display -- no local state.

  import type { daedalus } from "../../wailsjs/go/models";
  import Icon from "./Icon.svelte";

  let {
    checklist,
    title,
    onchange,
  }: {
    checklist: daedalus.CheckListItem[] | null;
    title?: string;
    onchange: (checklist: daedalus.CheckListItem[] | null) => void;
  } = $props();
</script>

{#if title || (checklist && checklist.length > 0)}
  {@const items = checklist || []}
  {@const done = items.filter(i => i.done).length}
  <div class="sidebar-section">
    <div class="section-header">
      <h4 class="sidebar-title">{title || "Checklist"}</h4>
      {#if items.length > 0}
        <span class="sidebar-inline-detail">
          <span class="sidebar-inline-sep">-</span>
          <span class="checklist-hint" class:all-done={done === items.length}>
            {done}/{items.length}
          </span>
        </span>
      {/if}
      <div class="section-header-actions">
        <button class="counter-header-btn remove" title="Remove checklist" onclick={() => onchange(null)}>
          <Icon name="trash" size={12} />
        </button>
      </div>
    </div>
  </div>
{:else}
  <div class="sidebar-section">
    <button class="add-counter-btn" title="Add a checklist" onclick={() => onchange([])}>+ Add checklist</button>
  </div>
{/if}

<style lang="scss">
  .checklist-hint {
    font-size: 0.8rem;
    color: var(--color-text-secondary);

    &.all-done {
      color: var(--color-success);
    }
  }
</style>
