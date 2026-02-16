<script lang="ts">
  // Modal for managing labels - colors, custom picker, and deletion.

  import { boardData, labelColors, labelsExpanded, addToast, saveWithToast } from "../stores/board";
  import { SaveLabelColors, RemoveLabel, RenameLabel, SaveLabelsExpanded } from "../../wailsjs/go/main/App";
  import { labelColor, autoFocus as autoFocusInput, backdropClose } from "../lib/utils";
  import Icon from "./Icon.svelte";

  let { onclose, onreload }: { onclose: () => void; onreload: () => Promise<void> } = $props();

  let activeLabel = $state<string | null>(null);
  let customPickerOpen = $state(false);
  let confirmingDelete = $state<string | null>(null);
  let editingLabel = $state<string | null>(null);
  let editingName = $state("");
  let hexInput = $state("");

  const PALETTE = [
    "#dc2626", "#ea580c", "#ca8a04", "#16a34a", "#0d9488",
    "#2563eb", "#7c3aed", "#c026d3", "#64748b", "#78716c",
  ];

  // Counts how many cards use each label, then returns sorted label+count pairs.
  let allLabels = $derived.by(() => {
    const counts = new Map<string, number>();
    for (const cards of Object.values($boardData)) {
      for (const card of cards) {
        if (card.metadata.labels) {
          for (const label of card.metadata.labels) {
            counts.set(label, (counts.get(label) || 0) + 1);
          }
        }
      }
    }
    return [...counts.entries()]
      .sort((a, b) => a[0].localeCompare(b[0]))
      .map(([name, count]) => ({ name, count }));
  });

  // Converts an HSL hue (0-360) to a hex color at fixed saturation/lightness.
  function hueToHex(hue: number): string {
    const s = 0.55;
    const l = 0.45;
    const a = s * Math.min(l, 1 - l);

    const f = (n: number) => {
      const k = (n + hue / 30) % 12;
      const c = l - a * Math.max(Math.min(k - 3, 9 - k, 1), -1);
      return Math.round(255 * c).toString(16).padStart(2, "0");
    };

    return `#${f(0)}${f(8)}${f(4)}`;
  }

  // Assigns a custom color to a label and persists to board.yaml.
  function pickColor(label: string, color: string): void {
    labelColors.update(colors => {
      const updated = { ...colors, [label]: color };
      saveWithToast(SaveLabelColors(updated), "save label colors");
      return updated;
    });
    hexInput = color;
  }

  // Removes a custom color override, reverting to the hash default.
  function resetColor(label: string): void {
    labelColors.update(colors => {
      const updated = { ...colors };
      delete updated[label];
      saveWithToast(SaveLabelColors(updated), "save label colors");
      return updated;
    });
    activeLabel = null;
    customPickerOpen = false;
  }

  // Toggles the swatch picker for a label.
  function togglePicker(label: string): void {
    if (activeLabel === label) {
      activeLabel = null;
      customPickerOpen = false;
    } else {
      activeLabel = label;
      customPickerOpen = false;
      hexInput = $labelColors[label] || "";
    }
    confirmingDelete = null;
  }

  // Deletes a label from all cards and removes its custom color.
  async function deleteLabel(label: string): Promise<void> {
    try {
      await RemoveLabel(label);

      labelColors.update(colors => {
        const updated = { ...colors };
        delete updated[label];
        return updated;
      });

      activeLabel = null;
      customPickerOpen = false;
      confirmingDelete = null;
      await onreload();

    } catch (e) {
      addToast(`Failed to delete label: ${e}`);
    }
  }

  // Enters inline edit mode for a label name.
  function startEditing(label: string): void {
    editingLabel = label;
    editingName = label;
  }

  // Saves the renamed label if it changed, then reloads the board.
  async function commitRename(): Promise<void> {
    const oldName = editingLabel;
    const newName = editingName.trim();
    editingLabel = null;

    if (!oldName || !newName || oldName === newName) {
      return;
    }

    try {
      await RenameLabel(oldName, newName);
      labelColors.update(colors => {
        if (colors[oldName]) {
          const updated = { ...colors, [newName]: colors[oldName] };
          delete updated[oldName];
          return updated;
        }
        return colors;
      });

      if (activeLabel === oldName) {
        activeLabel = newName;
      }
      await onreload();

    } catch (e) {
      addToast(`Failed to rename label: ${e}`);
    }
  }

  // Handles keydown in the rename input -- commit on Enter, cancel on Escape.
  function handleRenameKeydown(e: KeyboardEvent): void {
    if (e.key === "Enter") {
      e.preventDefault();
      commitRename();
    } else if (e.key === "Escape") {
      e.preventDefault();
      editingLabel = null;
    }
  }

  // Opens the custom color picker panel for the active label.
  function openCustomPicker(): void {
    if (activeLabel) {
      hexInput = $labelColors[activeLabel] || "";
    }
    customPickerOpen = !customPickerOpen;
  }

  // Picks a color from the hue bar based on click x-position.
  function handleHueBarClick(e: MouseEvent): void {
    if (!activeLabel) {
      return;
    }
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const hue = Math.round(((e.clientX - rect.left) / rect.width) * 360);
    const color = hueToHex(Math.max(0, Math.min(360, hue)));
    pickColor(activeLabel, color);
  }

  // Applies the hex input value if it's a valid hex color.
  function applyHexInput(): void {
    if (!activeLabel) {
      return;
    }

    const trimmed = hexInput.trim();
    if (/^#[0-9a-fA-F]{6}$/.test(trimmed)) {
      pickColor(activeLabel, trimmed);
    }
  }

  // Handles keydown in the hex input -- apply on Enter.
  function handleHexKeydown(e: KeyboardEvent): void {
    if (e.key === "Enter") {
      e.preventDefault();
      applyHexInput();
    }
  }

  // Toggles labels between expanded text and collapsed dots on all cards.
  function toggleLabels(): void {
    labelsExpanded.update(v => {
      const next = !v;
      saveWithToast(SaveLabelsExpanded(next), "save label state");
      return next;
    });
  }
