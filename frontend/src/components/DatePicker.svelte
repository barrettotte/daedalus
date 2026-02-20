<script lang="ts">
  // Calendar dropdown for selecting a date and optional time, positioned via fixed coordinates to escape overflow clipping.

  import { formatISOWithOffset } from "../lib/utils";
  import {
    parseISO, buildCalendarGrid, dayToString, formatTzLabel, tzOffsets,
  } from "../lib/calendar";

  let { value = "", onselect, inline = false }: {
    value?: string;
    onselect?: (iso: string) => void;
    inline?: boolean;
  } = $props();

  // Calendar dropdown visibility.
  let open = $state(false);

  // Trigger button ref and computed dropdown position (fixed positioning).
  let triggerEl: HTMLButtonElement | undefined = $state();
  let dropdownTop = $state(0);
  let dropdownLeft = $state(0);

  // Currently displayed year, month (0-11), hour (0-23), minute, and timezone offset.
  let viewYear = $state(new Date().getFullYear());
  let viewMonth = $state(new Date().getMonth());
  let viewHour = $state(12);
  let viewMinute = $state(0);
  let viewTzOffset = $state(-new Date().getTimezoneOffset());

  // Derived 12-hour display values from viewHour (0-23).
  let displayHour = $derived(viewHour % 12 || 12);
  let ampm = $derived(viewHour >= 12 ? "PM" : "AM");

  // Syncs calendar view to the bound value when it changes externally.
  $effect(() => {
    if (value) {
      const parsed = parseISO(value);
      if (parsed) {
        viewYear = parsed.year;
        viewMonth = parsed.month;
        viewHour = parsed.hour;
        viewMinute = parsed.minute;
        viewTzOffset = parsed.tzOffset;
      }
    }
  });

  // Formats the date portion of an ISO string for the trigger (e.g. "2026-02-15").
  // Uses parseISO instead of utils.formatDate to avoid browser timezone conversion.
  function formatTriggerDate(iso: string): string {
    if (!iso) {
      return "Select date";
    }
    const parsed = parseISO(iso);
    if (!parsed) {
      return "Select date";
    }
    const y = parsed.year;
    const mo = String(parsed.month + 1).padStart(2, "0");
    const d = String(parsed.day).padStart(2, "0");
    return `${y}-${mo}-${d}`;
  }

  // Formats the time portion of an ISO string for the trigger (e.g. "12:00 PM").
  function formatTriggerTime(iso: string): string {
    if (!iso) {
      return "";
    }
    const parsed = parseISO(iso);
    if (!parsed) {
      return "";
    }
    const h = parsed.hour % 12 || 12;
    const min = String(parsed.minute).padStart(2, "0");
    const ap = parsed.hour >= 12 ? "PM" : "AM";
    return `${h}:${min} ${ap}`;
  }

  // Extracts the YYYY-MM-DD portion from the value for grid highlighting.
  let selectedDateStr = $derived.by(() => {
    if (!value) {
      return "";
    }
    const parsed = parseISO(value);
    if (!parsed) {
      return "";
    }
    return dayToString(parsed.year, parsed.month, parsed.day);
  });

  // Today's date in the selected timezone for grid highlighting.
  let todayStr = $derived.by(() => {
    const now = new Date();
    const fakeUtc = new Date(now.getTime() + viewTzOffset * 60_000);
    return dayToString(
      fakeUtc.getUTCFullYear(), fakeUtc.getUTCMonth(), fakeUtc.getUTCDate(),
    );
  });

  // Steps the calendar view forward or backward by one month.
  function changeMonth(delta: number): void {
    viewMonth += delta;
    if (viewMonth < 0) {
      viewMonth = 11;
      viewYear--;
    } else if (viewMonth > 11) {
      viewMonth = 0;
      viewYear++;
    }
  }

  // Builds an ISO string from the current view state.
  function buildISO(year: number, month: number, day: number, hour: number, minute: number, tz: number): string {
    return formatISOWithOffset(year, month + 1, day, hour, minute, tz);
  }

  // Emits the current time for the existing date. Shared by time/tz change handlers.
  function emitTime(): void {
    if (!value || !onselect) {
      return;
    }

    const parsed = parseISO(value);
    if (!parsed) {
      return;
    }
    onselect(buildISO(
      parsed.year, parsed.month, parsed.day,
      viewHour, viewMinute, viewTzOffset,
    ));
  }

  // Handles a day click -- builds ISO with current time/tz, fires onselect, closes.
  function selectDay(day: number): void {
    if (onselect) {
      onselect(buildISO(viewYear, viewMonth, day, viewHour, viewMinute, viewTzOffset));
    }
    open = false;
  }

  // Selects current date and time in the selected timezone.
  function selectNow(): void {
    const now = new Date();
    const fakeUtc = new Date(now.getTime() + viewTzOffset * 60_000);

    viewYear = fakeUtc.getUTCFullYear();
    viewMonth = fakeUtc.getUTCMonth();
    viewHour = fakeUtc.getUTCHours();
    viewMinute = fakeUtc.getUTCMinutes();

    if (onselect) {
      onselect(buildISO(
        viewYear, viewMonth, fakeUtc.getUTCDate(),
        viewHour, viewMinute, viewTzOffset,
      ));
    }
    open = false;
  }

  // Handles 12-hour input commit -- converts to 24h and saves.
  function onHourInput(e: Event): void {
    const el = e.target as HTMLInputElement;
    const h12 = Math.max(1, Math.min(12, parseInt(el.value) || 12));

    if (ampm === "AM") {
      viewHour = h12 === 12 ? 0 : h12;
    } else {
      viewHour = h12 === 12 ? 12 : h12 + 12;
    }
    emitTime();
  }

  // Handles minute input commit -- clamps and saves.
  function onMinuteChange(): void {
    viewMinute = Math.max(0, Math.min(59, Math.floor(viewMinute) || 0));
    emitTime();
  }

  // Toggles AM/PM by flipping viewHour by 12 hours.
  function toggleAmPm(): void {
    viewHour = (viewHour + 12) % 24;
    emitTime();
  }

  // Saves updated timezone when the offset selector changes.
  function onTzChange(): void {
    emitTime();
  }

  // Toggles the calendar dropdown open/closed, computing fixed position from trigger.
  function toggleCalendar(): void {
    if (!open && triggerEl) {
      const rect = triggerEl.getBoundingClientRect();
      dropdownTop = rect.bottom + 4;
      dropdownLeft = rect.left + rect.width / 2;
    }
    open = !open;
  }

  // Closes the calendar when clicking the backdrop.
  function onBackdropClick(): void {
    open = false;
  }

  const dayHeaders = ["S", "M", "T", "W", "T", "F", "S"];

  const monthNames = [
    "Jan", "Feb", "Mar", "Apr", "May", "Jun",
    "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
  ];

  // Derived calendar grid for the current view.
  let grid = $derived(buildCalendarGrid(viewYear, viewMonth));
