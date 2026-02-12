<script>
  import { selectedCard, labelsExpanded, addToast } from "../stores/board.js";
  import { SaveLabelsExpanded } from "../../wailsjs/go/main/App";
  import { labelColor } from "../lib/utils.js";

  export let card;
  $: meta = card.metadata;
  $: isOverdue = meta.due ? new Date(meta.due) < new Date() : false;
  $: hasChecklist = meta.checklist && meta.checklist.length > 0;
  $: checkedCount = hasChecklist ? meta.checklist.filter(i => i.done).length : 0;
  $: hasDescription = card.previewText && card.previewText.replace(/^#\s+.*\n*/, "").trim().length > 0;

  // Sets this card as the selected card to open the detail modal.
  function openDetail() {
    selectedCard.set(card);
  }

  // Toggles all labels board-wide between expanded text and collapsed color pills, persisting to board.yaml.
  function toggleLabels() {
    labelsExpanded.update(v => {
      const next = !v;
      SaveLabelsExpanded(next).catch(e => addToast(`Failed to save label state: ${e}`));
      return next;
    });
  }
</script>

<div class="card" on:click={openDetail} on:keydown={e => e.key === 'Enter' && openDetail()}>
  {#if meta.labels && meta.labels.length > 0}
    <div class="labels">
      {#each meta.labels as label}
        <span class="label" class:collapsed={!$labelsExpanded} style="background: {labelColor(label)}" title={$labelsExpanded ? '' : label} on:click|stopPropagation={toggleLabels} on:keydown|stopPropagation={e => e.key === 'Enter' && toggleLabels()}>{#if $labelsExpanded}{label}{/if}</span>
      {/each}
    </div>
  {/if}

  <div class="title">{meta.title}</div>

  <div class="badges">
    {#if meta.due}
      <span class="badge" class:overdue={isOverdue} class:on-time={!isOverdue}>
        <svg class="badge-icon" viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="2"/><polyline points="12 6 12 12 16 14" fill="none" stroke="currentColor" stroke-width="2"/></svg>
        {meta.due.slice(0, 10)}
      </span>
    {/if}
    {#if hasChecklist}
      <span class="badge checklist-badge">
        <svg class="badge-icon" viewBox="0 0 24 24"><polyline points="9 11 12 14 22 4" fill="none" stroke="currentColor" stroke-width="2"/><path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" fill="none" stroke="currentColor" stroke-width="2"/></svg>
        {checkedCount}/{meta.checklist.length}
      </span>
    {/if}
    {#if hasDescription}
      <svg class="badge-icon desc-icon" viewBox="0 0 24 24"><line x1="4" y1="6" x2="20" y2="6" stroke="currentColor" stroke-width="2"/><line x1="4" y1="12" x2="16" y2="12" stroke="currentColor" stroke-width="2"/><line x1="4" y1="18" x2="12" y2="18" stroke="currentColor" stroke-width="2"/></svg>
    {/if}
  </div>
</div>

<style>
  .card {
    background: #2b303b;
    border-radius: 4px;
    padding: 8px 10px;
    margin: 0 6px;
    border-bottom: 1px solid rgba(0, 0, 0, 0.25);
    color: #c7d1db;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, sans-serif;
    cursor: pointer;
    text-align: left;
    contain: content;
  }

  .card:hover {
    background: #333846;
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
  }

  .label.collapsed {
    padding: 0;
    min-width: 28px;
    height: 8px;
    font-size: 0;
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
    color: #8c9bab;
    border-radius: 3px;
    padding: 1px 4px;
  }

  .badge.on-time {
    background: rgba(75, 206, 151, 0.15);
    color: #4bce97;
  }
  .badge.overdue {
    background: rgba(247, 68, 68, 0.15);
    color: #f87168;
  }

  .badge-icon {
    width: 12px;
    height: 12px;
    flex-shrink: 0;
  }
  .desc-icon {
    color: #6b7a8d;
    opacity: 0.6;
  }
</style>
