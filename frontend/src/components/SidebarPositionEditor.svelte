<script lang="ts">
  // Shared position editor widget for card sidebars. Provides list and position
  // dropdowns with optional Move button. Handles its own click-outside logic.

  import {
    boardData, boardConfig, sortedListKeys, listOrder,
    isAtLimit, isLocked,
  } from "../stores/board";
  import { getDisplayTitle, clickOutside } from "../lib/utils";
  import Icon from "./Icon.svelte";

  let {
    listKey,
    position,
    currentListKey,
    hasPendingMove,
    onselectlist,
    onselectposition,
    onmove,
  }: {
    listKey: string;
    position: number;
    currentListKey?: string;
    hasPendingMove?: boolean;
    onselectlist: (key: string) => void;
    onselectposition: (idx: number) => void;
    onmove?: () => void;
  } = $props();

  let listDropdownOpen = $state(false);
  let positionDropdownOpen = $state(false);

  let selectedListDisplay = $derived(listKey ? getDisplayTitle(listKey, $boardConfig) : "");
  let targetCards = $derived($boardData[listKey] || []);
  let positionCount = $derived(targetCards.length + (listKey === currentListKey ? 0 : 1));

  let selectedPositionDisplay = $derived.by(() => {
    const n = position + 1;
    if (positionCount <= 1) {
      return `${n}`;
    }
    if (position === 0) {
      return `${n} (top)`;
    }
    if (position === positionCount - 1) {
      return `${n} (bottom)`;
    }
    return `${n}`;
  });

  function selectList(key: string): void {
    onselectlist(key);
    listDropdownOpen = false;
  }

  function selectPosition(idx: number): void {
    onselectposition(idx);
    positionDropdownOpen = false;
  }

</script>

<div class="position-editor">
  <div class="position-field">
    <span class="position-label">List</span>
    <div class="dropdown-wrap" use:clickOutside={() => { listDropdownOpen = false; }}>
      <button class="dropdown-trigger" onclick={() => listDropdownOpen = !listDropdownOpen}>
        <span class="dropdown-trigger-text">{selectedListDisplay}</span>
        <span class="dropdown-chevron" class:open={listDropdownOpen}>
          <Icon name="chevron-down" size={12} />
        </span>
      </button>
      {#if listDropdownOpen}
        <div class="dropdown-menu">
          {#each sortedListKeys($boardData, $listOrder) as key}
            {@const locked = isLocked(key, $boardConfig)}
            {@const full = key !== currentListKey && isAtLimit(key, $boardData, $boardConfig)}
            {@const blocked = locked || full}
            <button class="dropdown-option" class:active={key === listKey} class:disabled={blocked}
              disabled={blocked} onclick={() => selectList(key)}>
              {getDisplayTitle(key, $boardConfig)}
              {#if locked}<span class="dropdown-hint">(locked)</span>{/if}
              {#if full}<span class="dropdown-hint">(full)</span>{/if}
            </button>
          {/each}
        </div>
      {/if}
    </div>
  </div>
  <div class="position-field">
    <span class="position-label">
      Position
      <span class="pos-presets">
        <button type="button" class="pos-preset" class:active={position === 0}
          onclick={() => selectPosition(0)} title="Top position (first)">T</button>
        <button type="button" class="pos-preset" class:active={positionCount > 1 && position === positionCount - 1}
          onclick={() => selectPosition(positionCount - 1)} title="Bottom position (last)">B</button>
      </span>
    </span>
    <div class="dropdown-wrap" use:clickOutside={() => { positionDropdownOpen = false; }}>
      <button class="dropdown-trigger mono" onclick={() => positionDropdownOpen = !positionDropdownOpen}>
        <span class="dropdown-trigger-text">{selectedPositionDisplay}</span>
        <span class="dropdown-chevron" class:open={positionDropdownOpen}>
          <Icon name="chevron-down" size={12} />
        </span>
      </button>
      {#if positionDropdownOpen}
        <div class="dropdown-menu">
          {#each Array.from({ length: positionCount }, (_, i) => i) as idx}
            <button class="dropdown-option mono" class:active={idx === position} onclick={() => selectPosition(idx)}>
              {idx + 1}{idx === 0 ? " (top)" : ""}{idx === positionCount - 1 && positionCount > 1 ? " (bottom)" : ""}
            </button>
          {/each}
        </div>
      {/if}
    </div>
  </div>
  {#if hasPendingMove && onmove}
    <button class="position-move-btn" onclick={onmove}>Move</button>
  {/if}
</div>

<style lang="scss">
  .position-editor {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .position-field {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .position-label {
    display: flex;
    align-items: center;
    font-size: 0.68rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-text-muted);
  }

  .mono {
    font-family: var(--font-mono);
  }

  .position-move-btn {
    all: unset;
    width: 100%;
    text-align: center;
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--color-text-secondary);
    background: var(--overlay-hover-light);
    border: 1px solid var(--color-border);
    border-radius: 4px;
    padding: 4px 6px;
    cursor: pointer;
    box-sizing: border-box;

    &:hover {
      background: var(--overlay-hover-medium);
      color: var(--color-text-primary);
    }
  }
</style>