</script>

<div class="datepicker" class:inline>
  <button class="datepicker-trigger" class:inline bind:this={triggerEl} onclick={toggleCalendar}>
    <span class="trigger-date">{formatTriggerDate(value)}</span>
    {#if formatTriggerTime(value)}
      <span class="trigger-time">{formatTriggerTime(value)}</span>
    {/if}
  </button>

  {#if open}
    <div class="datepicker-backdrop" onclick={onBackdropClick} role="presentation"></div>
    <div class="datepicker-dropdown" style="top: {dropdownTop}px; left: {dropdownLeft}px;">
      <div class="datepicker-nav">
        <button class="btn-icon datepicker-nav-btn" title="Previous month" onclick={() => changeMonth(-1)}>&lt;</button>
        <span class="datepicker-month-label">{monthNames[viewMonth]} {viewYear}</span>
        <button class="btn-icon datepicker-nav-btn" title="Next month" onclick={() => changeMonth(1)}>&gt;</button>
      </div>
      <div class="datepicker-grid">
        {#each dayHeaders as dh}
          <span class="datepicker-day-header">{dh}</span>
        {/each}
        {#each grid as cell}
          {#if cell === null}
            <span class="datepicker-cell empty"></span>
          {:else}
            {@const ds = dayToString(viewYear, viewMonth, cell)}
            <button class="datepicker-cell day" class:selected={ds === selectedDateStr} class:today={ds === todayStr} onclick={() => selectDay(cell)}>
              {cell}
            </button>
          {/if}
        {/each}
      </div>
      <div class="datepicker-time">
        <input type="number" class="datepicker-time-input" min="1" max="12" value={displayHour} onchange={onHourInput}
          onkeydown={e => e.key === 'Enter' && onHourInput(e)}
        />
        <span class="datepicker-time-sep">:</span>
        <input type="number" class="datepicker-time-input" min="0" max="59"
          bind:value={viewMinute} onchange={onMinuteChange}
          onkeydown={e => e.key === 'Enter' && onMinuteChange()}
        />
        <button class="datepicker-ampm-btn" title="Toggle AM/PM" onclick={toggleAmPm}>{ampm}</button>
        <select class="datepicker-tz-select" bind:value={viewTzOffset} onchange={onTzChange}>
          {#each tzOffsets as tz}
            <option value={tz}>{formatTzLabel(tz)}</option>
          {/each}
        </select>
      </div>
      <button class="datepicker-now-btn" title="Set to current date and time" onclick={selectNow}>Now</button>
    </div>
  {/if}
</div>

<style lang="scss">
  .datepicker {
    position: relative;
    width: 100%;

    &.inline {
      width: auto;
    }
  }

  .datepicker-trigger {
    all: unset;
    width: 100%;
    display: flex;
    align-items: center;
    gap: 4px;
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    color: var(--color-text-primary);
    padding: 4px 8px;
    border-radius: 4px;
    cursor: pointer;
    box-sizing: border-box;

    &:hover {
      border-color: var(--color-text-tertiary);
    }

    &.inline {
      width: auto;
      background: none;
      border: none;
      padding: 0;
      color: var(--color-text-secondary);

      &:hover {
        color: var(--color-text-primary);
      }
    }
  }

  .trigger-date {
    font-family: var(--font-mono);
    font-size: 0.75rem;
    white-space: nowrap;
  }

  .trigger-time {
    font-family: var(--font-mono);
    font-size: 0.65rem;
    color: var(--color-text-muted);
    white-space: nowrap;
  }


  .datepicker-backdrop {
    position: fixed;
    inset: 0;
    z-index: var(--z-dropdown);
  }

  .datepicker-dropdown {
    position: fixed;
    transform: translateX(-50%);
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: 6px;
    padding: 8px;
    z-index: calc(var(--z-dropdown) + 1);
    width: 230px;
    box-sizing: border-box;
  }

  .datepicker-nav {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 6px;
  }

  .datepicker-nav-btn {
    width: 22px;
    height: 22px;
  }

  .datepicker-month-label {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--color-text-primary);
  }

  .datepicker-grid {
    display: grid;
    grid-template-columns: repeat(7, 1fr);
    gap: 1px;
  }

  .datepicker-day-header {
    font-size: 0.6rem;
    font-weight: 600;
    color: var(--color-text-muted);
    text-align: center;
    padding: 2px 0;
  }

  .datepicker-cell {
    all: unset;
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: var(--font-mono);
    font-size: 0.65rem;
    border-radius: 4px;
    box-sizing: border-box;
    margin: 0 auto;

    &.empty {
      cursor: default;
    }

    &.day {
      color: var(--color-text-primary);
      cursor: pointer;

      &:hover {
        background: var(--overlay-hover);
      }

      &.today {
        outline: 1px solid var(--color-text-muted);
        outline-offset: -1px;
      }

      &.selected {
        background: var(--overlay-accent);
        color: var(--color-accent);
        font-weight: 700;
        outline: none;
      }
    }
  }

  .datepicker-time {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 2px;
    margin-top: 6px;
    padding-top: 8px;
    border-top: 1px solid var(--color-border);
  }

  .datepicker-time-input {
    width: 32px;
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    color: var(--color-text-primary);
    font-family: var(--font-mono);
    font-size: 0.7rem;
    padding: 3px 2px;
    border-radius: 4px;
    text-align: center;
    outline: none;
    box-sizing: border-box;
    appearance: textfield;
    -moz-appearance: textfield;

    &::-webkit-inner-spin-button,
    &::-webkit-outer-spin-button {
      -webkit-appearance: none;
      margin: 0;
    }

    &:focus {
      border-color: var(--color-accent);
    }
  }

  .datepicker-time-sep {
    font-family: var(--font-mono);
    font-size: 0.75rem;
    color: var(--color-text-primary);
    font-weight: 600;
  }

  .datepicker-ampm-btn {
    all: unset;
    font-size: 0.7rem;
    font-weight: 700;
    color: var(--color-text-primary);
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    border-radius: 4px;
    padding: 3px 4px;
    cursor: pointer;
    box-sizing: border-box;

    &:hover {
      background: var(--overlay-hover);
      border-color: var(--color-text-tertiary);
    }
  }

  .datepicker-tz-select {
    flex: 1;
    min-width: 0;
    margin-left: 6px;
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    color: var(--color-text-primary);
    font-size: 0.7rem;
    padding: 3px 2px;
    border-radius: 4px;
    outline: none;
    box-sizing: border-box;
    cursor: pointer;
    appearance: none;
    -webkit-appearance: none;

    &:focus {
      border-color: var(--color-accent);
    }
  }

  .datepicker-now-btn {
    all: unset;
    display: block;
    width: 100%;
    text-align: center;
    font-size: 0.65rem;
    font-weight: 600;
    color: var(--color-text-primary);
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    cursor: pointer;
    margin-top: 6px;
    padding: 3px 0;
    border-radius: 4px;
    box-sizing: border-box;

    &:hover {
      background: var(--overlay-hover);
      border-color: var(--color-text-tertiary);
    }
  }
</style>
