<script lang="ts">
  // Editable counter widget with +/- buttons, progress bar, and a settings panel.
  // Supports counting up (e.g. chapters: 0 to 10 by 1) and counting down (e.g. weight: 215 to 200 by 1).

  import type { daedalus } from "../../wailsjs/go/models";
  import Icon from "./Icon.svelte";

  let {
    counter,
    onsave,
  }: {
    counter?: daedalus.Counter;
    onsave?: (counter: daedalus.Counter | null) => void;
  } = $props();

  let settingsOpen = $state(false);
  let editLabel = $state("");
  let editFrom = $state("");
  let editTo = $state("");
  let editStep = $state("");

  // Progress percentage toward the goal. Works for both directions because
  // (current - start) / (max - start) naturally handles negative ranges:
  //   count-up   0->10, current=5:   (5-0)/(10-0)     = 50%
  //   count-down 215->200, current=210: (210-215)/(200-215) = -5/-15 = 33%
  let pct = $derived.by(() => {
    if (!counter) {
      return 0;
    }
    const range = counter.max - counter.start;
    if (range === 0) {
      return 0;
    }
    return Math.max(0, Math.min(100, ((counter.current - counter.start) / range) * 100));
  });

  let countingDown = $derived(counter ? counter.start > counter.max : false);

  let atStart = $derived(counter ? counter.current === counter.start : false);
  let atGoal = $derived(counter ? counter.current === counter.max : false);

  function adjust(direction: number): void {
    if (!counter || !onsave) {
      return;
    }
    const step = counter.step || 1;
    const lo = Math.min(counter.start, counter.max);
    const hi = Math.max(counter.start, counter.max);
    // For count-up, +1 adds step. For count-down, +1 subtracts step (toward the goal).
    const delta = countingDown ? -direction : direction;
    const next = Math.max(lo, Math.min(hi, counter.current + delta * step));
    if (next !== counter.current) {
      onsave({ ...counter, current: next } as daedalus.Counter);
    }
  }

  // Opens settings, populating string fields from current counter values.
  function openSettings(): void {
    if (counter) {
      editLabel = counter.label || "";
      editFrom = String(counter.start || 0);
      editTo = String(counter.max);
      editStep = String(counter.step || 1);
    }
    settingsOpen = true;
  }

  // Parses a string to an integer. Returns fallback for empty/invalid input.
  function toInt(value: string, fallback: number): number {
    const trimmed = value.trim();
    if (trimmed === "" || trimmed === "-") {
      return fallback;
    }
    const n = Number(trimmed);
    return Number.isFinite(n) ? Math.round(n) : fallback;
  }

  // Saves settings from string edit fields.
  function saveSettings(): void {
    if (!counter || !onsave) {
      return;
    }

    const start = toInt(editFrom, counter.start ?? 0);
    const max = toInt(editTo, counter.max ?? 10);
    const step = Math.max(1, toInt(editStep, counter.step ?? 1));

    if (start === max) {
      return;
    }

    // Reset current to start when the range changes, keep it otherwise (e.g. label-only edit).
    const rangeChanged = start !== (counter.start ?? 0) || max !== counter.max;

    onsave({
      ...counter,
      label: editLabel.trim(),
      start,
      max,
      step,
      current: rangeChanged ? start : counter.current,
    } as daedalus.Counter);
    settingsOpen = false;
  }

  // Creates a new default counter.
  function addCounter(): void {
    if (onsave) {
      onsave({ current: 0, max: 10, start: 0, step: 1, label: "" } as daedalus.Counter);
    }
  }

  // Removes the counter from the card.
  function removeCounter(): void {
    if (onsave) {
      onsave(null);
      settingsOpen = false;
    }
  }
</script>

{#if counter}
  <div class="sidebar-section">
    <div class="section-header">
      <h4 class="sidebar-title">{counter.label || "Counter"}</h4>
      <div class="section-header-actions">
        {#if settingsOpen}
          <button class="counter-header-btn save" title="Save settings" onclick={saveSettings}>
            <Icon name="check" size={12} />
          </button>
        {:else}
          <button class="counter-header-btn" title="Counter settings" onclick={openSettings}>
            <Icon name="pencil" size={12} />
          </button>
        {/if}
        <button class="counter-header-btn remove" title="Remove counter" onclick={removeCounter}>
          <Icon name="trash" size={12} />
        </button>
      </div>
    </div>
    <div class="counter-progress-row">
      <button class="btn-icon counter-btn" title="Undo progress" disabled={atStart} onclick={() => adjust(-1)}>-</button>
      <div class="progress-bar sidebar-progress">
        <div class="progress-fill" class:complete={pct >= 100} style="width: {pct}%"></div>
      </div>
      <span class="counter-fraction">{counter.current}/{counter.max}</span>
      <button class="btn-icon counter-btn" title="Make progress" disabled={atGoal} onclick={() => adjust(1)}>+</button>
    </div>
    {#if settingsOpen}
      <div class="counter-settings">
        <input type="text" class="form-input counter-input" bind:value={editLabel} placeholder="Label" onkeydown={e => e.key === 'Enter' && saveSettings()}/>
        <div class="counter-range-row">
          <input type="text" inputmode="numeric" class="form-input counter-input range-input"
            bind:value={editFrom} onkeydown={e => e.key === 'Enter' && saveSettings()}
          />
          <span class="range-text">to</span>
          <input type="text" inputmode="numeric" class="form-input counter-input range-input"
            bind:value={editTo} onkeydown={e => e.key === 'Enter' && saveSettings()}
          />
          <span class="range-text">by</span>
          <input type="text" inputmode="numeric" class="form-input counter-input range-input"
            bind:value={editStep} onkeydown={e => e.key === 'Enter' && saveSettings()}
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
  .counter-btn {
    width: 22px;
    height: 22px;
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    color: var(--color-text-primary);
    font-size: 0.85rem;
    font-weight: 600;
    box-sizing: border-box;

    &:hover:not(:disabled) {
      color: var(--color-text-primary);
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
    margin-top: 2px;
  }

  .counter-fraction {
    font-family: var(--font-mono);
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
    font-family: var(--font-mono);
    font-size: 0.75rem;
    padding: 5px 8px;
  }

  .range-input {
    width: 0;
    flex: 1;
    padding: 4px 6px;
    text-align: center;
  }

  .sidebar-progress {
    flex: 1;
  }
</style>
