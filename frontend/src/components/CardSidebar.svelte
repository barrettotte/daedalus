<script lang="ts">
  // Card detail sidebar. Shows position editor, labels, icon, dates, estimate,
  // counter, checklist summary, and timestamps.

  import {
    addToast, boardData,
    selectedCard, boardConfig,
    moveCardInBoard, computeListOrder, isAtLimit, isLocked, syncCardAfterMove,
  } from "../stores/board";
  import { formatDateTime, copyToClipboard } from "../lib/utils";
  import { MoveCard, LoadBoard } from "../../wailsjs/go/main/App";
  import type { daedalus } from "../../wailsjs/go/models";
  import CounterControl from "./CounterControl.svelte";
  import DateSection from "./DateSection.svelte";
  import SidebarLabelEditor from "./SidebarLabelEditor.svelte";
  import SidebarIconEditor from "./SidebarIconEditor.svelte";
  import SidebarEstimateEditor from "./SidebarEstimateEditor.svelte";
  import SidebarChecklistSummary from "./SidebarChecklistSummary.svelte";
  import SidebarPositionEditor from "./SidebarPositionEditor.svelte";
  import Icon from "./Icon.svelte";

  let {
    meta,
    onsavecounter,
    onsavedates,
    onsaveestimate,
    onsaveicon,
    onsavechecklist,
    onsavelabels,
  }: {
    meta: daedalus.CardMetadata;
    onsavecounter?: (counter: daedalus.Counter | null) => void;
    onsavedates?: (
      due: string | null,
      range: { start: string; end: string } | null,
    ) => void;
    onsaveestimate?: (estimate: number | null) => void;
    onsaveicon?: (icon: string) => void;
    onsavechecklist?: (checklist: daedalus.CheckListItem[] | null) => void;
    onsavelabels?: (labels: string[]) => void;
  } = $props();

  // The list key where the currently selected card lives.
  let cardListKey = $derived.by(() => {
    if (!$selectedCard) {
      return "";
    }
    for (const key of Object.keys($boardData)) {
      if ($boardData[key].some(c => c.filePath === $selectedCard!.filePath)) {
        return key;
      }
    }
    return "";
  });

  // Selected list key for the position editor (tracks user selection).
  let selectedListKey = $state("");
  // Selected 0-based position index for the position editor.
  let selectedPosition = $state(0);

  // Reset dropdowns when the selected card changes.
  $effect(() => {
    if ($selectedCard && cardListKey) {
      selectedListKey = cardListKey;
      const cards = $boardData[cardListKey] || [];
      const idx = cards.findIndex(c => c.filePath === $selectedCard!.filePath);
      selectedPosition = idx === -1 ? 0 : idx;
    }
  });

  // Reset position to top when switching to a different list, restore actual position when switching back.
  let prevSelectedListKey = $state("");
  $effect(() => {
    if (selectedListKey !== prevSelectedListKey) {
      if (selectedListKey !== cardListKey) {
        selectedPosition = 0;
      } else if ($selectedCard) {
        const cards = $boardData[cardListKey] || [];
        const idx = cards.findIndex(c => c.filePath === $selectedCard!.filePath);
        selectedPosition = idx === -1 ? 0 : idx;
      }
      prevSelectedListKey = selectedListKey;
    }
  });

  // Whether the current selection differs from the card's actual position.
  let hasPendingMove = $derived.by(() => {
    if (selectedListKey !== cardListKey) {
      return true;
    }
    const cards = $boardData[cardListKey] || [];
    const currentIdx = cards.findIndex(c => c.filePath === $selectedCard!.filePath);
    return selectedPosition !== currentIdx;
  });

  // Moves the card to the selected list and position.
  async function executeMove(): Promise<void> {
    if (!$selectedCard || !hasPendingMove) {
      return;
    }
    if (isLocked(cardListKey, $boardConfig) || isLocked(selectedListKey, $boardConfig)) {
      addToast("List is locked");
      return;
    }
    if (selectedListKey !== cardListKey && isAtLimit(selectedListKey, $boardData, $boardConfig)) {
      addToast("List is at its card limit");
      return;
    }

    const targetCards = $boardData[selectedListKey] || [];

    // When reordering within the same list, account for the card being removed first.
    let cardsForOrder: daedalus.KanbanCard[];
    if (selectedListKey === cardListKey) {
      cardsForOrder = targetCards.filter(c => c.filePath !== $selectedCard!.filePath);
    } else {
      cardsForOrder = targetCards;
    }

    const newListOrder = computeListOrder(cardsForOrder, selectedPosition);
    const originalPath = $selectedCard.filePath;

    moveCardInBoard(originalPath, cardListKey, selectedListKey, selectedPosition, newListOrder);

    try {
      const result = await MoveCard(originalPath, selectedListKey, newListOrder);

      if (result.filePath !== originalPath) {
        syncCardAfterMove(selectedListKey, result.metadata.id, result);
      }
      selectedCard.set(result);
      addToast("Card moved", "success");
    } catch (err) {
      addToast(`Failed to move card: ${err}`);
      const response = await LoadBoard("");
      boardData.set(response.lists);
    }
  }
