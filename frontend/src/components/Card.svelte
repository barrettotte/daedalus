<script lang="ts">
  // Card preview shown in list columns. Displays title, labels, badges, and handles drag-start.

  import Icon from "./Icon.svelte";
  import CardIcon from "./CardIcon.svelte";
  import { selectedCard, labelsExpanded, labelColors, dragState, addToast, contextMenu } from "../stores/board";
  import { SaveLabelsExpanded } from "../../wailsjs/go/main/App";
  import { labelColor, formatDate, formatDateTime } from "../lib/utils";
  import type { daedalus } from "../../wailsjs/go/models";

  let { card, listKey = "", focused = false }: { card: daedalus.KanbanCard; listKey?: string; focused?: boolean } = $props();

  let meta = $derived(card.metadata);
  let isDragging = $derived($dragState?.card?.filePath === card.filePath);
  let hasChecklist = $derived(meta.checklist && meta.checklist.length > 0);
  let checkedCount = $derived(hasChecklist ? meta.checklist!.filter(i => i.done).length : 0);
  let hasDescription = $derived(card.previewText && card.previewText.replace(/^#\s+.*\n*/, "").trim().length > 0);
  let checklistComplete = $derived(hasChecklist ? checkedCount === meta.checklist!.length : false);
  let counterComplete = $derived(meta.counter ? meta.counter.current === meta.counter.max : false);

  // Sets this card as the selected card to open the detail modal.
  function openDetail(): void {
    selectedCard.set(card);
  }

  // Starts drag operation, stores card and source list in dragState.
  function handleDragStart(e: DragEvent): void {
    dragState.set({ card, sourceListKey: listKey });

    // WebKitGTK requires setData for the drop event to fire
    e.dataTransfer!.setData("text/plain", card.filePath);
    e.dataTransfer!.effectAllowed = "move";

    // Hide default drag ghost
    const ghost = document.createElement("div");
    ghost.style.width = "1px";
    ghost.style.height = "1px";
    ghost.style.opacity = "0";

    document.body.appendChild(ghost);
    e.dataTransfer!.setDragImage(ghost, 0, 0);
    requestAnimationFrame(() => document.body.removeChild(ghost));
  }

  // Ends drag operation
  function handleDragEnd(): void {
    dragState.set(null);
  }

  // Opens the right-click context menu for this card.
  function handleContextMenu(e: MouseEvent): void {
    e.preventDefault();
    contextMenu.set({ card, listKey, x: e.clientX, y: e.clientY });
  }

  // Toggles all labels board-wide between expanded text and collapsed color pills, persisting to board.yaml.
  function toggleLabels(): void {
    labelsExpanded.update(v => {
      const next = !v;
      SaveLabelsExpanded(next).catch(e => addToast(`Failed to save label state: ${e}`));
      return next;
    });
  }
</script>

<div class="card" class:dragging={isDragging} class:focused={focused} draggable="true" role="button"
  tabindex="0" ondragstart={handleDragStart} ondragend={handleDragEnd} onclick={openDetail}
  oncontextmenu={handleContextMenu} onkeydown={e => e.key === 'Enter' && openDetail()}
>
  <div class="top-row">
    {#if meta.labels && meta.labels.length > 0}
      <div class="labels">
        {#each [...meta.labels].sort() as label}
          <span class="label" class:collapsed={!$labelsExpanded} style="background: {labelColor(label, $labelColors)}"
            title={$labelsExpanded ? '' : label} role="button" tabindex="0"
            onclick={(e: MouseEvent) => { e.stopPropagation(); toggleLabels(); }}
            onkeydown={(e: KeyboardEvent) => { e.stopPropagation(); e.key === 'Enter' && toggleLabels(); }}
          >{#if $labelsExpanded}{label}{/if}</span>
        {/each}
      </div>
    {/if}
    <span class="card-id">#{meta.id}</span>
  </div>

  <div class="title">{meta.title}</div>
  {#if meta.icon}<span class="card-icon"><CardIcon name={meta.icon} size={18} /></span>{/if}

  <div class="badges">
    {#if hasDescription}
      <span class="badge-icon desc-icon"><Icon name="description" size={12} /></span>
    {/if}
    {#if meta.due}
      <span class="badge" title={formatDateTime(meta.due)}>
        <Icon name="clock" size={12} />
        {meta.due.slice(0, 10)}
      </span>
    {/if}
    {#if hasChecklist}
      <span class="badge" class:checklist-done={checklistComplete}>
        <Icon name="checklist" size={12} />
        {checkedCount}/{meta.checklist!.length}
      </span>
    {/if}
    {#if meta.counter}
      <span class="badge" class:counter-done={counterComplete}>
        <Icon name="counter" size={12} />
        {meta.counter.current}/{meta.counter.max}
      </span>
    {/if}
    {#if meta.estimate != null}
      <span class="badge" title="Estimate: {meta.estimate}h">
        <Icon name="hourglass" size={12} />
        {meta.estimate}h
      </span>
    {/if}
    {#if meta.range}
      <span class="badge" title="{formatDateTime(meta.range.start)} - {formatDateTime(meta.range.end)}">
        <Icon name="calendar" size={12} />
        {formatDate(meta.range.start)} - {formatDate(meta.range.end)}
      </span>
    {/if}
  </div>
</div>

<style lang="scss">
  .card {
    background: var(--color-bg-surface);
    border-radius: 4px;
    padding: 8px 10px;
    min-height: 54px;
    margin: 0 6px;
    border-bottom: 1px solid var(--color-border);
    color: var(--color-text-primary);
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, sans-serif;
    cursor: pointer;
    text-align: left;
    contain: content;
    position: relative;
    outline: none;

    &:hover {
      background: var(--color-bg-surface-hover);
    }

    &.dragging {
      opacity: 0.4;
    }

    &.focused {
      outline: 2px solid var(--color-accent);
      outline-offset: -2px;
    }
  }

  .title {
    font-size: 0.85rem;
    font-weight: 400;
    line-height: 1.3;
    margin: 6px 0 8px 0;
    word-break: break-word;
  }

  .card-icon {
    position: absolute;
    bottom: 8px;
    right: 10px;
    display: inline-flex;
    color: var(--color-text-tertiary);
  }

  .top-row {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .card-id {
    font-size: 0.6rem;
    color: var(--color-text-muted);
    font-family: monospace;
    flex-shrink: 0;
    margin-left: auto;
  }

  .labels {
    display: flex;
    gap: 4px;
    flex-wrap: wrap;
    margin-left: 2px;
  }

  .label {
    font-size: 0.7rem;
    font-weight: 600;
    padding: 3px 10px;
    border-radius: 4px;
    color: var(--color-text-inverse);

    &.collapsed {
      padding: 0 8px;
      height: 8px;
      min-width: 0;
      font-size: 0;
    }
  }

  .badges {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
    align-items: center;
    margin-left: -4px;
    margin-bottom: 2px;
  }

  .badge {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: 0.7rem;
    line-height: 1;
    color: var(--color-text-tertiary);
    border-radius: 3px;
    padding: 1px 4px;

    &.checklist-done, &.counter-done {
      background: var(--overlay-success);
      color: var(--color-success);
    }
  }

  .badge-icon {
    display: inline-flex;
    flex-shrink: 0;
    padding: 1px 4px;
  }

  .desc-icon {
    color: var(--color-text-tertiary);
  }
</style>
