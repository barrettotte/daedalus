// Svelte stores for board state, card data, config, and derived views (filtering, sorting, toasts).

import { writable, derived } from 'svelte/store';
import type { Writable, Readable } from 'svelte/store';
import type { daedalus } from '../../wailsjs/go/models';
import { filterBoard } from '../lib/search';
import { SaveLabelsExpanded, SaveMinimalView, SaveTemplates } from '../../wailsjs/go/main/App';

// Map of list directory names to their sorted card arrays.
export type BoardLists = Record<string, daedalus.KanbanCard[]>;

// Map of list directory names to their config (title, limit, locked).
export type BoardConfigMap = Record<string, { title: string; limit: number; locked: boolean; color: string; icon: string }>;

// Keyboard focus position on the board (list + index within that list).
export interface FocusState {
  listKey: string;
  cardIndex: number;
}

// Timing breakdown for board loading phases.
export interface LoadProfileData {
  configMs: number;
  scanMs: number;
  mergeMs: number;
  totalMs: number;
  transferMs: number;
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

// Right-click context menu state.
export interface ContextMenuState {
  card: daedalus.KanbanCard;
  listKey: string;
  x: number;
  y: number;
}

// Toast notification entry.
export type ToastType = "error" | "success" | "info";
export interface Toast {
  id: number;
  message: string;
  type: ToastType;
}

export const boardData: Writable<BoardLists> = writable({});
export const boardTitle: Writable<string> = writable("Daedalus");
export const boardConfig: Writable<BoardConfigMap> = writable({});
export const labelColors: Writable<Record<string, string>> = writable({});
export const boardPath: Writable<string> = writable("");
export const isLoaded: Writable<boolean> = writable(false);
export const selectedCard: Writable<daedalus.KanbanCard | null> = writable(null);
export const draftListKey: Writable<string | null> = writable(null);
export const draftPosition: Writable<string> = writable("top");
export const showMetrics: Writable<boolean> = writable(false);
export const labelsExpanded: Writable<boolean> = writable(true);
export const minimalView: Writable<boolean> = writable(false);
export const dragState: Writable<DragInfo | null> = writable(null);
export const dropTarget: Writable<DropInfo | null> = writable(null);
export const focusedCard: Writable<FocusState | null> = writable(null);
export const openInEditMode: Writable<boolean> = writable(false);
export const listOrder: Writable<string[]> = writable([]);
export const loadProfile: Writable<LoadProfileData | null> = writable(null);
export const contextMenu: Writable<ContextMenuState | null> = writable(null);
export const maxCardId: Writable<number> = writable(0);

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
  maxCardId.update(current => Math.max(current, card.metadata.id));
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

// Returns true when a list is locked (no cards can be moved in or out).
export function isLocked(listKey: string, config: BoardConfigMap): boolean {
  return config[listKey]?.locked || false;
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

export const templates: Writable<daedalus.CardTemplate[]> = writable([]);

export const searchQuery: Writable<string> = writable("");

// Debounced version of searchQuery -- waits 150ms after the last keystroke.
let debounceTimer: ReturnType<typeof setTimeout> | null = null;
const debouncedQuery: Writable<string> = writable("");

searchQuery.subscribe(val => {
  if (debounceTimer) {
    clearTimeout(debounceTimer);
  }
  if (!val.trim()) {
    debouncedQuery.set("");
    return;
  }
  debounceTimer = setTimeout(() => {
    debouncedQuery.set(val);
  }, 150);
});

// Filters boardData by the debounced search query, returning matching cards per list.
export const filteredBoardData: Readable<BoardLists> = derived(
  [boardData, debouncedQuery],
  ([$boardData, $debouncedQuery]) => {
    if (!$debouncedQuery.trim()) {
      return $boardData;
    }
    return filterBoard($boardData, $debouncedQuery);
  },
);

export const toasts: Writable<Toast[]> = writable([]);
let toastId = 0;
const DEFAULT_TOAST_DURATION = 4000;

// Adds a toast notification that auto-dismisses after a timeout.
export function addToast(
    message: string,
    type: ToastType = "error",
    duration: number = DEFAULT_TOAST_DURATION,
): void {
    const id = ++toastId;
    toasts.update(t => [...t, { id, message, type }]);

    setTimeout(() => {
      toasts.update(t => t.filter(item => item.id !== id));
    }, duration);
}

// Fire-and-forget wrapper that catches promise rejections and shows an error toast.
export function saveWithToast(promise: Promise<unknown>, action: string): void {
  promise.catch(e => addToast(`Failed to ${action}: ${e}`));
}

// Syncs a card's filePath and listName in the store after a cross-list move renames the file on disk.
export function syncCardAfterMove(targetListKey: string, cardId: number, result: daedalus.KanbanCard): void {
  boardData.update(lists => {
    const tc = lists[targetListKey];
    if (tc) {
      const idx = tc.findIndex(c => c.metadata.id === cardId);
      if (idx !== -1) {
        tc[idx] = { ...tc[idx], filePath: result.filePath, listName: result.listName } as daedalus.KanbanCard;
      }
    }
    return lists;
  });
}

// Finds a card by its numeric ID across all board lists.
// Returns the card if found, or null otherwise.
export function findCardById(lists: BoardLists, id: number): daedalus.KanbanCard | null {
  for (const cards of Object.values(lists)) {
    const found = cards.find(c => c.metadata.id === id);
    if (found) {
      return found;
    }
  }
  return null;
}

// Toggles labels between expanded text and collapsed color pills on all cards, persisting to board.yaml.
export function toggleLabelsExpanded(): void {
  labelsExpanded.update(v => {
    const next = !v;
    saveWithToast(SaveLabelsExpanded(next), "save label state");
    return next;
  });
}

// Toggles minimal card view on/off, persisting to board.yaml.
export function toggleMinimalView(): void {
  minimalView.update(v => {
    const next = !v;
    saveWithToast(SaveMinimalView(next), "save minimal view state");
    return next;
  });
}
