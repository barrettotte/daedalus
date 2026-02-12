import { writable } from 'svelte/store';

export const boardData = writable({});
export const boardConfig = writable({});
export const isLoaded = writable(false);
export const selectedCard = writable(null);
export const draftListKey = writable(null);
export const draftPosition = writable("top");
export const showMetrics = writable(false);
export const labelsExpanded = writable(true);
export const dragState = writable(null);
export const dropTarget = writable(null);

// Updates a single card in the boardData store by matching filePath.
export function updateCardInBoard(updatedCard) {
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
export function removeCardFromBoard(filePath) {
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

// Adds a new card to the given list — prepends for "top", appends for "bottom".
export function addCardToBoard(listKey, card, position = "top") {
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
export function moveCardInBoard(filePath, sourceListKey, targetListKey, targetIndex, newListOrder) {
    boardData.update(lists => {
        const srcIdx = lists[sourceListKey].findIndex(c => c.filePath === filePath);
        if (srcIdx === -1) {
            return lists;
        }

        const card = { ...lists[sourceListKey][srcIdx] };
        card.metadata = { ...card.metadata, list_order: newListOrder };

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
export function computeListOrder(cards, targetIndex) {
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

// sort lists based on folder naming convention (01_, 02_, ...)
export const sortedListKeys = (lists) => {
    return Object.keys(lists).sort();
};

export const toasts = writable([]);
let toastId = 0;

// Adds a toast notification that auto-dismisses after a timeout.
export function addToast(message, duration = 4000) {
    const id = ++toastId;
    toasts.update(t => [...t, { id, message }]);
    setTimeout(() => {
        toasts.update(t => t.filter(item => item.id !== id));
    }, duration);
}
