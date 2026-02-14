<script lang="ts">
  import type { daedalus } from "../../wailsjs/go/models";

  let { checklist, ontoggle }: {
    checklist: daedalus.CheckListItem[];
    ontoggle: (idx: number) => void;
  } = $props();

  // Number of completed checklist items.
  let checkedCount = $derived(checklist.filter(i => i.done).length);

  // Completion percentage for the progress bar.
  let checkPct = $derived(checklist.length > 0 ? (checkedCount / checklist.length) * 100 : 0);

  // Whether all checklist items are done.
  let allDone = $derived(checklist.length > 0 && checkedCount === checklist.length);

  // Whether the checklist items are visible or collapsed.
  let expanded = $state(true);

  // Returns true if the string is a URL.
  function isUrl(str: string): boolean {
    return /^https?:\/\/\S+$/.test(str);
  }

</script>

<button class="section-header" title="Toggle checklist" onclick={() => expanded = !expanded}>
  <svg class="chevron" class:collapsed={!expanded} viewBox="0 0 24 24">
    <polyline points="6 9 12 15 18 9" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
  </svg>
  <svg class="section-icon" viewBox="0 0 24 24">
    <polyline points="9 11 12 14 22 4" fill="none" stroke="currentColor" stroke-width="2"/>
    <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" fill="none" stroke="currentColor" stroke-width="2"/>
  </svg>
  <h3 class="section-title">Checklist</h3>
  <div class="checklist-bar">
    <div class="progress-fill" class:complete={checkPct === 100} style="width: {checkPct}%"></div>
  </div>
  <span class="checklist-count" class:all-done={allDone}>{checkedCount}/{checklist.length}</span>
</button>
{#if expanded}
<ul class="checklist">
  {#each checklist as item, idx}
    <li class:done={item.done}>
      <button class="checkbox-btn" title="Toggle item" onclick={() => ontoggle(idx)}>
        <span class="checkbox" class:checked={item.done}>
          {#if item.done}
            <svg viewBox="0 0 16 16">
              <rect x="1" y="1" width="14" height="14" rx="2" fill="currentColor"/>
              <polyline points="4 8 7 11 12 5" fill="none" stroke="#22252b" stroke-width="2"/>
            </svg>
          {:else}
            <svg viewBox="0 0 16 16">
              <rect x="1" y="1" width="14" height="14" rx="2" fill="none" stroke="currentColor" stroke-width="1.5"/>
            </svg>
          {/if}
        </span>
      </button>
      <span class="check-text">
        {#if isUrl(item.desc)}
          <a href={item.desc} target="_blank" rel="noopener noreferrer" onclick={(e: MouseEvent) => e.stopPropagation()}>{item.desc}</a>
        {:else}
          {item.desc}
        {/if}
      </span>
    </li>
  {/each}
</ul>
{/if}

<style lang="scss">
  .section-header {
    all: unset;
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
    width: 100%;
    cursor: pointer;
    border-radius: 4px;
    padding: 4px 0;
    box-sizing: border-box;

    &:hover {
      background: var(--overlay-hover-faint);
    }
  }

  .chevron {
    width: 16px;
    height: 16px;
    color: var(--color-text-muted);
    flex-shrink: 0;
    transition: transform 0.15s;

    &.collapsed {
      transform: rotate(-90deg);
    }
  }

  .section-icon {
    width: 20px;
    height: 20px;
    color: var(--color-text-secondary);
    flex-shrink: 0;
  }

  .section-title {
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--color-text-primary);
    margin: 0;
  }

  .checklist-bar {
    flex: 1;
    height: 6px;
    background: var(--color-border);
    border-radius: 3px;
    overflow: hidden;
    margin: 0 8px;
  }

  .checklist-count {
    font-size: 0.75rem;
    color: var(--color-text-tertiary);
    flex-shrink: 0;

    &.all-done {
      color: var(--color-success);
    }
  }

  .progress-fill {
    height: 100%;
    background: var(--color-accent);
    border-radius: 4px;
    transition: width 0.3s;

    &.complete {
      background: var(--color-success);
    }
  }

  .checklist {
    list-style: none;
    padding: 0;
    margin: 0;
    max-height: 400px;
    overflow-y: auto;

    li {
      padding: 6px 8px;
      font-size: 0.85rem;
      display: flex;
      gap: 8px;
      align-items: flex-start;
      border-radius: 4px;

      &:hover {
        background: var(--overlay-hover-faint);
      }

      &.done .check-text {
        text-decoration: line-through;
        color: var(--color-text-muted);
      }
    }
  }

  .checkbox-btn {
    all: unset;
    cursor: pointer;
    display: flex;
    align-items: center;
    flex-shrink: 0;
  }

  .checkbox {
    flex-shrink: 0;
    width: 16px;
    height: 16px;
    margin-top: 1px;
    color: var(--color-text-secondary);

    &.checked {
      color: var(--color-accent);
    }

    svg {
      width: 16px;
      height: 16px;
    }
  }

  .check-text {
    line-height: 1.3;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;

    a {
      color: var(--color-accent);
      text-decoration: none;
      line-height: inherit;
      display: inline;

      &:hover {
        text-decoration: underline;
      }
    }
  }
</style>
