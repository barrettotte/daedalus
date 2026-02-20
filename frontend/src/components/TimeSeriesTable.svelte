<script lang="ts">
  // Table view for time series data. Shows entries in chronological order
  // with deltas, inline editing, and row-level delete.

  import Icon from "./Icon.svelte";
  import DatePicker from "./DatePicker.svelte";
  import { autoFocus, blurOnEnter, formatDateTime } from "../lib/utils";
  import { computeDeltas } from "../lib/timeseries";
  import type { daedalus } from "../../wailsjs/go/models";

  let {
    entries,
    onedit,
    oneditdate,
    onremove,
  }: {
    entries: daedalus.TimeSeriesEntry[];
    onedit?: (idx: number, value: number) => void;
    oneditdate?: (idx: number, date: string) => void;
    onremove?: (idx: number) => void;
  } = $props();

  let deltas = $derived(computeDeltas(entries));

  // Inline value editing state (-1 means not editing).
  let editingIdx = $state(-1);
  let editValue = $state("");

  // Inline date editing state (-1 means not editing).
  let editingDateIdx = $state(-1);

  // Closes the date editor when clicking outside any datepicker element.
  function handleWindowClick(e: MouseEvent): void {
    if (editingDateIdx < 0) {
      return;
    }
    const target = e.target as HTMLElement;
    if (target.closest('.datepicker') || target.closest('.date-text')) {
      return;
    }
    editingDateIdx = -1;
  }

  // Format a delta value with sign prefix rounded to 2 decimal places, or "--" for null (first entry).
  function formatDelta(d: number | null): string {
    if (d === null) {
      return "--";
    }
    const rounded = Math.round(d * 100) / 100;
    return rounded >= 0 ? `+${rounded}` : `${rounded}`;
  }

  // Opens a value cell for inline editing.
  function startEdit(idx: number, currentValue: number): void {
    editingIdx = idx;
    editValue = String(currentValue);
  }

  // Saves the edited value on blur.
  function blurEdit(): void {
    const idx = editingIdx;
    editingIdx = -1;
    const parsed = Number(editValue);
    if (!Number.isFinite(parsed)) {
      return;
    }
    if (parsed === entries[idx]?.v) {
      return;
    }
    onedit?.(idx, parsed);
  }
</script>

<svelte:window onclick={handleWindowClick} />

{#if entries.length > 0}
  <div class="ts-table-scroll">
    <table class="ts-table">
      <thead>
        <tr>
          <th>When</th>
          <th>Value</th>
          <th>Delta</th>
          <th class="th-actions"></th>
        </tr>
      </thead>
      <tbody>
        {#each entries as entry, i}
          <tr>
            <td class="td-date">
              {#if editingDateIdx === i}
                <DatePicker value={entry.t} onselect={(iso) => { oneditdate?.(i, iso); }} inline/>
              {:else}
                <span class="date-text" role="textbox" tabindex="0"
                  onclick={() => { editingDateIdx = i; }}
                  onkeydown={(e) => e.key === 'Enter' && (editingDateIdx = i)}
                >{formatDateTime(entry.t)}</span>
              {/if}
            </td>
            <td class="td-value">
              {#if editingIdx === i}
                <input class="edit-value-input" type="text" inputmode="numeric" bind:value={editValue} onblur={blurEdit} use:blurOnEnter use:autoFocus/>
              {:else}
                <span class="value-text" role="textbox" tabindex="0"
                  onclick={() => startEdit(i, entry.v)}
                  onkeydown={(e) => e.key === 'Enter' && startEdit(i, entry.v)}
                >{entry.v}</span>
              {/if}
            </td>
            <td class="td-delta">
              {formatDelta(deltas[i])}
            </td>
            <td class="td-actions">
              <button class="cl-action remove" title="Remove entry" onclick={() => onremove?.(i)}>
                <Icon name="trash" size={10} />
              </button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}

<style lang="scss">
  .ts-table-scroll {
    max-height: 300px;
    overflow-y: auto;
  }

  .ts-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.8rem;

    th, td {
      padding: 4px 8px;
      text-align: left;
      white-space: nowrap;
    }

    th {
      color: var(--color-text-muted);
      font-weight: 600;
      font-size: 0.7rem;
      text-transform: uppercase;
      letter-spacing: 0.03em;
      border-bottom: 1px solid var(--color-border);
      position: sticky;
      top: 0;
      background: var(--color-bg-base);
      z-index: 1;
    }

    td {
      color: var(--color-text-primary);
      border-bottom: 1px solid var(--overlay-hover-faint);
    }
  }

  .th-actions {
    width: 24px;
  }

  .td-date {
    font-family: var(--font-mono);
    color: var(--color-text-secondary);
  }

  .td-value {
    font-family: var(--font-mono);
  }

  .td-delta {
    font-family: var(--font-mono);
    color: var(--color-text-primary);
  }

  .td-actions {
    width: 24px;
    text-align: center;
  }

  .value-text, .date-text {
    cursor: pointer;
    padding: 1px 4px;
    border-radius: 3px;

    &:hover {
      background: var(--overlay-hover-faint);
    }
  }

  .edit-value-input {
    width: 60px;
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

  .cl-action {
    all: unset;
    display: flex;
    align-items: center;
    flex-shrink: 0;
    color: var(--color-text-muted);
    cursor: pointer;
    padding: 2px;
    border-radius: 3px;
    opacity: 0;

    &.remove:hover {
      color: var(--color-error);
    }
  }

  tr:hover .cl-action {
    opacity: 1;
  }
</style>
