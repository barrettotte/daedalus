import type { ActionReturn } from "svelte/action";

// Hashes a label string into a deterministic HSL color for consistent badge coloring.
export function labelColor(label: string): string {
  let hash = 0;
  for (let i = 0; i < label.length; i++) {
    hash = label.charCodeAt(i) + ((hash << 5) - hash);
  }
  const hue = ((hash % 360) + 360) % 360;
  return `hsl(${hue}, 55%, 45%)`;
}

// Strips the numeric prefix and underscores from directory names into display titles.
export function formatListName(rawName: string): string {
  const parts = rawName.split('___');
  const name = parts.length > 1 && parts[1] ? parts[1] : rawName;
  return name.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
}

// Formats a date value as "YYYY-MM-DD".
export function formatDate(d: string | null | undefined): string {
  if (!d) {
    return "";
  }
  const dt = new Date(d);
  const y = dt.getFullYear();
  const m = String(dt.getMonth() + 1).padStart(2, "0");
  const day = String(dt.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

// Formats a date value as "YYYY-MM-DD, h:mm AM/PM".
export function formatDateTime(d: string | null | undefined): string {
  if (!d) {
    return "";
  }
  const dt = new Date(d);
  const date = formatDate(d);

  let h = dt.getHours();
  const min = String(dt.getMinutes()).padStart(2, "0");
  const ampm = h >= 12 ? "PM" : "AM";
  h = h % 12 || 12;

  return `${date} ${h}:${min} ${ampm}`;
}

// Svelte action that focuses and selects the content of an input on mount.
export function autoFocus(node: HTMLInputElement | HTMLTextAreaElement): ActionReturn {
  node.focus();
  if (node.select) {
    node.select();
  }
  return {};
}
