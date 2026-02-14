import { writable, derived } from 'svelte/store';
import type { Writable, Readable } from 'svelte/store';
import type { daedalus } from '../../wailsjs/go/models';

// Map of list directory names to their sorted card arrays.
export type BoardLists = Record<string, daedalus.KanbanCard[]>;

// Map of list directory names to their config (title, limit).
export type BoardConfigMap = Record<string, { title: string; limit: number }>;

// Keyboard focus position on the board (list + index within that list).
export interface FocusState {
  listKey: string;
  cardIndex: number;
}

// Active drag operation state.
export interface DragInfo {
  card: daedalus.KanbanCard;
  sourceListKey: string;
}

// Target location for an in-progress drop.
export interface DropInfo {
  listKey: string;
  cardId: number | null;
  position: "above" | "below";
}

// Toast notification entry.
export interface Toast {
  id: number;
  message: string;
}

export const boardData: Writable<BoardLists> = writable({});
export const boardConfig: Writable<BoardConfigMap> = writable({});
export const labelColors: Writable<Record<string, string>> = writable({});
export const boardPath: Writable<string> = writable("");
export const isLoaded: Writable<boolean> = writable(false);
export const selectedCard: Writable<daedalus.KanbanCard | null> = writable(null);
export const draftListKey: Writable<string | null> = writable(null);
export const draftPosition: Writable<string> = writable("top");
export const showMetrics: Writable<boolean> = writable(false);
export const labelsExpanded: Writable<boolean> = writable(true);
export const dragState: Writable<DragInfo | null> = writable(null);
export const dropTarget: Writable<DropInfo | null> = writable(null);
export const focusedCard: Writable<FocusState | null> = writable(null);
export const openInEditMode: Writable<boolean> = writable(false);
export const listOrder: Writable<string[]> = writable([]);

// Updates a single card in the boardData store by matching filePath.
export function updateCardInBoard(updatedCard: daedalus.KanbanCard): void {
  boardData.update(lists => {
    for (const listKey of Object.keys(lists)) {
      const idx = lists[listKey].findIndex(c => c.filePath === updatedCard.filePath);
      if (idx !== -1) {
        lists[listKey][idx] = updatedCard;
        break;
      }
    }
    return lists;
  });
}

// Removes a card from the boardData store by matching filePath across all lists.
export function removeCardFromBoard(filePath: string): void {
  boardData.update(lists => {
    for (const listKey of Object.keys(lists)) {
      const idx = lists[listKey].findIndex(c => c.filePath === filePath);
      if (idx !== -1) {
        lists[listKey].splice(idx, 1);
        break;
      }
    }
    return lists;
  });
}

// Adds a new card to the given list. Prepends for "top", appends for "bottom",
// or splices at the parsed index for numeric position strings.
export function addCardToBoard(listKey: string, card: daedalus.KanbanCard, position: string = "top"): void {
  boardData.update(lists => {
    if (lists[listKey]) {
      if (position === "bottom") {
        lists[listKey] = [...lists[listKey], card];
      } else {
        const idx = parseInt(position, 10);
        if (!isNaN(idx)) {
          const clamped = Math.max(0, Math.min(idx, lists[listKey].length));
          const copy = [...lists[listKey]];
          copy.splice(clamped, 0, card);
          lists[listKey] = copy;
        } else {
          lists[listKey] = [card, ...lists[listKey]];
        }
      }
    }
    return lists;
  });
}

// Moves a card from one list position to another, updating list_order in the store.
export function moveCardInBoard(filePath: string, sourceListKey: string, targetListKey: string, targetIndex: number, newListOrder: number): void {
  boardData.update(lists => {
    const srcIdx = lists[sourceListKey].findIndex(c => c.filePath === filePath);
    if (srcIdx === -1) {
      return lists;
    }

    const card = { ...lists[sourceListKey][srcIdx] } as daedalus.KanbanCard;
    card.metadata = { ...card.metadata, list_order: newListOrder } as daedalus.CardMetadata;

    // Remove from source
    lists[sourceListKey] = [...lists[sourceListKey]];
    lists[sourceListKey].splice(srcIdx, 1);

    // Insert at target
    lists[targetListKey] = [...lists[targetListKey]];
    lists[targetListKey].splice(targetIndex, 0, card);

    return lists;
  });
}

