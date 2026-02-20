<script lang="ts">
  // Filter popover that shows available labels, icons, and URL domains from the board.
  // Clicking a value fires onselect with the search prefix string to insert.

  import type { BoardLists } from "../stores/board";
  import CardIcon from "./CardIcon.svelte";
  import { labelColor } from "../lib/utils";

  let {
    lists,
    colors,
    query,
    onselect,
  }: {
    lists: BoardLists;
    colors: Record<string, string>;
    query: string;
    onselect: (prefix: string) => void;
  } = $props();

  // Active tokens already in the search query, used to hide already-applied filters.
  let activeTokens = $derived(new Set(query.trim().split(/\s+/).filter(Boolean)));

  // Scan all cards to collect unique labels, icons, and URL domains, excluding already-applied filters.
  let availableLabels = $derived.by(() => {
    const set = new Set<string>();
    for (const cards of Object.values(lists)) {
      for (const card of cards) {
        for (const label of card.metadata.labels || []) {
          set.add(label);
        }
      }
    }
    return [...set].sort().filter(l => !activeTokens.has(`#${l}`));
  });

  let availableIcons = $derived.by(() => {
    const set = new Set<string>();
    for (const cards of Object.values(lists)) {
      for (const card of cards) {
        if (card.metadata.icon) {
          set.add(card.metadata.icon);
        }
      }
    }
    return [...set].sort().filter(i => !activeTokens.has(`icon:${i}`));
  });

  let availableDomains = $derived.by(() => {
    const set = new Set<string>();
    for (const cards of Object.values(lists)) {
      for (const card of cards) {
        if (card.metadata.url) {
          try {
            set.add(new URL(card.metadata.url).hostname);
          } catch {
            // skip malformed URLs
          }
        }
      }
    }
    return [...set].sort().filter(d => !activeTokens.has(`url:${d}`));
  });
</script>

<div class="filter-popover">
  {#if availableLabels.length > 0}
    <div class="filter-section">
      <div class="filter-section-title">Labels</div>
      <div class="filter-chips">
        {#each availableLabels as label}
          <button class="filter-chip label-chip" style="background: {labelColor(label, colors)}" onclick={() => onselect(`#${label}`)}>{label}</button>
        {/each}
      </div>
    </div>
  {/if}

  {#if availableIcons.length > 0}
    <div class="filter-section">
      <div class="filter-section-title">Icons</div>
      <div class="filter-chips">
        {#each availableIcons as icon}
          <button class="filter-chip icon-chip" onclick={() => onselect(`icon:${icon}`)}>
            <CardIcon name={icon} size={14} />
          </button>
        {/each}
      </div>
    </div>
  {/if}

  {#if availableDomains.length > 0}
    <div class="filter-section">
      <div class="filter-section-title">URL Domains</div>
      <div class="filter-chips">
        {#each availableDomains as domain}
          <button class="filter-chip domain-chip" onclick={() => onselect(`url:${domain}`)}>
            {domain}
          </button>
        {/each}
      </div>
    </div>
  {/if}

  {#if availableLabels.length === 0 && availableIcons.length === 0 && availableDomains.length === 0}
    <div class="filter-empty">No filterable fields on current cards</div>
  {/if}
</div>

<style lang="scss">
  .filter-popover {
    position: absolute;
    top: 100%;
    right: 0;
    margin-top: 4px;
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: 6px;
    padding: 8px;
    min-width: 200px;
    max-width: 320px;
    max-height: 300px;
    overflow-y: auto;
    box-shadow: var(--shadow-md);
    z-index: var(--z-dropdown);
  }

  .filter-section {
    margin-bottom: 8px;

    &:last-child {
      margin-bottom: 0;
    }
  }

  .filter-section-title {
    font-size: 0.65rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-text-muted);
    margin-bottom: 4px;
  }

  .filter-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }

  .filter-chip {
    all: unset;
    font-size: 0.7rem;
    padding: 2px 8px;
    border-radius: 4px;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }

  .label-chip {
    color: var(--color-text-inverse);
    font-weight: 600;

    &:hover {
      opacity: 0.85;
    }
  }

  .icon-chip {
    background: var(--overlay-hover);
    color: var(--color-text-primary);
    font-size: 0.85rem;
    padding: 3px 6px;

    &:hover {
      background: var(--overlay-hover-medium);
    }
  }

  .domain-chip {
    background: var(--overlay-hover);
    color: var(--color-text-secondary);
    font-family: var(--font-mono);

    &:hover {
      background: var(--overlay-hover-medium);
      color: var(--color-text-primary);
    }
  }

  .filter-empty {
    font-size: 0.75rem;
    color: var(--color-text-muted);
    padding: 4px 0;
  }
</style>
