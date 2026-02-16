<script lang="ts">
  // Collapsible checklist with progress bar, editable title, add/remove items, and click-to-toggle.

  import Icon from "./Icon.svelte";
  import { autoFocus } from "../lib/utils";
  import type { daedalus } from "../../wailsjs/go/models";

  let {
    checklist,
    title = "",
    ontoggle,
    onadd,
    onremove,
    onedit,
    ontitlechange,
    onreorder,
    ondelete,
  }: {
    checklist: daedalus.CheckListItem[];
    title?: string;
    ontoggle: (idx: number) => void;
    onadd?: (desc: string) => void;
    onremove?: (idx: number) => void;
    onedit?: (idx: number, desc: string) => void;
    ontitlechange?: (title: string) => void;
    onreorder?: (fromIdx: number, toIdx: number) => void;
    ondelete?: () => void;
  } = $props();

  // Number of completed checklist items.
  let checkedCount = $derived(checklist.filter(i => i.done).length);

  // Completion percentage for the progress bar (rounded to nearest integer).
  let checkPct = $derived(checklist.length > 0 ? Math.round((checkedCount / checklist.length) * 100) : 0);

  // Whether the checklist items are visible or collapsed.
  let expanded = $state(true);

  // Title editing state.
  let editingTitle = $state(false);
  let editTitleValue = $state("");

  // Item editing state (-1 means not editing).
  let editingItemIdx = $state(-1);
  let editItemValue = $state("");

  // New item input state.
  let newItemDesc = $state("");

  // Drag reorder state (-1 means not dragging).
  let dragFromIdx = $state(-1);
  let dragOverIdx = $state(-1);

  // Starts dragging a checklist item with a custom ghost showing only the name.
  function handleDragStart(e: DragEvent, idx: number): void {
    dragFromIdx = idx;
    e.dataTransfer!.effectAllowed = "move";
    e.dataTransfer!.setData("text/plain", String(idx));

    const ghost = document.createElement("div");
    ghost.textContent = checklist[idx].desc;
    ghost.style.cssText = [
      "position: fixed", "top: -1000px", "left: -1000px",
      "padding: 4px 10px", "border-radius: 4px", "font-size: 0.85rem",
      "background: var(--color-bg-elevated, #2a2d35)", "color: var(--color-text-primary, #e0e0e0)",
      "white-space: nowrap", "max-width: 300px", "overflow: hidden", "text-overflow: ellipsis",
    ].join(";");

    document.body.appendChild(ghost);
    e.dataTransfer!.setDragImage(ghost, 0, 0);
    requestAnimationFrame(() => document.body.removeChild(ghost));
  }

  // Tracks which item the cursor is over during drag.
  function handleDragOver(e: DragEvent, idx: number): void {
    e.preventDefault();
    e.dataTransfer!.dropEffect = "move";
    dragOverIdx = idx;
  }

  // Drops the dragged item at the target position.
  // toIdx may equal checklist.length when dropping on the trailing drop zone.
  function handleDrop(e: DragEvent, toIdx: number): void {
    e.preventDefault();
    const dest = toIdx >= checklist.length ? checklist.length - 1 : toIdx;

    if (dragFromIdx >= 0 && dragFromIdx !== dest) {
      onreorder?.(dragFromIdx, dest);
    }
    dragFromIdx = -1;
    dragOverIdx = -1;
  }

  // Resets drag state when drag ends (e.g. cancelled).
  function handleDragEnd(): void {
    dragFromIdx = -1;
    dragOverIdx = -1;
  }

  // Returns true if the string is a URL.
  function isUrl(str: string): boolean {
    return /^https?:\/\/\S+$/.test(str);
  }

  // Opens the title for inline editing.
  function startEditTitle(e: MouseEvent): void {
    e.stopPropagation();
    editTitleValue = title || "Checklist";
    editingTitle = true;
  }

  // Saves title on blur.
  function blurTitle(): void {
    editingTitle = false;
    const val = editTitleValue.trim();
    if (val && val !== (title || "Checklist")) {
      ontitlechange?.(val);
    }
  }

  // Opens an item for inline editing.
  function startEditItem(idx: number, desc: string): void {
    editingItemIdx = idx;
    editItemValue = desc;
  }

  // Saves the edited item on blur.
  function blurEditItem(): void {
    const idx = editingItemIdx;
    editingItemIdx = -1;
    const val = editItemValue.trim();
    if (!val || val === checklist[idx]?.desc) {
      return;
    }
    onedit?.(idx, val);
  }

  // Index of the item showing a "copied" checkmark (-1 means none).
  let copiedIdx = $state(-1);
  let copiedTimer: ReturnType<typeof setTimeout> | undefined;

  // Copies a checklist item's description to the clipboard and flashes a checkmark.
  function copyItem(idx: number, desc: string): void {
    navigator.clipboard.writeText(desc);
    clearTimeout(copiedTimer);
    copiedIdx = idx;
    copiedTimer = setTimeout(() => { copiedIdx = -1; }, 1500);
  }

  // Adds a new checklist item from the input.
  function submitNewItem(): void {
    const desc = newItemDesc.trim();
    if (!desc) {
      return;
    }
    onadd?.(desc);
    newItemDesc = "";
  }
