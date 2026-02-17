<script lang="ts">
  // Shared position editor widget for card sidebars. Provides list and position
  // dropdowns with optional Move button. Handles its own click-outside logic.

  import {
    boardData, boardConfig, sortedListKeys, listOrder,
    isAtLimit, isLocked,
  } from "../stores/board";
  import { getDisplayTitle, clickOutside } from "../lib/utils";

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
    <div class="pos-dropdown" use:clickOutside={() => { listDropdownOpen = false; }}>
      <button class="pos-trigger" onclick={() => listDropdownOpen = !listDropdownOpen}>
        <span class="pos-trigger-text">{selectedListDisplay}</span>
        <svg class="pos-chevron" class:open={listDropdownOpen} viewBox="0 0 16 16" width="12" height="12">
          <path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </button>
      {#if listDropdownOpen}
        <div class="pos-menu">
          {#each sortedListKeys($boardData, $listOrder) as key}
            {@const locked = isLocked(key, $boardConfig)}
            {@const full = key !== currentListKey && isAtLimit(key, $boardData, $boardConfig)}
            {@const blocked = locked || full}
            <button class="pos-option" class:active={key === listKey} class:disabled={blocked}
              disabled={blocked} onclick={() => selectList(key)}>
              {getDisplayTitle(key, $boardConfig)}
              {#if locked}<span class="pos-hint">(locked)</span>{/if}
              {#if full}<span class="pos-hint">(full)</span>{/if}
            </button>
          {/each}
        </div>
      {/if}
    </div>
  </div>
  <div class="position-field">
    <span class="position-label">Position</span>
    <div class="pos-dropdown" use:clickOutside={() => { positionDropdownOpen = false; }}>
      <button class="pos-trigger mono" onclick={() => positionDropdownOpen = !positionDropdownOpen}>
        <span class="pos-trigger-text">{selectedPositionDisplay}</span>
        <svg class="pos-chevron" class:open={positionDropdownOpen} viewBox="0 0 16 16" width="12" height="12">
          <path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </button>
      {#if positionDropdownOpen}
        <div class="pos-menu">
          {#each Array.from({ length: positionCount }, (_, i) => i) as idx}
            <button class="pos-option mono" class:active={idx === position} onclick={() => selectPosition(idx)}>
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
    font-size: 0.68rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-text-muted);
  }

  .pos-dropdown {
    position: relative;
  }

  .pos-trigger {
    all: unset;
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 4px;
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    color: var(--color-text-primary);
    font-size: 0.8rem;
    padding: 4px 6px;
    border-radius: 4px;
    cursor: pointer;
    box-sizing: border-box;

    &:hover {
      border-color: var(--color-text-tertiary);
    }
  }

  .pos-trigger-text {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }

  .pos-chevron {
    color: var(--color-text-tertiary);
    transition: transform 0.15s;
    flex-shrink: 0;

    &.open {
      transform: rotate(180deg);
    }
  }

  .pos-menu {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    right: 0;
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: 4px;
    padding: 4px 0;
    z-index: 10;
    max-height: 200px;
    overflow-y: auto;
  }

  .mono {
    font-family: var(--font-mono);
  }

  .pos-option {
    all: unset;
    display: flex;
    align-items: center;
    width: 100%;
    padding: 5px 8px;
    font-size: 0.8rem;
    color: var(--color-text-primary);
    cursor: pointer;
    box-sizing: border-box;

    &:hover:not(:disabled) {
      background: var(--overlay-hover);
    }

    &.active {
      color: var(--color-accent);
    }

    &.disabled {
      color: var(--color-text-muted);
      cursor: not-allowed;
    }
  }

  .pos-hint {
    font-size: 0.7rem;
    color: var(--color-text-muted);
    margin-left: auto;
  }

  .position-move-btn {
    all: unset;
    text-align: center;
    background: var(--color-accent);
    color: var(--color-text-inverse);
    font-size: 0.78rem;
    font-weight: 600;
    padding: 4px 8px;
    border-radius: 4px;
    cursor: pointer;

    &:hover {
      opacity: 0.9;
    }
  }
</style>
