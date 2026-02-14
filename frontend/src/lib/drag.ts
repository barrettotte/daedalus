import { get, writable } from "svelte/store";
import type { Writable } from "svelte/store";
import {
  boardData, boardConfig, dragState, dropTarget,
  moveCardInBoard, computeListOrder, isAtLimit, addToast,
} from "../stores/board";
import { MoveCard } from "../../wailsjs/go/main/App";
import type { daedalus } from "../../wailsjs/go/models";

// Reactive cursor position for the drag ghost overlay.
export const dragPos: Writable<{ x: number; y: number }> = writable({ x: 0, y: 0 });

// Reference to the board container element for horizontal auto-scroll.
let boardContainerEl: HTMLDivElement | undefined;

// Auto-scroll state (module-level, not reactive).
let autoScrollRaf: number | null = null;
let autoScrollVContainer: Element | null = null;
let autoScrollVSpeed = 0;
let autoScrollHSpeed = 0;

// Tracks elements with active drop indicator classes for efficient cleanup.
let activeIndicators: Set<Element> = new Set();

// Sets the board container DOM reference (called from component onMount).
export function setBoardContainer(el: HTMLDivElement | undefined): void {
  boardContainerEl = el;
}

// Returns the board container element (for binding in the component template).
export function getBoardContainer(): HTMLDivElement | undefined {
  return boardContainerEl;
}

// Clears drop indicator classes from tracked elements.
export function clearDropIndicators(): void {
  for (const el of activeIndicators) {
    el.classList.remove("drop-above", "drop-below", "drop-top", "drop-bottom");
  }
  activeIndicators.clear();
}

// Adds a drop indicator class to an element and tracks it for cleanup.
function addIndicator(el: Element, cls: string): void {
  el.classList.add(cls);
  activeIndicators.add(el);
}

// Allows the element to be a valid drop target.
export function handleDragEnter(e: DragEvent): void {
  e.preventDefault();
  e.dataTransfer!.dropEffect = "move";
}

// Positions the drop indicator and triggers auto-scroll during drag.
export function handleDragOver(e: DragEvent, listKey: string): void {
  e.preventDefault();
  dragPos.set({ x: e.clientX, y: e.clientY });

  const drag = get(dragState);
  if (!drag) {
    e.dataTransfer!.dropEffect = "move";
    return;
  }

  const lists = get(boardData);
  const config = get(boardConfig);

  // Block cross-list drops into lists that are at their card limit.
  if (drag.sourceListKey !== listKey && isAtLimit(listKey, lists, config)) {
    e.dataTransfer!.dropEffect = "none";
    clearDropIndicators();
    return;
  }

  e.dataTransfer!.dropEffect = "move";
  clearDropIndicators();

  // Find the closest item-slot under the cursor
  const slot = (e.target as HTMLElement).closest("[data-card-id]");
  if (slot) {
    const rect = slot.getBoundingClientRect();
    const midY = rect.top + rect.height / 2;

    if (e.clientY < midY) {
      // Top half - insert before this card
      addIndicator(slot, "drop-above");
      dropTarget.set({
        listKey,
        cardId: Number((slot as HTMLElement).dataset.cardId),
        position: "above",
      });
    } else {
      // Bottom half - insert after this card
      const next = slot.nextElementSibling;

      if (next && next.hasAttribute("data-card-id")) {
        addIndicator(next, "drop-above");
        dropTarget.set({
          listKey,
          cardId: Number((next as HTMLElement).dataset.cardId),
          position: "above",
        });
      } else {
        // Last card - show indicator below
        addIndicator(slot, "drop-below");
        dropTarget.set({
          listKey,
          cardId: Number((slot as HTMLElement).dataset.cardId),
          position: "below",
        });
      }
    }
  } else {
    // No card under cursor. Determine if near top or bottom of the list
    const listBody = e.currentTarget as HTMLElement;
    const rect = listBody.getBoundingClientRect();
    const cards = lists[listKey] || [];

    if (cards.length > 0 && e.clientY < rect.top + rect.height / 3) {
      dropTarget.set({ listKey, cardId: cards[0].metadata.id, position: "above" });
      addIndicator(listBody, "drop-top");
    } else {
      dropTarget.set({ listKey, cardId: null, position: "below" });
    }
  }

  // Auto-scroll: vertical (inside the list's scroll container) and horizontal (board)
  handleAutoScroll(e);
}