</script>

<div class="modal-backdrop centered z-high" role="presentation" use:backdropClose={onclose}>
  <div class="modal-dialog size-md" role="dialog">
    <div class="modal-header">
      <h2 class="modal-title">Label Manager</h2>
      <div class="header-actions">
        <button class="collapse-toggle" onclick={toggleLabels} title={$labelsExpanded ? "Collapse labels to dots" : "Expand labels to text"}>
          {$labelsExpanded ? "Collapse" : "Expand"}
        </button>
        <button class="modal-close" onclick={onclose} title="Close">
          <Icon name="close" size={16} />
        </button>
      </div>
    </div>
    <div class="editor-body">
      {#if allLabels.length === 0}
        <p class="empty-msg">No labels found on any cards.</p>
      {:else}
        <table class="label-table">
          <thead>
            <tr>
              <th class="col-label">Label</th>
              <th class="col-color">Color</th>
              <th class="col-cards">Cards</th>
              <th class="col-actions"></th>
            </tr>
          </thead>
          <tbody>
            {#each allLabels as { name, count }}
              {@const isCustom = !!$labelColors[name]}
              <tr class="label-row" class:active={activeLabel === name}>
                <td class="col-label">
                  {#if editingLabel === name}
                    <input type="text" class="rename-input" bind:value={editingName} onblur={commitRename}
                      onkeydown={handleRenameKeydown} use:autoFocusInput
                    />
                  {:else}
                    <button class="label-name-btn" onclick={() => startEditing(name)}>{name}</button>
                  {/if}
                </td>
                <td class="col-color">
                  <button class="color-swatch-btn" style="background: {labelColor(name, $labelColors)}" onclick={() => togglePicker(name)} title="Edit color"></button>
                </td>
                <td class="col-cards">
                  <span class="card-count">{count}</span>
                </td>
                <td class="col-actions">
                  <div class="actions-inner">
                    {#if isCustom}
                      <button class="reset-btn" onclick={() => resetColor(name)} title="Reset color">
                        <Icon name="refresh" size={10} />
                      </button>
                    {/if}
                    {#if confirmingDelete === name}
                      <button class="delete-btn confirming" onclick={() => deleteLabel(name)} title="Click again to confirm">confirm?</button>
                    {:else}
                      <button class="delete-btn" onclick={() => { confirmingDelete = name; }} title="Delete label from all cards">
                        <Icon name="trash" size={12} />
                      </button>
                    {/if}
                  </div>
                </td>
              </tr>
              {#if activeLabel === name}
                <tr class="picker-row">
                  <td colspan="4">
                    <div class="picker-panel">
                      <div class="swatch-row">
                        {#each PALETTE as color}
                          <button class="swatch" class:selected={$labelColors[name] === color}
                            style="background: {color}" title={color} onclick={() => pickColor(name, color)}
                          ></button>
                        {/each}
                        <button class="custom-toggle" class:active={customPickerOpen} onclick={openCustomPicker} title="Custom color">
                          <span class="custom-toggle-rainbow"></span>
                          <span class="custom-toggle-plus">+</span>
                        </button>
                      </div>
                      {#if customPickerOpen}
                        <div class="custom-picker">
                          <button type="button" class="hue-bar" title="Pick hue" onclick={handleHueBarClick}></button>
                          <div class="hex-row">
                            <div class="hex-preview" style="background: {$labelColors[name] || labelColor(name)}"></div>
                            <input type="text" class="hex-input" placeholder="#000000" bind:value={hexInput}
                              onblur={applyHexInput} onkeydown={handleHexKeydown}
                            />
                          </div>
                        </div>
                      {/if}
                    </div>
                  </td>
                </tr>
              {/if}
            {/each}
          </tbody>
        </table>
      {/if}
    </div>
  </div>
</div>

<style lang="scss">
  .editor-body {
    padding: 12px 20px 20px 20px;
    max-height: 60vh;
    overflow-y: auto;
  }

  .empty-msg {
    font-size: 0.85rem;
    color: var(--color-text-muted);
    margin: 0;
  }

  .label-table {
    width: 100%;
    border-collapse: collapse;
    border-spacing: 0;
  }

  .label-table th {
    font-size: 0.68rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-text-muted);
    text-align: left;
    padding: 0 16px 8px 0;
    border-bottom: 1px solid var(--color-border);
    white-space: nowrap;
  }

  .col-label {
    width: 100%;
  }

  .col-color {
    white-space: nowrap;
    padding-right: 16px !important;
  }

  .col-cards {
    text-align: left !important;
  }

  .col-actions {
    text-align: right !important;
  }

  .actions-inner {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 2px;
  }

  .label-row td {
    padding: 6px 16px 6px 0;
    vertical-align: middle;
    border-bottom: 1px solid var(--color-border);
  }

  .label-row.active td {
    border-bottom-color: transparent;
  }

  .label-name-btn {
    all: unset;
    font-size: 0.82rem;
    color: var(--color-text-primary);
    font-weight: 500;
    cursor: pointer;
    border-radius: 3px;
    padding: 1px 2px;

    &:hover {
      background: var(--overlay-hover-light);
    }
  }

  .rename-input {
    all: unset;
    font-size: 0.82rem;
    font-weight: 500;
    color: var(--color-text-primary);
    background: var(--color-bg-base);
    border: 1px solid var(--color-accent);
    border-radius: 3px;
    padding: 1px 4px;
    width: 100%;
    box-sizing: border-box;
  }

  .color-swatch-btn {
    all: unset;
    width: 22px;
    height: 22px;
    border-radius: 4px;
    cursor: pointer;
    transition: opacity 0.15s;
    flex-shrink: 0;

    &:hover {
      opacity: 0.8;
    }
  }

  .card-count {
    font-size: 0.78rem;
    color: var(--color-text-muted);
  }

  .reset-btn {
    all: unset;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    color: var(--color-text-muted);
    cursor: pointer;
    border-radius: 3px;
    flex-shrink: 0;

    &:hover {
      color: var(--color-text-primary);
      background: var(--overlay-hover-medium);
    }
  }

  .delete-btn {
    all: unset;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    color: var(--color-text-muted);
    cursor: pointer;
    border-radius: 3px;

    &:hover {
      color: var(--color-error);
      background: var(--overlay-hover-medium);
    }

    &.confirming {
      width: auto;
      font-size: 0.68rem;
      font-weight: 600;
      color: var(--color-error);
      padding: 2px 6px;
    }
  }

  .picker-row td {
    padding: 0 0 6px 0;
    border-bottom: 1px solid var(--color-border);
  }

  .picker-panel {
    padding: 6px 0;
  }

  .swatch-row {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
    align-items: center;
  }

  .swatch {
    all: unset;
    width: 20px;
    height: 20px;
    border-radius: 50%;
    cursor: pointer;
    border: 2px solid transparent;
    box-sizing: border-box;
    transition: border-color 0.15s, transform 0.15s;

    &:hover {
      transform: scale(1.15);
    }

    &.selected {
      border-color: var(--color-text-primary);
    }
  }

  .custom-toggle {
    all: unset;
    position: relative;
    width: 20px;
    height: 20px;
    border-radius: 50%;
    cursor: pointer;
    border: 2px solid transparent;
    box-sizing: border-box;
    transition: border-color 0.15s, transform 0.15s;

    &:hover {
      transform: scale(1.15);
    }

    &.active {
      border-color: var(--color-text-primary);
    }
  }

  .custom-toggle-rainbow {
    position: absolute;
    inset: 0;
    border-radius: 50%;
    background: conic-gradient(
      #dc2626, #ea580c, #ca8a04, #16a34a, #0d9488,
      #2563eb, #7c3aed, #c026d3, #dc2626
    );
  }

  .custom-toggle-plus {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.7rem;
    font-weight: 700;
    color: #fff;
    text-shadow: 0 0 3px rgba(0, 0, 0, 0.7);
  }

  .custom-picker {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: 8px;
  }

  .hue-bar {
    height: 16px;
    border-radius: 4px;
    cursor: crosshair;
    background: linear-gradient(
      to right,
      hsl(0, 55%, 45%),
      hsl(30, 55%, 45%),
      hsl(60, 55%, 45%),
      hsl(90, 55%, 45%),
      hsl(120, 55%, 45%),
      hsl(150, 55%, 45%),
      hsl(180, 55%, 45%),
      hsl(210, 55%, 45%),
      hsl(240, 55%, 45%),
      hsl(270, 55%, 45%),
      hsl(300, 55%, 45%),
      hsl(330, 55%, 45%),
      hsl(360, 55%, 45%)
    );
    border: 1px solid var(--color-border-medium);

    &:hover {
      opacity: 0.9;
    }
  }

  .hex-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .hex-preview {
    width: 24px;
    height: 24px;
    border-radius: 4px;
    border: 1px solid var(--color-border-medium);
    flex-shrink: 0;
  }

  .hex-input {
    flex: 1;
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    color: var(--color-text-primary);
    font-family: var(--font-mono);
    font-size: 0.78rem;
    padding: 4px 8px;
    border-radius: 4px;
    outline: none;

    &:focus {
      border-color: var(--color-accent);
    }

    &::placeholder {
      color: var(--color-text-muted);
    }
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .collapse-toggle {
    all: unset;
    font-size: 0.75rem;
    color: var(--color-text-muted);
    cursor: pointer;
    padding: 2px 8px;
    border-radius: 4px;

    &:hover {
      color: var(--color-text-primary);
      background: var(--overlay-hover);
    }
  }
</style>
