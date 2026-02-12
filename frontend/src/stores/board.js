import { writable } from 'svelte/store';

export const boardData = writable({});
export const boardConfig = writable({});
export const isLoaded = writable(false);
export const selectedCard = writable(null);
export const showMetrics = writable(false);
export const labelsExpanded = writable(true);

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
