// Shared utilities: date formatting, label colors, display helpers, and Svelte actions.

import type { ActionReturn } from "svelte/action";
import type { BoardLists, BoardConfigMap } from "../stores/board";
import { addToast } from "../stores/board";

// Max cards shown in half-collapsed lists before the "Show N more" button.
export const HALF_COLLAPSED_CARD_LIMIT = 5;

// Copies text to the clipboard and shows a success/error toast.
export async function copyToClipboard(text: string, label: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text);
    addToast(`${label} copied`, "success");
  } catch {
    addToast(`Failed to copy ${label}`);
  }
}

// Returns a color for a label - custom override if set, otherwise a deterministic HSL hash.
export function labelColor(label: string, customColors?: Record<string, string>): string {
  if (customColors && customColors[label]) {
    return customColors[label];
  }

  let hash = 0;
  for (let i = 0; i < label.length; i++) {
    hash = label.charCodeAt(i) + ((hash << 5) - hash);
  }
  const hue = ((hash % 360) + 360) % 360;
  return `hsl(${hue}, 55%, 45%)`;
}

// Converts directory slug into a display title (replaces dashes/underscores, title-cases words).
export function formatListName(rawName: string): string {
  return rawName.replace(/[-_]/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
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

// Formats a date value as "YYYY-MM-DD h:mm AM/PM".
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
  const hStr = String(h).padStart(2, "\u00A0");

  return `${date} ${hStr}:${min} ${ampm}`;
}

// Returns the config title override if set, otherwise the formatted directory name.
export function getDisplayTitle(listKey: string, config: BoardConfigMap): string {
  const cfg = config[listKey];
  if (cfg && cfg.title) {
    return cfg.title;
  }
  return formatListName(listKey);
}

// Returns "count/limit" when a limit is set, otherwise just the count.
export function getCountDisplay(listKey: string, lists: BoardLists, config: BoardConfigMap): string {
  const count = lists[listKey]?.length || 0;
  const cfg = config[listKey];

  if (cfg && cfg.limit > 0) {
    return `${count}/${cfg.limit}`;
  }
  return `${count}`;
}

// Returns true when the card count exceeds the configured limit.
export function isOverLimit(listKey: string, lists: BoardLists, config: BoardConfigMap): boolean {
  const cfg = config[listKey];
  if (!cfg || cfg.limit <= 0) {
    return false;
  }
  return (lists[listKey]?.length || 0) > cfg.limit;
}

// Formats date/time components as an ISO string with an explicit timezone offset.
// offsetMinutes is minutes east of UTC (e.g., -300 for UTC-5, +330 for UTC+5:30).
export function formatISOWithOffset(
  year: number, month: number, day: number,
  hour: number, minute: number,
  offsetMinutes: number,
): string {
  const sign = offsetMinutes >= 0 ? "+" : "-";
  const absOff = Math.abs(offsetMinutes);
  const offH = String(Math.floor(absOff / 60)).padStart(2, "0");
  const offM = String(absOff % 60).padStart(2, "0");

  const mo = String(month).padStart(2, "0");
  const d = String(day).padStart(2, "0");
  const h = String(hour).padStart(2, "0");
  const min = String(minute).padStart(2, "0");

  return `${year}-${mo}-${d}T${h}:${min}:00${sign}${offH}:${offM}`;
}

// Formats a Date as an ISO string with local timezone offset (e.g., 2026-02-13T17:00:00-05:00).
export function toLocalISO(dt: Date): string {
  return formatISOWithOffset(
    dt.getFullYear(), dt.getMonth() + 1, dt.getDate(),
    dt.getHours(), dt.getMinutes(),
    -dt.getTimezoneOffset(),
  );
}

// Svelte action that closes a modal when clicking the backdrop. Uses mousedown/mouseup
// to prevent accidental closes when a click starts inside the modal and drags to the backdrop.
export function backdropClose(node: HTMLElement, onclose: () => void): ActionReturn<() => void> {
  let mouseDownOnBackdrop = false;

  function handleMousedown(e: MouseEvent): void {
    if (e.button !== 0) {
      return;
    }
    mouseDownOnBackdrop = e.target === e.currentTarget;
  }

  function handleMouseup(e: MouseEvent): void {
    if (e.button !== 0) {
      return;
    }
    if (mouseDownOnBackdrop && e.target === e.currentTarget) {
      onclose();
    }
    mouseDownOnBackdrop = false;
  }

  node.addEventListener("mousedown", handleMousedown);
  node.addEventListener("mouseup", handleMouseup);

  return {
    update(newOnclose: () => void) {
      onclose = newOnclose;
    },
    destroy() {
      node.removeEventListener("mousedown", handleMousedown);
      node.removeEventListener("mouseup", handleMouseup);
    },
  };
}

// Svelte action that calls a callback when a click lands outside the attached element.
export function clickOutside(node: HTMLElement, callback: () => void): ActionReturn<() => void> {
  function handleClick(e: MouseEvent): void {
    if (e.button !== 0) {
      return;
    }
    const target = e.target as Node;
    if (!target.isConnected) {
      return;
    }
    if (!node.contains(target)) {
      callback();
    }
  }

  window.addEventListener("click", handleClick);

  return {
    update(newCallback: () => void) {
      callback = newCallback;
    },
    destroy() {
      window.removeEventListener("click", handleClick);
    },
  };
}

// Svelte action that focuses and selects the content of an input on mount.
export function autoFocus(node: HTMLInputElement | HTMLTextAreaElement): ActionReturn {
  node.focus();
  if (node.select) {
    node.select();
  }
  return {};
}

// Svelte action that blurs (unfocuses) an input when the Enter key is pressed.
export function blurOnEnter(node: HTMLInputElement | HTMLTextAreaElement): ActionReturn {

  function handleKeydown(e: Event): void {
    if ((e as KeyboardEvent).key === "Enter") {
      node.blur();
    }
  }

  node.addEventListener("keydown", handleKeydown);
  return {
    destroy() {
      node.removeEventListener("keydown", handleKeydown);
    },
  };
}

// Counts words in a string by splitting on whitespace.
export function wordCount(text: string): number {
  const trimmed = text.trim();
  return trimmed ? trimmed.split(/\s+/).length : 0;
}

// Returns true if the icon string references a file-based icon (SVG or PNG).
export function isFileIcon(icon: string): boolean {
  return icon ? icon.endsWith(".svg") || icon.endsWith(".png") : false;
}
