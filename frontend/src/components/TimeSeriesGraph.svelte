<script lang="ts">
  // Simple SVG line graph for time series entries. Plots values over time
  // with a polyline, dot markers, and light axis labels.

  import type { daedalus } from "../../wailsjs/go/models";

  let { entries }: { entries: daedalus.TimeSeriesEntry[] } = $props();

  // Layout constants (SVG coordinate space).
  const W = 400;
  const H = 160;
  const PAD = { top: 12, right: 12, bottom: 28, left: 44 };
  const plotW = W - PAD.left - PAD.right;
  const plotH = H - PAD.top - PAD.bottom;

  // Compute plot points from entries.
  let points = $derived.by(() => {
    if (entries.length === 0) {
      return [];
    }

    const vals = entries.map(e => e.v);
    const minV = Math.min(...vals);
    const maxV = Math.max(...vals);
    const rangeV = maxV - minV || 1;

    return entries.map((e, i) => {
      const x = PAD.left + (entries.length === 1 ? plotW / 2 : (i / (entries.length - 1)) * plotW);
      const y = PAD.top + plotH - ((e.v - minV) / rangeV) * plotH;
      return { x, y, v: e.v, t: e.t };
    });
  });

  // Y-axis tick values (min, mid, max).
  let yTicks = $derived.by(() => {
    if (entries.length === 0) {
      return [];
    }
    const vals = entries.map(e => e.v);
    const minV = Math.min(...vals);
    const maxV = Math.max(...vals);
  
    if (minV === maxV) {
      return [{ label: String(minV), y: PAD.top + plotH / 2 }];
    }
    const mid = (minV + maxV) / 2;

    return [
      { label: formatNum(maxV), y: PAD.top },
      { label: formatNum(mid), y: PAD.top + plotH / 2 },
      { label: formatNum(minV), y: PAD.top + plotH },
    ];
  });

  // X-axis labels - show first, middle, and last dates.
  let xLabels = $derived.by(() => {
    if (entries.length === 0) {
      return [];
    }
  
    const fmt = (t: string) => {
      const d = new Date(t);
      return `${d.getMonth() + 1}/${d.getDate()}`;
    };
  
    const labels: { label: string; x: number }[] = [];
    labels.push({ label: fmt(entries[0].t), x: points[0]?.x ?? PAD.left });
    if (entries.length > 2) {
      const midIdx = Math.floor(entries.length / 2);
      labels.push({ label: fmt(entries[midIdx].t), x: points[midIdx]?.x ?? PAD.left + plotW / 2 });
    }
  
    if (entries.length > 1) {
      const last = entries.length - 1;
      labels.push({ label: fmt(entries[last].t), x: points[last]?.x ?? PAD.left + plotW });
    }
    return labels;
  });

  // Polyline points attribute string.
  let polylinePoints = $derived(points.map(p => `${p.x},${p.y}`).join(" "));

  function formatNum(n: number): string {
    if (Number.isInteger(n)) {
      return String(n);
    }
    return n.toFixed(1);
  }
</script>

{#if entries.length >= 2}
  <svg class="ts-graph" viewBox="0 0 {W} {H}" preserveAspectRatio="xMidYMid meet">
    <!-- Axis border lines -->
    <line x1={PAD.left} y1={PAD.top} x2={PAD.left} y2={PAD.top + plotH} stroke="var(--color-border)" stroke-width="1"/>
    <line x1={PAD.left} y1={PAD.top + plotH} x2={PAD.left + plotW} y2={PAD.top + plotH} stroke="var(--color-border)" stroke-width="1"/>

    <!-- Grid lines -->
    {#each yTicks as tick}
      <line x1={PAD.left} y1={tick.y} x2={PAD.left + plotW} y2={tick.y} stroke="var(--color-border)" stroke-width="0.5" stroke-dasharray="3,3"/>
      <text x={PAD.left - 6} y={tick.y + 3} class="axis-label" text-anchor="end">{tick.label}</text>
    {/each}

    <!-- Line -->
    <polyline points={polylinePoints} fill="none" stroke="var(--color-accent)" stroke-width="1.5" stroke-linejoin="round" stroke-linecap="round" />

    <!-- Dots -->
    {#each points as p}
      <circle cx={p.x} cy={p.y} r="2.5" fill="var(--color-accent)" />
    {/each}

    <!-- X-axis labels -->
    {#each xLabels as label}
      <text x={label.x} y={PAD.top + plotH + 16} class="axis-label" text-anchor="middle">{label.label}</text>
    {/each}
  </svg>
{:else}
  <p class="ts-graph-empty">Need at least 2 entries to show a graph.</p>
{/if}

<style lang="scss">
  .ts-graph {
    width: 100%;
    height: auto;
    max-height: 180px;
  }

  .ts-graph-empty {
    color: var(--color-text-muted);
    font-size: 0.75rem;
    text-align: center;
    padding: 12px 0;
  }

  .axis-label {
    fill: var(--color-text-muted);
    font-size: 9px;
    font-family: var(--font-mono);
  }
</style>
