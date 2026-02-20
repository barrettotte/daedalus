// List column drag-and-drop reordering.
// Manages drag state, drop position calculation, and list order persistence.

import { get, writable } from "svelte/store";
import type { Writable } from "svelte/store";
import { boardData, listOrder, sortedListKeys, saveWithToast } from "../stores/board";
import { dragPos, handleAutoScroll, stopAutoScroll } from "./drag";
import { SaveListOrder } from "../../wailsjs/go/main/App";

// Which list column is currently being dragged (null when idle).
export const listDragging: Writable<string | null> = writable(null);

// Which list column the cursor is hovering over during drag.
export const listDropTarget: Writable<string | null> = writable(null);

// Which side of the target column to insert on.
export const listDropSide: Writable<"left" | "right" | null> = writable(null);

// Visual drop line position (fixed coordinates).
export const dropLineX: Writable<number> = writable(0);
export const dropLineTop: Writable<number> = writable(0);
export const dropLineHeight: Writable<number> = writable(0);

// Document-level dragover handler active during list drags.
// Updates ghost position and triggers auto-scroll at edges.
function listDragOverHandler(e: DragEvent): void {
  e.preventDefault();
  dragPos.set({ x: e.clientX, y: e.clientY });
  handleAutoScroll(e);
}

// Begins a list column drag operation and attaches a document-level dragover listener.
export function startListDrag(listKey: string, isPinned: boolean): void {
  if (isPinned) {
    return;
  }
  listDragging.set(listKey);
  document.addEventListener("dragover", listDragOverHandler as EventListener);
}

// Ends a list column drag operation and removes the document-level listener.
export function endListDrag(): void {
  listDragging.set(null);
  listDropTarget.set(null);
  listDropSide.set(null);

  stopAutoScroll();
  document.removeEventListener("dragover", listDragOverHandler as EventListener);
}

// Computes drop position from cursor X relative to scrollable column midpoints.
// container is the board container element; scrollableKeys is the list of non-pinned list keys.
export function computeListDragOver(e: DragEvent, container: HTMLElement, scrollableKeys: string[]): void {
  if (!get(listDragging)) {
    return;
  }

  const columns = container.querySelectorAll(".list-column:not(.pinned-left):not(.pinned-right)");
  if (columns.length === 0) {
    return;
  }

  // Find the insertion edge closest to the cursor.
  // Bias thresholds at edges so first/last positions are easier to hit:
  // first column splits at 2/3, last at 1/3, interior at 1/2.
  let targetKey = scrollableKeys[0];
  let side: "left" | "right" = "left";
  const last = columns.length - 1;

  for (let i = 0; i < columns.length; i++) {
    const rect = columns[i].getBoundingClientRect();
    const bias = i === 0 ? 0.67 : i === last ? 0.33 : 0.5;
    const splitX = rect.left + rect.width * bias;

    if (e.clientX < splitX) {
      targetKey = scrollableKeys[i];
      side = "left";
      break;
    }

    // Past this column's split point -- tentatively place after it
    targetKey = scrollableKeys[i];
    side = "right";
  }

  listDropTarget.set(targetKey);
  listDropSide.set(side);

  // Position the visual drop line at the chosen edge
  const idx = scrollableKeys.indexOf(targetKey);
  const col = columns[idx] as HTMLElement;
  const rect = col.getBoundingClientRect();

  dropLineX.set(side === "left" ? rect.left - 6 : rect.right + 6);
  dropLineTop.set(rect.top);
  dropLineHeight.set(rect.height);
}

// Handles dropping a list column, reordering and persisting.
export function handleListDrop(e: DragEvent): void {
  e.preventDefault();

  const dragging = get(listDragging);
  const target = get(listDropTarget);
  const side = get(listDropSide);

  if (!dragging || !target || dragging === target) {
    endListDrag();
    return;
  }

  const allKeys = sortedListKeys(get(boardData), get(listOrder));
  const srcIdx = allKeys.indexOf(dragging);
  if (srcIdx === -1) {
    endListDrag();
    return;
  }

  const reordered = [...allKeys];
  reordered.splice(srcIdx, 1);

  // Find target position after source removal, then adjust for drop side.
  let insertIdx = reordered.indexOf(target);
  if (insertIdx === -1) {
    endListDrag();
    return;
  }

  if (side === "right") {
    insertIdx += 1;
  }
  reordered.splice(insertIdx, 0, dragging);

  listOrder.set(reordered);
  saveWithToast(SaveListOrder(reordered), "save list order");
  endListDrag();
}

// Cleanup handler for onDestroy -- removes document-level listener if still active.
export function cleanupListDrag(): void {
  document.removeEventListener("dragover", listDragOverHandler as EventListener);
}
