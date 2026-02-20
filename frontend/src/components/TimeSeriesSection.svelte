<script lang="ts">
  // Container for time series data with table, inline add row, and edit/remove handlers.

  import type { daedalus } from "../../wailsjs/go/models";
  import { nowString, addEntry, editEntry, editDate, removeEntry } from "../lib/timeseries";
  import TimeSeriesTable from "./TimeSeriesTable.svelte";
  import TimeSeriesGraph from "./TimeSeriesGraph.svelte";
  import DatePicker from "./DatePicker.svelte";
  import Icon from "./Icon.svelte";

  let {
    timeseries,
    onsave,
  }: {
    timeseries: daedalus.TimeSeries;
    onsave: (ts: daedalus.TimeSeries | null) => void;
  } = $props();

  // View toggle: table (default) or graph.
  let showGraph = $state(false);

  // Add-row input state.
  let newDate = $state(nowString());
  let newValue = $state("");

  // Validates input and adds a new entry (or updates an existing date).
  function submitEntry(): void {
    const val = parseFloat(newValue);
    if (!newDate || !Number.isFinite(val)) {
      return;
    }
    const updated = addEntry(timeseries.entries || [], newDate, val);
    onsave({ ...timeseries, entries: updated } as daedalus.TimeSeries);
    newValue = "";
  }

  // Updates a single entry's date by chronological index.
  function handleEditDate(idx: number, date: string): void {
    const updated = editDate(timeseries.entries || [], idx, date);
    onsave({ ...timeseries, entries: updated } as daedalus.TimeSeries);
  }

  // Updates a single entry's value by chronological index.
  function handleEdit(idx: number, value: number): void {
    const updated = editEntry(timeseries.entries || [], idx, value);
    onsave({ ...timeseries, entries: updated } as daedalus.TimeSeries);
  }

  // Removes an entry by chronological index.
  function handleRemove(idx: number): void {
    const updated = removeEntry(timeseries.entries || [], idx);
    onsave({ ...timeseries, entries: updated } as daedalus.TimeSeries);
  }
</script>

<div class="ts-wrapper">
  <div class="ts-header">
    {#if timeseries.label}
      <span class="ts-label">{timeseries.label}</span>
    {/if}
    <button class="ts-toggle-btn" class:active={showGraph} title={showGraph ? "Show table" : "Show graph"}
      onclick={() => { showGraph = !showGraph; }}
    >
      <Icon name="trending-up" size={14} />
    </button>
  </div>

  <div class="ts-body">
    {#if showGraph}
      <TimeSeriesGraph entries={timeseries.entries || []} />
    {:else}
      <TimeSeriesTable
        entries={timeseries.entries || []}
        onedit={handleEdit}
        oneditdate={handleEditDate}
        onremove={handleRemove}
      />
    {/if}
  </div>

  <div class="ts-add-row">
    <DatePicker value={newDate} onselect={(iso) => { newDate = iso; }} inline />
    <input type="text" inputmode="numeric" class="form-input ts-value-input" placeholder="Value"
      bind:value={newValue} onkeydown={e => e.key === 'Enter' && submitEntry()}
    />
    <button class="add-item-btn" title="Add entry" onclick={submitEntry}>
      <Icon name="plus" size={12} />
    </button>
  </div>
</div>

<style lang="scss">
  .ts-wrapper {
    background: var(--overlay-subtle);
    border-radius: 6px;
    padding: 10px 12px;
  }

  .ts-header {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 8px;
  }

  .ts-label {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--color-text-secondary);
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .ts-toggle-btn {
    all: unset;
    display: flex;
    align-items: center;
    margin-left: auto;
    color: var(--color-text-muted);
    cursor: pointer;
    padding: 2px 4px;
    border-radius: 3px;

    &:hover {
      color: var(--color-text-primary);
    }

    &.active {
      color: var(--color-accent);
    }
  }

  .ts-body {
    margin-bottom: 8px;
  }

  .ts-add-row {
    display: flex;
    align-items: center;
    gap: 4px;
    padding-top: 6px;
    border-top: 1px solid var(--overlay-hover-light);
  }

  .ts-value-input {
    flex: 1;
    min-width: 0;
    font-size: 0.8rem;
  }

  .add-item-btn {
    all: unset;
    display: flex;
    align-items: center;
    flex-shrink: 0;
    color: var(--color-text-muted);
    cursor: pointer;
    padding: 2px;
    border-radius: 3px;

    &:hover {
      color: var(--color-accent);
    }
  }
</style>