// Computes a list_order float64 for inserting a card at targetIndex in the given cards array.
export function computeListOrder(cards: daedalus.KanbanCard[], targetIndex: number): number {
  if (cards.length === 0) {
    return 0;
  }
  if (targetIndex <= 0) {
    return cards[0].metadata.list_order - 1;
  }
  if (targetIndex >= cards.length) {
    return cards[cards.length - 1].metadata.list_order + 1;
  }

  const before = cards[targetIndex - 1].metadata.list_order;
  const after = cards[targetIndex].metadata.list_order;
  return (before + after) / 2;
}

// Returns true when a list has a limit set and the card count is at or above it.
export function isAtLimit(listKey: string, lists: BoardLists, config: BoardConfigMap): boolean {
  const cfg = config[listKey];
  if (!cfg || cfg.limit <= 0) {
    return false;
  }
  return (lists[listKey]?.length || 0) >= cfg.limit;
}

// Sort lists by custom order first, then alphabetically for any remaining keys.
export const sortedListKeys = (lists: BoardLists, order: string[] = []): string[] => {
  const allKeys = new Set(Object.keys(lists));
  const result: string[] = [];

  for (const key of order) {
    if (allKeys.has(key)) {
      result.push(key);
      allKeys.delete(key);
    }
  }
  result.push(...[...allKeys].sort());
  return result;
};

export const searchQuery: Writable<string> = writable("");

// Parsed search token: plain text, #label prefix, #<digits> card ID, or @date prefix.
interface SearchToken {
  type: "text" | "label" | "date" | "id";
  value: string;
}

// Parses a query string into typed search tokens.
function parseSearchTokens(query: string): SearchToken[] {
  const tokens: SearchToken[] = [];
  const parts = query.trim().split(/\s+/);
  for (const part of parts) {
    if (!part) {
      continue;
    }
    if (part.startsWith("#")) {
      const val = part.slice(1);
      if (val && /^\d+$/.test(val)) {
        tokens.push({ type: "id", value: val });
      } else if (val) {
        tokens.push({ type: "label", value: val.toLowerCase() });
      }
    } else if (part.startsWith("@")) {
      const val = part.slice(1);
      if (val) {
        tokens.push({ type: "date", value: val });
      }
    } else {
      tokens.push({ type: "text", value: part.toLowerCase() });
    }
  }
  return tokens;
}

// Returns true when a card matches a single search token.
function cardMatchesToken(card: daedalus.KanbanCard, token: SearchToken): boolean {
  if (token.type === "text") {
    const title = (card.metadata.title || "").toLowerCase();
    const preview = (card.previewText || "").toLowerCase();
    return title.includes(token.value) || preview.includes(token.value);
  }

  if (token.type === "id") {
    return card.metadata.id === Number(token.value);
  }

  if (token.type === "label") {
    const labels = card.metadata.labels || [];
    return labels.some(l => l.toLowerCase().includes(token.value));
  }

  if (token.type === "date") {
    if (!card.metadata.created) {
      return false;
    }
    const created = new Date(card.metadata.created);
    const y = created.getFullYear();
    const m = String(created.getMonth() + 1).padStart(2, "0");
    const d = String(created.getDate()).padStart(2, "0");
    const dateStr = `${y}-${m}-${d}`;
    return dateStr.startsWith(token.value);
  }

  return false;
}

// Filters board data by the search query. All tokens must match (AND logic).
function filterBoard(lists: BoardLists, query: string): BoardLists {
  const tokens = parseSearchTokens(query);
  if (tokens.length === 0) {
    return lists;
  }

  const result: BoardLists = {};
  for (const key of Object.keys(lists)) {
    result[key] = lists[key].filter(
      card => tokens.every(t => cardMatchesToken(card, t))
    );
  }
  return result;
}

// Filters boardData by the current search query, returning matching cards per list.
export const filteredBoardData: Readable<BoardLists> = derived([boardData, searchQuery], ([$boardData, $searchQuery]) => {
  if (!$searchQuery.trim()) {
    return $boardData;
  }
  return filterBoard($boardData, $searchQuery);
});

export const toasts: Writable<Toast[]> = writable([]);
let toastId = 0;

// Adds a toast notification that auto-dismisses after a timeout.
export function addToast(message: string, duration: number = 4000): void {
    const id = ++toastId;
    toasts.update(t => [...t, { id, message }]);

    setTimeout(() => {
        toasts.update(t => t.filter(item => item.id !== id));
    }, duration);
}
