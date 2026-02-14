<script lang="ts">
  import { formatISOWithOffset } from "../lib/utils";

  let { value = "", onselect }: {
    value?: string;
    onselect?: (iso: string) => void;
  } = $props();

  // Calendar dropdown visibility.
  let open = $state(false);

  // Currently displayed year, month (0-11), hour (0-23), minute, and timezone offset.
  let viewYear = $state(new Date().getFullYear());
  let viewMonth = $state(new Date().getMonth());
  let viewHour = $state(12);
  let viewMinute = $state(0);
  let viewTzOffset = $state(-new Date().getTimezoneOffset());

  // Derived 12-hour display values from viewHour (0-23).
  let displayHour = $derived(viewHour % 12 || 12);
  let ampm = $derived(viewHour >= 12 ? "PM" : "AM");

  // Parses an ISO datetime string into its components without timezone conversion.
  function parseISO(iso: string): {
    year: number;
    month: number;
    day: number;
    hour: number;
    minute: number;
    tzOffset: number;
  } | null {

    const m = iso.match(/^(\d{4})-(\d{2})-(\d{2})(?:T(\d{2}):(\d{2})(?::(\d{2})(?:\.\d+)?)?([+-]\d{2}:\d{2}|Z)?)?/);
    if (!m) {
      return null;
    }
    const hour = m[4] ? parseInt(m[4]) : 0;
    const minute = m[5] ? parseInt(m[5]) : 0;

    let tzOffset: number;
    const tz = m[7];
    if (!tz) {
      tzOffset = -new Date().getTimezoneOffset();
    } else if (tz === "Z") {
      tzOffset = 0;
    } else {
      const tzSign = tz[0] === "+" ? 1 : -1;
      const parts = tz.slice(1).split(":");
      tzOffset = tzSign * (parseInt(parts[0]) * 60 + parseInt(parts[1]));
    }
    return {
      year: parseInt(m[1]),
      month: parseInt(m[2]) - 1,
      day: parseInt(m[3]),
      hour,
      minute,
      tzOffset,
    };
  }

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

  // Returns the number of days in a given month/year.
  function daysInMonth(year: number, month: number): number {
    return new Date(year, month + 1, 0).getDate();
  }

  // Returns the day-of-week (0=Sun) for the 1st of a given month/year.
  function firstDayOfMonth(year: number, month: number): number {
    return new Date(year, month, 1).getDay();
  }

  // Builds a 6x7 grid of day numbers, with nulls for empty leading/trailing cells.
  function buildCalendarGrid(year: number, month: number): (number | null)[] {
    const total = daysInMonth(year, month);
    const startDay = firstDayOfMonth(year, month);
    const cells: (number | null)[] = [];

    for (let i = 0; i < startDay; i++) {
      cells.push(null);
    }
    for (let d = 1; d <= total; d++) {
      cells.push(d);
    }
    while (cells.length % 7 !== 0) {
      cells.push(null);
    }
    return cells;
  }

  // Formats a day number into YYYY-MM-DD for grid comparison.
  function dayToString(year: number, month: number, day: number): string {
    const m = String(month + 1).padStart(2, "0");
    const d = String(day).padStart(2, "0");
    return `${year}-${m}-${d}`;
  }

  // Formats an ISO string for the trigger display with 12-hour time.
  function formatTrigger(iso: string): string {
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
    const h = parsed.hour % 12 || 12;
    const min = String(parsed.minute).padStart(2, "0");
    const ap = parsed.hour >= 12 ? "PM" : "AM";
    return `${y}-${mo}-${d} ${h}:${min} ${ap}`;
  }

  // Maps UTC offsets (in minutes) to common timezone abbreviations.
  const tzNames: Record<number, string> = {
    [-720]: "AoE",
    [-660]: "SST",
    [-600]: "HST",
    [-540]: "AKST",
    [-480]: "PST",
    [-420]: "MST",
    [-360]: "CST",
    [-300]: "EST",
    [-240]: "AST",
    [-210]: "NST",
    [-180]: "BRT",
    [-120]: "GST",
    [-60]: "CVT",
    [0]: "GMT",
    [60]: "CET",
    [120]: "EET",
    [180]: "MSK",
    [210]: "IRST",
    [240]: "GST",
    [270]: "AFT",
    [300]: "PKT",
    [330]: "IST",
    [345]: "NPT",
    [360]: "BST",
    [390]: "MMT",
    [420]: "ICT",
    [480]: "CST",
    [525]: "ACWST",
    [540]: "JST",
    [570]: "ACST",
    [600]: "AEST",
    [630]: "LHST",
    [660]: "SBT",
    [720]: "NZST",
    [765]: "CHAST",
    [780]: "PHST",
    [840]: "LINT",
  };

  // Formats a timezone offset as "NAME (UTC+/-X)" or just "UTC+/-X" for unknown offsets.
  function formatTzLabel(offset: number): string {
    const sign = offset >= 0 ? "+" : "-";
    const abs = Math.abs(offset);
    const h = Math.floor(abs / 60);
    const m = abs % 60;
    const utc = m === 0 ? `UTC${sign}${h}` : `UTC${sign}${h}:${String(m).padStart(2, "0")}`;

    const name = tzNames[offset];
    if (name) {
      return `${name} (${utc})`;
    }
    return utc;
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

  // Toggles the calendar dropdown open/closed.
  function toggleCalendar(): void {
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

  // Common UTC offsets in minutes east of UTC.
  const tzOffsets = [
    -720, -660, -600, -540, -480, -420, -360, -300, -240, -210,
    -180, -120, -60, 0, 60, 120, 180, 210, 240, 270,
    300, 330, 345, 360, 390, 420, 480, 525, 540, 570,
    600, 630, 660, 720, 765, 780, 840,
  ];

  // Derived calendar grid for the current view.
  let grid = $derived(buildCalendarGrid(viewYear, viewMonth));
</script>

<div class="datepicker">
  <button class="datepicker-trigger" onclick={toggleCalendar}>
    {formatTrigger(value)}
  </button>

  {#if open}
    <div class="datepicker-backdrop" onclick={onBackdropClick} role="presentation"></div>
    <div class="datepicker-dropdown">
      <div class="datepicker-nav">
        <button class="datepicker-nav-btn" title="Previous month" onclick={() => changeMonth(-1)}>&lt;</button>
        <span class="datepicker-month-label">{monthNames[viewMonth]} {viewYear}</span>
        <button class="datepicker-nav-btn" title="Next month" onclick={() => changeMonth(1)}>&gt;</button>
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
            <button class="datepicker-cell day" class:selected={ds === selectedDateStr}
              class:today={ds === todayStr} onclick={() => selectDay(cell)}
            >
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
  }

  .datepicker-trigger {
    all: unset;
    width: 100%;
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    color: var(--color-text-primary);
    font-size: 0.75rem;
    padding: 5px 8px;
    border-radius: 4px;
    cursor: pointer;
    box-sizing: border-box;
    text-align: center;

    &:hover {
      border-color: var(--color-text-tertiary);
    }
  }

  .datepicker-backdrop {
    position: fixed;
    inset: 0;
    z-index: 99;
  }

  .datepicker-dropdown {
    position: absolute;
    top: calc(100% + 4px);
    left: 50%;
    transform: translateX(-50%);
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: 6px;
    padding: 8px;
    z-index: 100;
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
    all: unset;
    width: 22px;
    height: 22px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 4px;
    color: var(--color-text-primary);
    font-size: 0.8rem;
    cursor: pointer;

    &:hover {
      background: var(--overlay-hover);
    }
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
