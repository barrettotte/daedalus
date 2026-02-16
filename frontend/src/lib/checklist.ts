// Pure checklist mutation functions. Each takes a checklist array and returns
// a new array -- callers handle persistence (backend save or local assignment).

import type { daedalus } from "../../wailsjs/go/models";

export function toggleChecklistItem(items: daedalus.CheckListItem[], idx: number): daedalus.CheckListItem[] {
  return items.map((item, i) => (i === idx ? { ...item, done: !item.done } : { ...item }));
}

export function addChecklistItem(items: daedalus.CheckListItem[], desc: string): daedalus.CheckListItem[] {
  const maxIdx = items.length > 0 ? Math.max(...items.map(i => i.idx)) : -1;
  const newItem = { idx: maxIdx + 1, desc, done: false } as daedalus.CheckListItem;
  return [...items, newItem];
}

export function editChecklistItem(items: daedalus.CheckListItem[], idx: number, desc: string): daedalus.CheckListItem[] {
  return items.map((item, i) => (i === idx ? { ...item, desc } : { ...item }));
}

export function removeChecklistItem(items: daedalus.CheckListItem[], idx: number): daedalus.CheckListItem[] {
  return items.filter((_, i) => i !== idx);
}

export function reorderChecklistItem(items: daedalus.CheckListItem[], fromIdx: number, toIdx: number): daedalus.CheckListItem[] {
  const result = [...items];
  const [moved] = result.splice(fromIdx, 1);
  result.splice(toIdx, 0, moved);
  return result;
}
