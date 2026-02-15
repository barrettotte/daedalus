<script lang="ts">
  import type { daedalus } from "../../wailsjs/go/models";
  import Icon from "./Icon.svelte";

  let {
    counter,
    onsave,
  }: {
    counter?: daedalus.Counter;
    onsave?: (counter: daedalus.Counter | null) => void;
  } = $props();

  // Counter settings panel state
  let counterSettingsOpen = $state(false);
  let editLabel = $state("");
  let editStart = $state(0);
  let editMax = $state(0);
  let editStep = $state(1);

  // Counter completion percentage.
  let counterPct = $derived.by(() => {
    if (!counter) {
      return 0;
    }
    const { current, max, start } = counter;
    const range = max - start;

    if (range === 0) {
      return 0;
    }
    const pct = ((current - start) / range) * 100;
    return Math.max(0, Math.min(100, pct));

  });

  // Whether the counter counts down (start > max).
  let countingDown = $derived(counter ? (counter.start || 0) > counter.max : false);

  // Whether the counter has reached its start or goal bound.
  let atStart = $derived.by(() => {
    if (!counter) {
      return false;
    }
    return counter.current === (counter.start || 0);
  });

  let atGoal = $derived.by(() => {
    if (!counter) {
      return false;
    }
    return counter.current === counter.max;
  });

  // Increments or decrements the counter's current value by step and saves.
  function adjustCounter(delta: number): void {
    if (!counter || !onsave) {
      return;
    }
    const step = counter.step || 1;
    const lo = Math.min(counter.start || 0, counter.max);
    const hi = Math.max(counter.start || 0, counter.max);
    const next = Math.max(lo, Math.min(hi, counter.current + delta * step));

    if (next === counter.current) {
      return;
    }
    const updated = { ...counter, current: next };
    onsave(updated as daedalus.Counter);
  }

  // Opens the counter settings panel, populating edit fields from current values.
  function openCounterSettings(): void {
    if (counter) {
      editLabel = counter.label || "";
      editStart = counter.start || 0;
      editMax = counter.max;
      editStep = counter.step || 1;
    }
    counterSettingsOpen = true;
  }

  // Saves counter settings from the edit fields.
  function saveCounterSettings(): void {
    if (!counter || !onsave) {
      return;
    }
    const lo = Math.min(editStart, editMax);
    const hi = Math.max(editStart, editMax);
    const cur = counter.current;
    const needsReset = cur < lo || cur > hi;

    const updated = {
      ...counter,
      label: editLabel,
      start: editStart,
      max: editMax,
      step: Math.max(1, editStep || 1),
      current: needsReset ? editStart : cur,
    };

    onsave(updated as daedalus.Counter);
    counterSettingsOpen = false;
  }

  // Adds a new default counter to the card.
  function addCounter(): void {
    if (!onsave) {
      return;
    }
    onsave({ current: 0, max: 10, start: 0, label: "" } as daedalus.Counter);
  }

  // Removes the counter from the card.
  function removeCounter(): void {
    if (!onsave) {
      return;
    }
    onsave(null);
    counterSettingsOpen = false;
  }
</script>

