<script lang="ts">
  // Sidebar section for editing due date and optional date range using DatePicker dropdowns.

  import { toLocalISO } from "../lib/utils";
  import DatePicker from "./DatePicker.svelte";
  import Icon from "./Icon.svelte";

  let {
    due,
    range,
    onsave,
  }: {
    due?: string;
    range?: { start: string; end: string };
    onsave?: (due: string | null, range: { start: string; end: string } | null) => void;
  } = $props();

  // Adds a single date (due) set to now.
  function addDate(): void {
    if (!onsave) {
      return;
    }
    onsave(toLocalISO(new Date()), null);
  }

  // Removes all dates (due and range).
  function removeAllDates(): void {
    if (!onsave) {
      return;
    }
    onsave(null, null);
  }

  // Handles due date selection from the date picker.
  function onDueDateSelect(iso: string): void {
    if (!onsave) {
      return;
    }
    onsave(iso, null);
  }

  // Promotes a single due date to a range by adding an end date (start + 1 day).
  function addEndDate(): void {
    if (!onsave || !due) {
      return;
    }
    const start = due;
    const dt = new Date(start);
    dt.setDate(dt.getDate() + 1);

    const end = toLocalISO(dt);
    onsave(null, { start, end });
  }

  // Demotes a range back to a single due date (keeps the start date).
  function removeEndDate(): void {
    if (!onsave || !range) {
      return;
    }
    onsave(range.start, null);
  }

  // Handles range date selection from the date picker, swapping if start > end.
  function onRangeDateSelect(field: "start" | "end", iso: string): void {
    if (!onsave || !range) {
      return;
    }
    let start = field === "start" ? iso : range.start;
    let end = field === "end" ? iso : range.end;

    if (start > end) {
      [start, end] = [end, start];
    }
    onsave(null, { start, end });
  }
</script>

{#if range}
  <div class="sidebar-section">
    <div class="date-header">
      <h4 class="sidebar-title">Date Range</h4>
      <button class="counter-header-btn remove" title="Remove dates" onclick={removeAllDates}>
        <Icon name="trash" size={12} />
      </button>
    </div>
    <div class="counter-range-row">
      <DatePicker value={range.start} onselect={d => onRangeDateSelect('start', d)} />
      <span class="range-text">to</span>
      <DatePicker value={range.end} onselect={d => onRangeDateSelect('end', d)} />
    </div>
    <button class="date-expand-btn" title="Convert to single date" onclick={removeEndDate}>- Remove end date</button>
  </div>
{:else if due}
  <div class="sidebar-section">
    <div class="date-header">
      <h4 class="sidebar-title">Date</h4>
      <button class="counter-header-btn remove" title="Remove date" onclick={removeAllDates}>
        <Icon name="trash" size={12} />
      </button>
    </div>
    <DatePicker value={due} onselect={onDueDateSelect} />
    <button class="date-expand-btn" title="Add an end date to create a range" onclick={addEndDate}>+ Add end date</button>
  </div>
{:else}
  <div class="sidebar-section">
    <button class="add-counter-btn" title="Add a due date" onclick={addDate}>+ Add date</button>
  </div>
{/if}

<style lang="scss">
  .date-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 4px;

    .sidebar-title {
      margin: 0;
    }
  }

  .date-expand-btn {
    all: unset;
    display: block;
    width: 100%;
    text-align: center;
    font-size: 0.7rem;
    color: var(--color-text-muted);
    cursor: pointer;
    margin-top: 6px;

    &:hover {
      color: var(--color-text-primary);
    }
  }

</style>