// Clears indicators when the cursor leaves a drop zone.
export function handleDragLeave(e: DragEvent): void {
  const related = e.relatedTarget as Node | null;
  if (related && !(e.currentTarget as HTMLElement).contains(related)) {
    clearDropIndicators();
  }
}

// Executes the card move on drop. Calls onError to reload the board on failure.
export async function handleDrop(e: DragEvent, listKey: string, onError: () => void): Promise<void> {
  e.preventDefault();
  clearDropIndicators();

  const drag = get(dragState);
  const drop = get(dropTarget);
  dragState.set(null);
  dropTarget.set(null);

  if (!drag || !drop) {
    return;
  }
  const lists = get(boardData);
  const config = get(boardConfig);

  // Block cross-list moves into lists at their card limit.
  if (drag.sourceListKey !== listKey && isAtLimit(listKey, lists, config)) {
    addToast("List is at its card limit");
    return;
  }

  const cards = lists[listKey] || [];
  let targetIndex: number;

  if (drop.cardId == null) {
    // Drop on empty list or empty area
    targetIndex = cards.length;
  } else {
    const cardIdx = cards.findIndex(c => c.metadata.id === drop.cardId);

    if (cardIdx === -1) {
      targetIndex = cards.length;
    } else {
      targetIndex = drop.position === "above" ? cardIdx : cardIdx + 1;
    }
  }

  // Adjust targetIndex if dragging within the same list and the source is before the target
  const sourceCards = lists[drag.sourceListKey] || [];
  const sourceIdx = sourceCards.findIndex(c => c.filePath === drag.card.filePath);

  if (drag.sourceListKey === listKey && sourceIdx !== -1) {
    // No-op if dropping at the same position
    if (targetIndex === sourceIdx || targetIndex === sourceIdx + 1) {
      return;
    }
    // Adjust for removal of source card
    if (sourceIdx < targetIndex) {
      targetIndex--;
    }
  }

  // Build the target cards array without the dragged card for computing list_order
  const targetCards = (drag.sourceListKey === listKey)
    ? cards.filter(c => c.filePath !== drag.card.filePath)
    : cards;

  const newListOrder = computeListOrder(targetCards, targetIndex);

  // Capture original path before optimistic update (needed for the API call)
  const originalPath = drag.card.filePath;
  moveCardInBoard(originalPath, drag.sourceListKey, listKey, targetIndex, newListOrder);

  try {
    const result = await MoveCard(originalPath, listKey, newListOrder);

    // Cross-list moves change the filePath on disk. Sync the store with the backend response
    if (result.filePath !== originalPath) {
      boardData.update(bLists => {
        const listCards = bLists[listKey];
        if (listCards) {
          const idx = listCards.findIndex(c => c.metadata.id === drag.card.metadata.id);

          if (idx !== -1) {
            listCards[idx] = {
              ...listCards[idx],
              filePath: result.filePath,
              listName: result.listName,
            } as daedalus.KanbanCard;
          }
        }
        return bLists;
      });
    }
  } catch (err) {
    addToast(`Failed to move card: ${err}`);
    onError();
  }
}

