import { writable } from 'svelte/store';
import type { Writable } from 'svelte/store';
import type { daedalus } from '../../wailsjs/go/models';

// Map of list directory names to their sorted card arrays.
export type BoardLists = Record<string, daedalus.KanbanCard[]>;

// Map of list directory names to their config (title, limit).
export type BoardConfigMap = Record<string, daedalus.ListConfig>;

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

// Adds a new card to the given list. Prepends for "top", appends for "bottom".
export function addCardToBoard(listKey: string, card: daedalus.KanbanCard, position: string = "top"): void {
    boardData.update(lists => {
        if (lists[listKey]) {
            if (position === "bottom") {
                lists[listKey] = [...lists[listKey], card];
            } else {
                lists[listKey] = [card, ...lists[listKey]];
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

// Sort lists based on folder naming convention (01_, 02_, ...)
export const sortedListKeys = (lists: BoardLists): string[] => {
    return Object.keys(lists).sort();
};

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
