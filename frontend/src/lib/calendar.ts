// Calendar grid, ISO parsing, and timezone utilities for DatePicker.

// Parses an ISO datetime string into its components without timezone conversion.
// Uses regex extraction instead of new Date() to avoid browser TZ reinterpretation.
export function parseISO(iso: string): {
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

  return { year: parseInt(m[1]), month: parseInt(m[2]) - 1, day: parseInt(m[3]), hour, minute, tzOffset };
}

// Returns the number of days in a given month/year.
export function daysInMonth(year: number, month: number): number {
  return new Date(year, month + 1, 0).getDate();
}

// Returns the day-of-week (0=Sun) for the 1st of a given month/year.
export function firstDayOfMonth(year: number, month: number): number {
  return new Date(year, month, 1).getDay();
}

// Builds a 6x7 grid of day numbers, with nulls for empty leading/trailing cells.
export function buildCalendarGrid(year: number, month: number): (number | null)[] {
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
export function dayToString(year: number, month: number, day: number): string {
  const m = String(month + 1).padStart(2, "0");
  const d = String(day).padStart(2, "0");
  return `${year}-${m}-${d}`;
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
export function formatTzLabel(offset: number): string {
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

// All timezone offsets available in the dropdown, sorted ascending.
export const tzOffsets: number[] = Object.keys(tzNames).map(Number).sort((a, b) => a - b);