// Handles drag over the list header area, showing indicator at top.
export function handleHeaderDragOver(e: DragEvent, listKey: string): void {
  e.preventDefault();
  dragPos.set({ x: e.clientX, y: e.clientY });

  const drag = get(dragState);
  if (!drag) {
    e.dataTransfer!.dropEffect = "move";
    return;
  }

  const lists = get(boardData);
  const config = get(boardConfig);

  // Block cross-list drops into lists that are at their card limit.
  if (drag.sourceListKey !== listKey && isAtLimit(listKey, lists, config)) {
    e.dataTransfer!.dropEffect = "none";
    clearDropIndicators();
    return;
  }

  e.dataTransfer!.dropEffect = "move";
  clearDropIndicators();
  const cards = lists[listKey] || [];
  if (cards.length > 0) {
    dropTarget.set({ listKey, cardId: cards[0].metadata.id, position: "above" });
  } else {
    dropTarget.set({ listKey, cardId: null, position: "below" });
  }

  // Show indicator at the top of the list body
  const listCol = (e.currentTarget as HTMLElement).closest(".list-column");
  if (listCol) {
    const listBody = listCol.querySelector(".list-body");
    if (listBody) {
      addIndicator(listBody, "drop-top");
    }
  }
}

// Handles drag over the footer add-button area, targeting end of list.
export function handleFooterDragOver(e: DragEvent, listKey: string): void {
  e.preventDefault();
  dragPos.set({ x: e.clientX, y: e.clientY });

  const drag = get(dragState);
  if (!drag) {
    e.dataTransfer!.dropEffect = "move";
    return;
  }

  const lists = get(boardData);
  const config = get(boardConfig);

  // Block cross-list drops into lists that are at their card limit.
  if (drag.sourceListKey !== listKey && isAtLimit(listKey, lists, config)) {
    e.dataTransfer!.dropEffect = "none";
    clearDropIndicators();
    return;
  }

  e.dataTransfer!.dropEffect = "move";
  clearDropIndicators();
  dropTarget.set({ listKey, cardId: null, position: "below" });
  handleAutoScroll(e);
}

// Updates auto-scroll speeds based on cursor proximity to viewport edges.
export function handleAutoScroll(e: DragEvent): void {
  const edgeSize = 40;
  const speed = 10;
  let hSpeed = 0;
  let vSpeed = 0;
  let vContainer: Element | null = null;

  // Horizontal scroll
  if (boardContainerEl) {
    const rect = boardContainerEl.getBoundingClientRect();
    if (e.clientX < rect.left + edgeSize) {
      hSpeed = -speed;
    } else if (e.clientX > rect.right - edgeSize) {
      hSpeed = speed;
    }
  }

  // Vertical scroll
  const target = e.target as HTMLElement;
  vContainer = target.closest(".virtual-scroll-container") || (e.currentTarget as HTMLElement).querySelector(".virtual-scroll-container");

  if (vContainer) {
    const rect = vContainer.getBoundingClientRect();
    if (e.clientY < rect.top + edgeSize) {
      vSpeed = -speed;
    } else if (e.clientY > rect.bottom - edgeSize) {
      vSpeed = speed;
    }
  }

  autoScrollHSpeed = hSpeed;
  autoScrollVSpeed = vSpeed;
  autoScrollVContainer = vContainer;

  // Start the loop if scrolling is needed and not already running
  if ((hSpeed !== 0 || vSpeed !== 0) && !autoScrollRaf) {
    autoScrollTick();
  }
}

// Continuous scroll animation loop.
function autoScrollTick(): void {
  if (autoScrollHSpeed === 0 && autoScrollVSpeed === 0) {
    autoScrollRaf = null;
    return;
  }
  if (autoScrollHSpeed !== 0 && boardContainerEl) {
    boardContainerEl.scrollLeft += autoScrollHSpeed;
  }
  if (autoScrollVSpeed !== 0 && autoScrollVContainer) {
    autoScrollVContainer.scrollTop += autoScrollVSpeed;
  }
  autoScrollRaf = requestAnimationFrame(autoScrollTick);
}

// Stops the auto-scroll loop and clears related state.
export function stopAutoScroll(): void {
  autoScrollHSpeed = 0;
  autoScrollVSpeed = 0;
  autoScrollVContainer = null;

  if (autoScrollRaf) {
    cancelAnimationFrame(autoScrollRaf);
    autoScrollRaf = null;
  }
}
