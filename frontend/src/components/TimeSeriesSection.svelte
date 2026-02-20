<script lang="ts">
  // Container for time series data with table, inline add row, and edit/remove handlers.

  import type { daedalus } from "../../wailsjs/go/models";
  import { nowString, addEntry, editEntry, editDate, removeEntry } from "../lib/timeseries";
  import { autoFocus, blurOnEnter } from "../lib/utils";
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

  // Whether the time series body is visible or collapsed.
  let expanded = $state(true);

  // View toggle: table (default) or graph.
  let showGraph = $state(false);

  // Add-row input state.
  let newDate = $state(nowString());
  let newValue = $state("");

  // Label editing state.
  let editingLabel = $state(false);
  let editLabelValue = $state("");

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

  // Opens the label for inline editing.
  function startEditLabel(): void {
    editLabelValue = timeseries.label || "Time Series";
    editingLabel = true;
  }

  // Saves the label on blur.
  function blurLabel(): void {
    editingLabel = false;
    const val = editLabelValue.trim();
    if (val && val !== (timeseries.label || "Time Series")) {
      onsave({ ...timeseries, label: val } as daedalus.TimeSeries);
    }
  }
</script>

<div class="ts-wrapper">
  <div class="ts-header">
    <button class="ts-toggle" title="Toggle time series" onclick={() => expanded = !expanded}>
      <span class="section-chevron" class:collapsed={!expanded}>
        <Icon name="chevron-down" size={12} />
      </span>
      {#if editingLabel}
        <input class="ts-label-input" type="text" bind:value={editLabelValue}
          onclick={(e) => e.stopPropagation()} onblur={blurLabel} use:blurOnEnter use:autoFocus
        />
      {:else}
        <span class="ts-label" role="textbox" tabindex="0" title="Click to rename"
          onclick={(e) => { e.stopPropagation(); startEditLabel(); }}
          onkeydown={(e) => e.key === 'Enter' && startEditLabel()}
        >{timeseries.label || "Time Series"}</span>
      {/if}
    </button>
    <button class="row-action ts-graph-btn" class:active={showGraph} title={showGraph ? "Show table" : "Show graph"}
      onclick={() => { showGraph = !showGraph; }}
    >
      <Icon name="trending-up" size={14} />
    </button>
    {#if !editingLabel}
      <button class="row-action" title="Rename" onclick={startEditLabel}>
        <Icon name="pencil" size={12} />
      </button>
    {/if}
    <button class="row-action remove" title="Delete time series" onclick={() => onsave(null)}>
      <Icon name="trash" size={12} />
    </button>
  </div>

  {#if expanded}
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

    {#if !showGraph}
      <div class="ts-add-row">
        <DatePicker value={newDate} onselect={(iso) => { newDate = iso; }} inline />
        <input type="text" inputmode="numeric" class="form-input ts-value-input" placeholder="Value"
          bind:value={newValue} onkeydown={e => e.key === 'Enter' && submitEntry()}
        />
        <button class="add-item-btn" title="Add entry" onclick={submitEntry}>
          <Icon name="plus" size={12} />
        </button>
      </div>
    {/if}
  {/if}
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
    gap: 4px;
  }

  .ts-toggle {
    all: unset;
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
    min-width: 0;
    cursor: pointer;
    border-radius: 4px;
    box-sizing: border-box;
  }

  .ts-label {
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--color-text-primary);
    margin: 0;
    cursor: text;

    &:hover {
      color: var(--color-accent);
    }
  }

  .ts-label-input {
    flex: 1;
    min-width: 0;
    background: var(--color-bg-base);
    border: 1px solid var(--color-accent);
    color: var(--color-text-primary);
    font-size: 0.9rem;
    font-weight: 600;
    padding: 0 6px;
    border-radius: 4px;
    outline: none;
    box-sizing: border-box;
  }

  .ts-header:hover :global(.row-action) {
    opacity: 1;
  }

  .ts-graph-btn {
    margin-left: auto;

    &.active {
      color: var(--color-accent);
    }
  }

  .ts-body {
    margin-top: 8px;
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

</style>
