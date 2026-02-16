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
    <div class="section-header">
      <h4 class="sidebar-title">Date Range</h4>
      <div class="section-header-actions">
        <button class="counter-header-btn" title="Remove end date" onclick={removeEndDate}>
          <Icon name="close" size={12} />
        </button>
        <button class="counter-header-btn remove" title="Remove dates" onclick={removeAllDates}>
          <Icon name="trash" size={12} />
        </button>
      </div>
    </div>
    <DatePicker value={range.start} onselect={d => onRangeDateSelect('start', d)} />
    <div class="range-end-row">
      <span class="range-text">to</span>
      <DatePicker value={range.end} onselect={d => onRangeDateSelect('end', d)} />
    </div>
  </div>
{:else if due}
  <div class="sidebar-section">
    <div class="section-header">
      <h4 class="sidebar-title">Date</h4>
      <div class="section-header-actions">
        <button class="counter-header-btn" title="Add end date" onclick={addEndDate}>
          <Icon name="plus" size={12} />
        </button>
        <button class="counter-header-btn remove" title="Remove date" onclick={removeAllDates}>
          <Icon name="trash" size={12} />
        </button>
      </div>
    </div>
    <DatePicker value={due} onselect={onDueDateSelect} />
  </div>
{:else}
  <div class="sidebar-section">
    <button class="add-counter-btn" title="Add a due date" onclick={addDate}>+ Add date</button>
  </div>
{/if}

<style lang="scss">
  .range-end-row {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-top: 4px;
  }
</style>
