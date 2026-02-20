<script lang="ts">
  // Modal showing board statistics bar charts.

  import { boardData, boardConfig, labelColors, listOrder } from "../stores/board";
  import { backdropClose, getDisplayTitle } from "../lib/utils";
  import Icon from "./Icon.svelte";

  let { onclose }: { onclose: () => void } = $props();

  let totalCards = $derived(Object.values($boardData).reduce((sum, cards) => sum + cards.length, 0));
  let listCount = $derived($listOrder.length);

  let listStats = $derived.by(() => {
    const stats: { name: string; count: number; pct: number; tipPct: string }[] = [];
    let maxCount = 0;

    for (const key of $listOrder) {
      const count = ($boardData[key] || []).length;
      if (count > maxCount) {
        maxCount = count;
      }
      const title = getDisplayTitle(key, $boardConfig);
      stats.push({ name: title, count, pct: 0, tipPct: "" });
    }

    for (const s of stats) {
      s.pct = maxCount > 0 ? (s.count / maxCount) * 100 : 0;
      s.tipPct = totalCards > 0 ? ((s.count / totalCards) * 100).toFixed(2) : "0.00";
    }

    return stats;
  });

  let labelStats = $derived.by(() => {
    const counts: Record<string, number> = {};
    let unlabeled = 0;

    for (const cards of Object.values($boardData)) {
      for (const card of cards) {

        const labels = card.metadata?.labels || [];
        if (labels.length === 0) {
          unlabeled++;
        } else {

          for (const label of labels) {
            counts[label] = (counts[label] || 0) + 1;
          }
        }
      }
    }
    const entries = Object.entries(counts).sort((a, b) => b[1] - a[1]);

    if (unlabeled > 0) {
      entries.push(["(no label)", unlabeled]);
    }

    const maxCount = entries.length > 0 ? Math.max(...entries.map(([, c]) => c)) : 0;
    const totalAssigned = entries.reduce((sum, [, c]) => sum + c, 0);

    return entries.map(([name, count]) => ({
      name,
      count,
      pct: maxCount > 0 ? (count / maxCount) * 100 : 0,
      tipPct: totalAssigned > 0 ? ((count / totalAssigned) * 100).toFixed(2) : "0.00",
      color: name === "(no label)" ? "var(--color-text-muted)" : ($labelColors[name] || "var(--color-accent)"),
    }));
  });

  let domainStats = $derived.by(() => {
    const counts: Record<string, number> = {};

    for (const cards of Object.values($boardData)) {
      for (const card of cards) {

        const url = card.metadata?.url;
        if (!url || (!url.startsWith("http://") && !url.startsWith("https://"))) {
          continue;
        }
        try {
          const domain = new URL(url).hostname.replace(/^www\./, "");
          counts[domain] = (counts[domain] || 0) + 1;
        } catch {
          // skip malformed URLs
        }
      }
    }
    const entries = Object.entries(counts).sort((a, b) => b[1] - a[1]);
    const maxCount = entries.length > 0 ? entries[0][1] : 0;
    const totalDomains = entries.reduce((sum, [, c]) => sum + c, 0);

    return entries.map(([name, count]) => ({
      name,
      count,
      pct: maxCount > 0 ? (count / maxCount) * 100 : 0,
      tipPct: totalDomains > 0 ? ((count / totalDomains) * 100).toFixed(2) : "0.00",
    }));
  });
</script>

<div class="modal-backdrop centered" role="presentation" use:backdropClose={onclose}>
  <div class="modal-dialog size-lg" role="dialog">
    <div class="modal-header">
      <div class="modal-title-group">
        <h2 class="modal-title">Board Statistics</h2>
        <span class="stats-total">{totalCards} cards across {listCount} lists</span>
      </div>
      <button class="modal-close" onclick={onclose} title="Close">
        <Icon name="close" size={16} />
      </button>
    </div>
    <div class="stats-body">

      <h3 class="stats-section-title" title="Number of cards in each list">Cards per List</h3>
      <div class="stats-bars">
        {#each listStats as stat}
          <div class="bar-row" title="{stat.name}: {stat.count} cards ({stat.tipPct}%)">
            <span class="bar-label">{stat.name}</span>
            <div class="bar-track">
              <div class="bar-fill" style="width: {stat.pct}%"></div>
            </div>
            <span class="bar-value">{stat.count} ({stat.tipPct}%)</span>
          </div>
        {/each}
      </div>

      {#if labelStats.length > 0}
        <h3 class="stats-section-title" title="Number of cards tagged with each label">Cards per Label</h3>
        <div class="stats-bars">
          {#each labelStats as stat}
            <div class="bar-row" title="{stat.name}: {stat.count} cards ({stat.tipPct}%)">
              <span class="bar-label">{stat.name}</span>
              <div class="bar-track">
                <div class="bar-fill" style="width: {stat.pct}%; background: {stat.color}"></div>
              </div>
              <span class="bar-value">{stat.count} ({stat.tipPct}%)</span>
            </div>
          {/each}
        </div>
      {/if}

      {#if domainStats.length > 0}
        <h3 class="stats-section-title" title="Number of cards with an HTTP/HTTPS URL, grouped by domain">Cards per Domain</h3>
        <div class="stats-bars">
          {#each domainStats as stat}
            <div class="bar-row" title="{stat.name}: {stat.count} cards ({stat.tipPct}%)">
              <span class="bar-label">{stat.name}</span>
              <div class="bar-track">
                <div class="bar-fill" style="width: {stat.pct}%"></div>
              </div>
              <span class="bar-value">{stat.count} ({stat.tipPct}%)</span>
            </div>
          {/each}
        </div>
      {/if}

    </div>
  </div>
</div>

<style lang="scss">
  .stats-body {
    padding: 0 20px 32px;
    overflow-y: auto;
    max-height: calc(80vh - 60px);
  }

  .modal-title-group {
    display: flex;
    align-items: baseline;
    gap: 10px;
  }

  .stats-total {
    color: var(--color-text-muted);
    font-size: 0.78rem;
  }

  .stats-section-title {
    font-size: 0.85rem;
    font-weight: 600;
    margin: 28px 0 8px;
    color: var(--color-text-secondary);
  }

  .stats-bars {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .bar-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .bar-label {
    width: 120px;
    font-size: 0.78rem;
    color: var(--color-text-secondary);
    text-align: right;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .bar-track {
    flex: 1;
    max-width: 420px;
    height: 18px;
    background: var(--overlay-hover-medium);
    border-radius: 4px;
    overflow: hidden;
  }

  .bar-fill {
    height: 100%;
    background: var(--color-accent);
    border-radius: 4px;
    transition: width 0.2s ease;
    min-width: 2px;
  }

  .bar-value {
    min-width: 80px;
    font-size: 0.78rem;
    color: var(--color-text-muted);
    text-align: right;
    white-space: nowrap;
  }
</style>