{#if counter}
  <div class="sidebar-section">
    <div class="counter-header">
      <h4 class="sidebar-title">{counter.label || "Counter"}</h4>
      <div class="counter-header-right">
        {#if counterSettingsOpen}
          <button class="counter-header-btn save" title="Save settings" onclick={saveCounterSettings}>
            <svg viewBox="0 0 24 24" width="12" height="12">
              <polyline points="20 6 9 17 4 12" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
        {:else}
          <button class="counter-header-btn" title="Counter settings" onclick={openCounterSettings}>
            <svg viewBox="0 0 24 24" width="12" height="12">
              <circle cx="12" cy="12" r="3" fill="none" stroke="currentColor" stroke-width="2"/>
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65
                1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9
                19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0
                4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65
                0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65
                0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0
                1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0
                1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"
                fill="none" stroke="currentColor" stroke-width="2"
              />
            </svg>
          </button>
        {/if}
        <button class="counter-header-btn remove" title="Remove counter" onclick={removeCounter}>
          <Icon name="trash" size={12} />
        </button>
      </div>
    </div>
    <div class="counter-progress-row">
      <button class="counter-btn" title="Decrease" disabled={atStart} onclick={() => adjustCounter(countingDown ? 1 : -1)}>-</button>
      <div class="progress-bar sidebar-progress">
        <div class="progress-fill" class:complete={counterPct >= 100} style="width: {counterPct}%"></div>
      </div>
      <span class="counter-fraction">{counter.current}/{counter.max}</span>
      <button class="counter-btn" title="Increase" disabled={atGoal} onclick={() => adjustCounter(countingDown ? -1 : 1)}>+</button>
    </div>
    {#if counterSettingsOpen}
      <div class="counter-settings">
        <input type="text" class="counter-input" bind:value={editLabel} placeholder="Label" onkeydown={e => e.key === 'Enter' && saveCounterSettings()}/>
        <div class="counter-range-row">
          <input type="number" class="counter-input range-input" bind:value={editStart}
            onblur={() => {
              editStart = Math.max(0, Number(editStart) || 0);
              if (editStart === editMax) { 
                editMax = editStart + 1; 
              }
            }}
            onkeydown={e => e.key === 'Enter' && saveCounterSettings()}
          />
          <span class="range-text">to</span>
          <input type="number" class="counter-input range-input" bind:value={editMax}
            onblur={() => {
              editMax = Math.max(0, Number(editMax) || 0);
              if (editMax === editStart) {
                editMax = editStart + 1;
              }
            }}
            onkeydown={e => e.key === 'Enter' && saveCounterSettings()}
          />
          <span class="range-text">by</span>
          <input type="number" class="counter-input range-input" bind:value={editStep} min="1"
            onblur={() => editStep = Math.max(1, editStep || 1)}
            onkeydown={e => e.key === 'Enter' && saveCounterSettings()}
          />
        </div>
      </div>
    {/if}
  </div>
{:else}
  <div class="sidebar-section">
    <button class="add-counter-btn" title="Add a progress counter" onclick={addCounter}>+ Add counter</button>
  </div>
{/if}

<style lang="scss">
  .counter-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 4px;

    .sidebar-title {
      margin: 0;
    }
  }

  .counter-header-right {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .counter-btn {
    all: unset;
    width: 22px;
    height: 22px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    border-radius: 4px;
    color: var(--color-text-primary);
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;
    box-sizing: border-box;

    &:hover:not(:disabled) {
      background: var(--overlay-hover);
      border-color: var(--color-text-tertiary);
    }

    &:disabled {
      opacity: 0.3;
      cursor: not-allowed;
    }
  }

  .counter-progress-row {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .counter-fraction {
    font-size: 0.7rem;
    font-weight: 600;
    color: var(--color-text-primary);
    flex-shrink: 0;
  }

  .counter-settings {
    margin-top: 10px;
    padding-top: 10px;
    border-top: 1px solid var(--color-border);
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .counter-input {
    width: 100%;
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    color: var(--color-text-primary);
    font-size: 0.75rem;
    padding: 5px 8px;
    border-radius: 4px;
    outline: none;
    box-sizing: border-box;
    &:focus {
      border-color: var(--color-accent);
    }
  }

  .range-input {
    width: 0;
    flex: 1;
    padding: 4px 6px;
    text-align: center;
  }

  .progress-bar {
    height: 6px;
    background: var(--color-border);
    border-radius: 3px;
    overflow: hidden;
    margin-bottom: 8px;
    max-width: 100%;
    box-sizing: border-box;
  }

  .sidebar-progress {
    margin-bottom: 0;
    flex: 1;
  }

</style>
