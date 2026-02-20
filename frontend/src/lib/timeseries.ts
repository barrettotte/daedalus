// Pure time-series mutation functions. Each takes entries and returns a new
// array -- callers handle persistence (backend save or local assignment).

import type { daedalus } from "../../wailsjs/go/models";
import { toLocalISO } from "./utils";

// Returns the current date and time as a local ISO string.
export function nowString(): string {
  return toLocalISO(new Date());
}

// Sorts entries by timestamp ascending (chronological). Parses ISO strings
// via Date so entries with different timezone offsets sort correctly.
export function sortEntries(entries: daedalus.TimeSeriesEntry[]): daedalus.TimeSeriesEntry[] {
  return [...entries].sort((a, b) => new Date(a.t).getTime() - new Date(b.t).getTime());
}

// Adds or updates an entry. If the date already exists, updates the value.
// Returns a new sorted array.
export function addEntry(
  entries: daedalus.TimeSeriesEntry[],
  date: string,
  value: number,
): daedalus.TimeSeriesEntry[] {
  const trimmed = date.trim();
  if (!trimmed || !Number.isFinite(value)) {
    return entries;
  }

  const existing = entries.findIndex(e => e.t === trimmed);
  let result: daedalus.TimeSeriesEntry[];

  if (existing >= 0) {
    result = entries.map((e, i) => (i === existing ? { ...e, v: value } : { ...e }));
  } else {
    result = [...entries, { t: trimmed, v: value } as daedalus.TimeSeriesEntry];
  }
  return sortEntries(result);
}

// Updates the timestamp of an entry at the given index, then re-sorts.
export function editDate(
  entries: daedalus.TimeSeriesEntry[],
  idx: number,
  newDate: string,
): daedalus.TimeSeriesEntry[] {
  const trimmed = newDate.trim();
  if (!trimmed || entries[idx]?.t === trimmed) {
    return entries;
  }
  const updated = entries.map((e, i) => (i === idx ? { ...e, t: trimmed } : { ...e }));
  return sortEntries(updated);
}

// Updates the value of an entry at the given index.
export function editEntry(
  entries: daedalus.TimeSeriesEntry[],
  idx: number,
  value: number,
): daedalus.TimeSeriesEntry[] {
  if (!Number.isFinite(value)) {
    return entries;
  }
  return entries.map((e, i) => (i === idx ? { ...e, v: value } : { ...e }));
}

// Removes an entry by index.
export function removeEntry(
  entries: daedalus.TimeSeriesEntry[],
  idx: number,
): daedalus.TimeSeriesEntry[] {
  return entries.filter((_, i) => i !== idx);
}

// Computes deltas between consecutive entries (chronological order).
// Returns an array of (number | null) where null means "first entry, no delta".
export function computeDeltas(entries: daedalus.TimeSeriesEntry[]): (number | null)[] {
  return entries.map((e, i) => (i === 0 ? null : e.v - entries[i - 1].v));
}
