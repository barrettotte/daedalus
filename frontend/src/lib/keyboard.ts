// Board-level keyboard shortcut handler. 
// Maps keys to card/list navigation, modal opening, and search.

import type { daedalus } from '../../wailsjs/go/models';
import type { BoardLists, FocusState } from '../stores/board';
import { HALF_COLLAPSED_CARD_LIMIT } from './utils';

// Snapshot of board state needed for keyboard shortcut resolution.
export interface KeyboardState {
  showAbout: boolean;
  showLabelEditor: boolean;
  showKeyboardHelp: boolean;
  draftListKey: string | null;
  selectedCard: daedalus.KanbanCard | null;
  focusedCard: FocusState | null;
  boardData: BoardLists;
  sortedKeys: string[];
  collapsedLists: Set<string>;
  halfCollapsedLists: Set<string>;
}

// Callbacks the keyboard handler can invoke.
export interface KeyboardActions {
  setShowAbout: (v: boolean) => void;
  setShowLabelEditor: (v: boolean) => void;
  setShowKeyboardHelp: (v: boolean) => void;
  setFocusedCard: (v: FocusState | null) => void;
  openCard: (card: daedalus.KanbanCard) => void;
  openCardEdit: (card: daedalus.KanbanCard) => void;
  openSearch: (prefill?: string) => void;
  createCard: (listKey: string) => void;
  createCardDefault: () => void;
  scrollListIntoView: (key: string) => void;
}

// Processes a global keydown event against board state and dispatches actions.
export function handleBoardKeydown(e: KeyboardEvent, state: KeyboardState, actions: KeyboardActions): void {
  const tag = (e.target as HTMLElement).tagName;
  const isTyping = tag === "INPUT" || tag === "TEXTAREA" || (e.target as HTMLElement).isContentEditable;

  // Escape closes overlays first; all other keys ignored while they're open.
  if (state.showAbout) {
    if (e.key === "Escape") {
      e.preventDefault();
      actions.setShowAbout(false);
    }
    return;
  }
  if (state.showLabelEditor) {
    if (e.key === "Escape") {
      e.preventDefault();
      actions.setShowLabelEditor(false);
    }
    return;
  }
  if (state.showKeyboardHelp) {
    if (e.key === "Escape") {
      e.preventDefault();
      actions.setShowKeyboardHelp(false);
    }
    return;
  }

  // Skip all shortcuts when typing in inputs
  if (isTyping) {
    return;
  }

  // Skip when modal is open (CardDetail handles its own keys).
  // Clear focus on Escape so the highlight disappears when the modal closes.
  if (state.selectedCard || state.draftListKey) {
    if (e.key === "Escape") {
      actions.setFocusedCard(null);
    }
    return;
  }

  const keys = state.sortedKeys;
  if (keys.length === 0) {
    return;
  }

  // ? - Toggle keyboard help
  if (e.key === "?") {
    e.preventDefault();
    actions.setShowKeyboardHelp(true);
    return;
  }

  // / - Open and focus search bar
  if (e.key === "/") {
    e.preventDefault();
    actions.openSearch();
    return;
  }

  // # - Open search with # prefix for card ID jump
  if (e.key === "#") {
    e.preventDefault();
    actions.openSearch("#");
    return;
  }

  // N - Create new card in leftmost list
  if (e.key === "n" && !e.metaKey && !e.ctrlKey && !e.altKey) {
    e.preventDefault();
    actions.createCardDefault();
    return;
  }

  // Arrow keys - navigate focus (skip if Ctrl/Cmd held)
  if ((e.key === "ArrowUp" || e.key === "ArrowDown" || e.key === "ArrowLeft" || e.key === "ArrowRight") && !e.ctrlKey && !e.metaKey) {
    e.preventDefault();

    const focus = state.focusedCard;
    if (!focus) {
      // No current focus - start at first card of first non-collapsed list
      for (const key of keys) {
        if (!state.collapsedLists.has(key) && (state.boardData[key] || []).length > 0) {
          actions.setFocusedCard({ listKey: key, cardIndex: 0 });
          actions.scrollListIntoView(key);
          break;
        }
      }
      return;
    }

    if (e.key === "ArrowUp") {
      if (focus.cardIndex > 0) {
        actions.setFocusedCard({ listKey: focus.listKey, cardIndex: focus.cardIndex - 1 });
      }
    } else if (e.key === "ArrowDown") {
      const cards = state.boardData[focus.listKey] || [];
      const maxIndex = state.halfCollapsedLists.has(focus.listKey)
        ? Math.min(HALF_COLLAPSED_CARD_LIMIT - 1, cards.length - 1) : cards.length - 1;

      if (focus.cardIndex < maxIndex) {
        actions.setFocusedCard({ listKey: focus.listKey, cardIndex: focus.cardIndex + 1 });
      }
    } else if (e.key === "ArrowLeft" || e.key === "ArrowRight") {
      const listIdx = keys.indexOf(focus.listKey);
      const delta = e.key === "ArrowLeft" ? -1 : 1;
      let targetIdx = listIdx + delta;

      // Skip collapsed or empty lists
      while (targetIdx >= 0 && targetIdx < keys.length) {
        const targetKey = keys[targetIdx];
        if (!state.collapsedLists.has(targetKey) && (state.boardData[targetKey] || []).length > 0) {
          break;
        }
        targetIdx += delta;
      }

      if (targetIdx >= 0 && targetIdx < keys.length) {
        const targetKey = keys[targetIdx];
        const targetCards = state.boardData[targetKey] || [];
        const clampedIndex = Math.min(focus.cardIndex, targetCards.length - 1);
        actions.setFocusedCard({ listKey: targetKey, cardIndex: clampedIndex });
        actions.scrollListIntoView(targetKey);
      }
    }
    return;
  }

  // Enter - open focused card
  if (e.key === "Enter" && state.focusedCard) {
    e.preventDefault();
    const card = (state.boardData[state.focusedCard.listKey] || [])[state.focusedCard.cardIndex];

    if (card) {
      actions.openCard(card);
    }
    return;
  }

  // Escape - clear focus
  if (e.key === "Escape" && state.focusedCard) {
    e.preventDefault();
    actions.setFocusedCard(null);
    return;
  }

  // E - open focused card in edit mode
  if (e.key === "e" && !e.ctrlKey && !e.metaKey && state.focusedCard) {
    e.preventDefault();
    const card = (state.boardData[state.focusedCard.listKey] || [])[state.focusedCard.cardIndex];

    if (card) {
      actions.openCardEdit(card);
    }
    return;
  }

}