</script>

<div class="cl-wrapper">
  <div class="cl-header-row">
    <button class="cl-header" title="Toggle checklist" onclick={() => expanded = !expanded}>
      <span class="chevron" class:collapsed={!expanded}>
        <Icon name="chevron-down" size={12} />
      </span>
      {#if editingTitle}
        <input class="cl-title-input" type="text" bind:value={editTitleValue} onclick={(e) => e.stopPropagation()} onblur={blurTitle}
          onkeydown={e => e.key === 'Enter' && (e.target as HTMLInputElement).blur()} use:autoFocus
        />
      {:else}
        <span class="cl-title" role="textbox" tabindex="0" title="Click to rename" onclick={startEditTitle}
          onkeydown={(e) => e.key === 'Enter' && startEditTitle(e as unknown as MouseEvent)}
        >{title || "Checklist"}</span>
      {/if}
    </button>
    {#if !editingTitle}
      <button class="cl-action" title="Rename checklist" onclick={() => { editTitleValue = title || "Checklist"; editingTitle = true; }}>
        <Icon name="pencil" size={12} />
      </button>
    {/if}
    <button class="cl-action remove" title="Delete checklist" onclick={() => ondelete?.()}>
      <Icon name="trash" size={12} />
    </button>
  </div>
  {#if checklist.length > 0}
    <div class="cl-progress">
      <span class="cl-pct" class:complete={checkPct === 100}>{checkPct}%</span>
      <div class="progress-bar" style="flex: 1">
        <div class="progress-fill" class:complete={checkPct === 100} style="width: {checkPct}%"></div>
      </div>
      <span class="cl-count" class:complete={checkPct === 100}>{checkedCount}/{checklist.length}</span>
    </div>
  {/if}

  {#if expanded}
    <div class="cl-body">
      {#if checklist.length > 0}
        <ul class="checklist">
          {#each checklist as item, idx}
            <li class:done={item.done}
              class:dragging-item={dragFromIdx === idx}
              class:drag-over={dragOverIdx === idx && dragFromIdx !== idx}
              draggable="true"
              ondragstart={(e) => handleDragStart(e, idx)}
              ondragover={(e) => handleDragOver(e, idx)}
              ondrop={(e) => handleDrop(e, idx)}
              ondragend={handleDragEnd}
            >
              <button class="checkbox-btn" title="Toggle item" onclick={() => ontoggle(idx)}>
                <span class="checkbox" class:checked={item.done}>
                  {#if item.done}
                    <svg viewBox="0 0 16 16">
                      <rect x="1" y="1" width="14" height="14" rx="2" fill="currentColor"/>
                      <polyline points="4 8 7 11 12 5" fill="none" stroke="var(--color-bg-base)" stroke-width="2"/>
                    </svg>
                  {:else}
                    <svg viewBox="0 0 16 16">
                      <rect x="1" y="1" width="14" height="14" rx="2" fill="none" stroke="currentColor" stroke-width="1.5"/>
                    </svg>
                  {/if}
                </span>
              </button>
              {#if editingItemIdx === idx}
                <input class="edit-item-input" type="text" bind:value={editItemValue} onblur={blurEditItem}
                  onkeydown={e => e.key === 'Enter' && (e.target as HTMLInputElement).blur()} use:autoFocus
                />
              {:else}
                <span class="check-text" role="textbox" tabindex="0"
                  onclick={() => startEditItem(idx, item.desc)}
                  onkeydown={(e) => e.key === 'Enter' && startEditItem(idx, item.desc)}
                >
                  {#if isUrl(item.desc)}
                    <a href={item.desc} target="_blank" rel="noopener noreferrer"
                      onclick={(e: MouseEvent) => e.stopPropagation()}
                    >{item.desc}</a>
                  {:else}
                    {item.desc}
                  {/if}
                </span>
              {/if}
              <button class="cl-action" title="Edit item" onclick={() => startEditItem(idx, item.desc)}>
                <Icon name="pencil" size={10} />
              </button>
              <button class="cl-action" class:copied={copiedIdx === idx}
                title={copiedIdx === idx ? "Copied!" : "Copy to clipboard"}
                onclick={() => copyItem(idx, item.desc)}
              >
                <Icon name={copiedIdx === idx ? "check" : "copy"} size={10} />
              </button>
              <button class="cl-action remove" title="Remove item" onclick={() => onremove?.(idx)}>
                <Icon name="trash" size={10} />
              </button>
            </li>
          {/each}
          {#if dragFromIdx >= 0}
            <li class="drop-end" class:drag-over={dragOverIdx === checklist.length}
              ondragover={(e) => handleDragOver(e, checklist.length)}
              ondrop={(e) => handleDrop(e, checklist.length)}
            ></li>
          {/if}
        </ul>
      {/if}
      <div class="add-item-row">
        <input class="add-item-input" type="text" placeholder="Add item..." bind:value={newItemDesc}
          onkeydown={e => e.key === 'Enter' && submitNewItem()}
        />
        <button class="add-item-btn" title="Add item" onclick={submitNewItem}>
          <Icon name="plus" size={12} />
        </button>
      </div>
    </div>
  {/if}
</div>

<style lang="scss">
  .cl-wrapper {
    background: var(--overlay-subtle);
    border-radius: 6px;
    padding: 10px 12px;
  }

  .cl-header-row {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .cl-header {
    all: unset;
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
    min-width: 0;
    cursor: pointer;
    border-radius: 4px;
    box-sizing: border-box;
  }

  .cl-action {
    all: unset;
    display: flex;
    align-items: center;
    flex-shrink: 0;
    color: var(--color-text-muted);
    cursor: pointer;
    padding: 2px;
    border-radius: 3px;
    opacity: 0;

    &:hover {
      color: var(--color-text-primary);
    }

    &.remove:hover {
      color: var(--color-error);
    }

    &.copied {
      opacity: 1;
      color: var(--color-success);
    }
  }

  .cl-header-row:hover .cl-action,
  .checklist li:hover .cl-action {
    opacity: 1;
  }

  .chevron {
    display: flex;
    align-items: center;
    color: var(--color-text-muted);
    flex-shrink: 0;
    transition: transform 0.15s;

    &.collapsed {
      transform: rotate(-90deg);
    }
  }

  .cl-title {
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--color-text-primary);
    margin: 0;
    cursor: text;

    &:hover {
      color: var(--color-accent);
    }
  }

  .cl-title-input {
    flex: 1;
    min-width: 0;
    background: var(--color-bg-base);
    border: 1px solid var(--color-accent);
    color: var(--color-text-primary);
    font-size: 0.9rem;
    font-weight: 600;
    padding: 0 6px;
    border-radius: 4px;
    outline: none;
    box-sizing: border-box;
    font-family: inherit;
  }

  .cl-progress {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 6px;
  }

  .cl-pct,
  .cl-count {
    font-size: 0.7rem;
    font-weight: 600;
    color: var(--color-text-tertiary);
    flex-shrink: 0;

    &.complete {
      color: var(--color-success);
    }
  }

  .cl-body {
    margin-top: 8px;
  }

  .checklist {
    list-style: none;
    padding-top: 1px;
    padding-left: 0;
    padding-right: 0;
    padding-bottom: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
    max-height: 400px;
    overflow-y: auto;

    li {
      padding: 4px;
      font-size: 0.85rem;
      display: flex;
      gap: 8px;
      align-items: center;
      border-radius: 4px;

      &:hover {
        background: var(--overlay-hover-faint);
      }

      &.done {
        opacity: 0.6;
      }

      &.done .check-text {
        text-decoration: line-through;
        color: var(--color-text-muted);
      }

      &.dragging-item {
        opacity: 0.4;
      }

      &.drag-over {
        position: relative;

        &::before {
          content: "";
          position: absolute;
          top: -1px;
          left: 28px;
          right: 12px;
          height: 2px;
          background: var(--color-accent);
        }
      }

      &.drop-end {
        min-height: 12px;
        padding: 0;

        &:hover {
          background: none;
        }
      }
    }
  }

  .checkbox-btn {
    all: unset;
    cursor: pointer;
    display: flex;
    align-items: center;
    flex-shrink: 0;
  }

  .checkbox {
    flex-shrink: 0;
    width: 16px;
    height: 16px;
    color: var(--color-text-secondary);

    &.checked {
      color: var(--color-accent);
    }

    svg {
      width: 16px;
      height: 16px;
    }
  }

  .check-text {
    flex: 1;
    min-width: 0;
    line-height: 1.4;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    padding: 1px 0;
    cursor: pointer;

    a {
      color: var(--color-accent);
      text-decoration: none;
      line-height: inherit;
      display: inline;

      &:hover {
        text-decoration: underline;
      }
    }
  }

  .edit-item-input {
    flex: 1;
    min-width: 0;
    background: var(--color-bg-base);
    border: 1px solid var(--color-accent);
    color: var(--color-text-primary);
    font-size: 0.85rem;
    padding: 1px 6px;
    border-radius: 4px;
    outline: none;
    box-sizing: border-box;
    font-family: inherit;
  }

  .add-item-row {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-top: 6px;
    padding-top: 6px;
    border-top: 1px solid var(--overlay-hover-light);
  }

  .add-item-input {
    flex: 1;
    min-width: 0;
    background: transparent;
    border: none;
    color: var(--color-text-primary);
    font-size: 0.8rem;
    padding: 2px 4px;
    outline: none;
    box-sizing: border-box;

    &::placeholder {
      color: var(--color-text-muted);
    }

    &:focus {
      background: var(--color-bg-base);
      border-radius: 4px;
    }
  }

  .add-item-btn {
    all: unset;
    display: flex;
    align-items: center;
    flex-shrink: 0;
    color: var(--color-text-muted);
    cursor: pointer;
    padding: 2px;
    border-radius: 3px;

    &:hover {
      color: var(--color-accent);
    }
  }
</style>