</script>

<div class="sidebar">
  <div class="sidebar-section card-top-section">
    {#if $selectedCard?.filePath}
      <button class="file-path-btn" title={$selectedCard.filePath} onclick={() => copyToClipboard($selectedCard!.filePath, "File path")}>
        <Icon name="file-text" size={12} />
      </button>
    {/if}
    <button class="card-id-btn" title="Copy ID" onclick={() => copyToClipboard(String(meta.id), "Card ID")}>
      Card #{meta.id}
    </button>

    <SidebarPositionEditor
      listKey={selectedListKey}
      position={selectedPosition}
      currentListKey={cardListKey}
      {hasPendingMove}
      onselectlist={(key) => { selectedListKey = key; }}
      onselectposition={(idx) => { selectedPosition = idx; }}
      onmove={executeMove}
    />
  </div>

  <SidebarLabelEditor labels={meta.labels || []} onchange={(l) => onsavelabels?.(l)} />

  <SidebarIconEditor icon={meta.icon || ""} onchange={(i) => onsaveicon?.(i)} />

  <DateSection due={meta.due} range={meta.range} onsave={onsavedates} />

  <SidebarEstimateEditor estimate={meta.estimate ?? null} onchange={(e) => onsaveestimate?.(e)} />

  <CounterControl counter={meta.counter} onsave={onsavecounter} />

  <SidebarChecklistSummary
    checklist={meta.checklist ?? null}
    title={meta.checklist_title || undefined}
    onchange={(c) => onsavechecklist?.(c)}
  />

  <div class="sidebar-section timestamps">
    {#if meta.created}
      <div class="timestamp-row">
        <span class="timestamp-label">Created</span>
        <span class="timestamp-value">{formatDateTime(meta.created)}</span>
      </div>
    {/if}
    <div class="timestamp-row">
      <span class="timestamp-label">Updated</span>
      <span class="timestamp-value">
        {formatDateTime(meta.updated && meta.updated !== meta.created ? meta.updated : meta.created)}
      </span>
    </div>
  </div>

</div>

<style lang="scss">

  .card-top-section {
    position: relative;
  }

  .card-id-btn {
    all: unset;
    align-self: center;
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--color-text-muted);
    cursor: pointer;
    padding: 2px 6px;
    border-radius: 4px;

    &:hover {
      background: var(--overlay-hover);
      color: var(--color-text-primary);
    }
  }

  .file-path-btn {
    all: unset;
    position: absolute;
    top: 6px;
    right: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    cursor: pointer;
    color: var(--color-text-muted);
    border-radius: 3px;

    &:hover {
      background: var(--overlay-hover);
      color: var(--color-text-primary);
    }
  }

  .timestamps {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .timestamp-row {
    display: flex;
    align-items: baseline;
    gap: 6px;
  }

  .timestamp-label {
    font-size: 0.6rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-text-muted);
    flex-shrink: 0;
  }

  .timestamp-value {
    font-family: var(--font-mono);
    font-size: 0.65rem;
    color: var(--color-text-secondary);
    white-space: nowrap;
    margin-left: auto;
    text-align: right;
  }
</style>
