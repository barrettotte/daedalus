<script lang="ts">
  import Icon from "./Icon.svelte";
  import { selectedCard, labelsExpanded, labelColors, dragState, addToast } from "../stores/board";
  import { SaveLabelsExpanded } from "../../wailsjs/go/main/App";
  import { labelColor, formatDate } from "../lib/utils";
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
  onkeydown={e => e.key === 'Enter' && openDetail()}
>
  {#if meta.labels && meta.labels.length > 0}
    <div class="labels">
      {#each meta.labels as label}
        <span class="label" class:collapsed={!$labelsExpanded} style="background: {labelColor(label, $labelColors)}"
          title={$labelsExpanded ? '' : label} role="button" tabindex="0"
          onclick={(e: MouseEvent) => { e.stopPropagation(); toggleLabels(); }}
          onkeydown={(e: KeyboardEvent) => { e.stopPropagation(); e.key === 'Enter' && toggleLabels(); }}
        >{#if $labelsExpanded}{label}{/if}</span>
      {/each}
    </div>
  {/if}

  <div class="title">{meta.title}</div>

  <div class="badges">
    {#if hasDescription}
      <svg class="badge-icon desc-icon" viewBox="0 0 24 24">
        <line x1="4" y1="6" x2="20" y2="6" stroke="currentColor" stroke-width="2"/>
        <line x1="4" y1="12" x2="16" y2="12" stroke="currentColor" stroke-width="2"/>
        <line x1="4" y1="18" x2="12" y2="18" stroke="currentColor" stroke-width="2"/>
      </svg>
    {/if}
    {#if meta.due}
      <span class="badge">
        <svg class="badge-icon" viewBox="0 0 24 24">
          <circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="2"/>
          <polyline points="12 6 12 12 16 14" fill="none" stroke="currentColor" stroke-width="2"/>
        </svg>
        {meta.due.slice(0, 10)}
      </span>
    {/if}
    {#if hasChecklist}
      <span class="badge" class:checklist-done={checklistComplete}>
        <Icon name="checklist" size={16} />
        {checkedCount}/{meta.checklist!.length}
      </span>
    {/if}
    {#if meta.counter}
      <span class="badge" class:counter-done={counterComplete}>
        <svg class="badge-icon" viewBox="0 0 24 24">
          <line x1="4" y1="9" x2="20" y2="9" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          <line x1="4" y1="15" x2="20" y2="15" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          <line x1="9" y1="4" x2="9" y2="20" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          <line x1="15" y1="4" x2="15" y2="20" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
        </svg>
        {meta.counter.current}/{meta.counter.max}
      </span>
    {/if}
    {#if meta.range}
      <span class="badge">
        <svg class="badge-icon" viewBox="0 0 24 24">
          <rect x="3" y="4" width="18" height="18" rx="2" fill="none" stroke="currentColor" stroke-width="2"/>
          <line x1="3" y1="10" x2="21" y2="10" stroke="currentColor" stroke-width="2"/>
          <line x1="16" y1="2" x2="16" y2="6" stroke="currentColor" stroke-width="2"/>
          <line x1="8" y1="2" x2="8" y2="6" stroke="currentColor" stroke-width="2"/>
        </svg>
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
    margin: 0 6px;
    border-bottom: 1px solid rgba(0, 0, 0, 0.25);
    color: var(--color-text-primary);
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, sans-serif;
    cursor: pointer;
    text-align: left;
    contain: content;

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
    margin: 10px 0 8px 0;
    word-break: break-word;
  }

  .labels {
    display: flex;
    gap: 4px;
    flex-wrap: wrap;
    margin: 2px 0 0 0;
  }

  .label {
    font-size: 0.65rem;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 3px;
    color: #fff;

    &.collapsed {
      padding: 0;
      min-width: 28px;
      height: 8px;
      font-size: 0;
    }
  }

  .badges {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
    align-items: center;
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
    width: 12px;
    height: 12px;
    flex-shrink: 0;
  }

  .desc-icon {
    color: var(--color-text-muted);
    opacity: 0.6;
  }
</style>
