import { writable } from 'svelte/store';

export const boardData = writable({});
export const isLoaded = writable(false);
export const selectedCard = writable(null);

// sort lists based on folder naming convention (01_, 02_, ...)
export const sortedListKeys = (lists) => {
    return Object.keys(lists).sort();
};
