<script lang="ts">
  // Modal for managing labels - colors, custom picker, and deletion.

  import { boardData, labelColors, labelsExpanded, addToast, saveWithToast, toggleLabelsExpanded } from "../stores/board";
  import { SaveLabelColors, RemoveLabel, RenameLabel } from "../../wailsjs/go/main/App";
  import { labelColor, autoFocus as autoFocusInput, backdropClose } from "../lib/utils";
  import Icon from "./Icon.svelte";
  import ColorPicker from "./ColorPicker.svelte";

  let { onclose, onreload }: { onclose: () => void; onreload: () => Promise<void> } = $props();

  let activeLabel = $state<string | null>(null);
  let confirmingDelete = $state<string | null>(null);
  let editingLabel = $state<string | null>(null);
  let editingName = $state("");
  let addingNew = $state(false);
  let newLabelName = $state("");

  // Counts how many cards use each label, merges with board-defined labels, then returns sorted.
  let allLabels = $derived.by(() => {
    const counts = new Map<string, number>();
    for (const name of Object.keys($labelColors)) {
      counts.set(name, 0);
    }
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

  // Assigns a custom color to a label and persists to board.yaml.
  function pickColor(label: string, color: string): void {
    labelColors.update(colors => {
      const updated = { ...colors, [label]: color };
      saveWithToast(SaveLabelColors(updated), "save label colors");
      return updated;
    });
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
  }

  // Toggles the swatch picker for a label.
  function togglePicker(label: string): void {
    activeLabel = activeLabel === label ? null : label;
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

  // Handles a color change from the ColorPicker for the active label.
  function handleColorChange(label: string, hex: string): void {
    if (hex) {
      pickColor(label, hex);
    } else {
      resetColor(label);
    }
  }

  // Creates a new label with a default hash-based color.
  function commitNewLabel(): void {
    const name = newLabelName.trim();
    addingNew = false;
    newLabelName = "";
    if (!name) {
      return;
    }
    if ($labelColors[name] || allLabels.some(l => l.name === name)) {
      addToast(`Label "${name}" already exists`);
      return;
    }
    labelColors.update(colors => {
      const updated = { ...colors, [name]: labelColor(name, colors) };
      saveWithToast(SaveLabelColors(updated), "save label colors");
      return updated;
    });
  }

  // Handles keydown in the new label input.
  function handleNewLabelKeydown(e: KeyboardEvent): void {
    if (e.key === "Enter") {
      e.preventDefault();
      commitNewLabel();
    } else if (e.key === "Escape") {
      e.preventDefault();
      addingNew = false;
      newLabelName = "";
    }
  }

</script>

<div class="modal-backdrop centered z-high" role="presentation" use:backdropClose={onclose}>
  <div class="modal-dialog size-md" role="dialog">
    <div class="modal-header">
      <h2 class="modal-title">Label Manager</h2>
      <div class="header-actions">
        <button class="header-btn" onclick={() => { addingNew = true; }} title="Create a new label">+ New</button>
        <button class="header-btn" onclick={toggleLabelsExpanded} title={$labelsExpanded ? "Collapse labels to dots" : "Expand labels to text"}>
          {$labelsExpanded ? "Collapse" : "Expand"}
        </button>
        <button class="modal-close" onclick={onclose} title="Close">
          <Icon name="close" size={16} />
        </button>
      </div>
    </div>
    <div class="editor-body">
      {#if allLabels.length === 0 && !Object.keys($labelColors).length && !addingNew}
        <p class="empty-msg">No labels found. Click "+ New" to create one.</p>
      {:else}
        <table class="manager-table">
          <thead>
            <tr>
              <th class="col-label">Label</th>
              <th class="col-color">Color</th>
              <th class="col-cards">Cards</th>
              <th class="col-actions"></th>
            </tr>
          </thead>
          <tbody>
            {#if addingNew}
              <tr class="label-row">
                <td class="col-label" colspan="4">
                  <input type="text" class="form-input rename-input" placeholder="Label name" bind:value={newLabelName}
                    onblur={commitNewLabel} onkeydown={handleNewLabelKeydown} use:autoFocusInput
                  />
                </td>
              </tr>
            {/if}
            {#each allLabels as { name, count }}
              {@const isCustom = !!$labelColors[name]}
              <tr class="label-row" class:active={activeLabel === name}>
                <td class="col-label">
                  {#if editingLabel === name}
                    <input type="text" class="form-input rename-input" bind:value={editingName} onblur={commitRename}
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
                      <button class="btn-icon reset-btn" onclick={() => resetColor(name)} title="Reset color">
                        <Icon name="refresh" size={10} />
                      </button>
                    {/if}
                    {#if confirmingDelete === name}
                      <button class="btn-icon delete-btn confirming" onclick={() => deleteLabel(name)} title="Click again to confirm">confirm?</button>
                    {:else}
                      <button class="btn-icon delete-btn" onclick={() => { confirmingDelete = name; }} title="Delete label from all cards">
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
                      <ColorPicker color={$labelColors[name] || ""} onchange={(hex) => handleColorChange(name, hex)} />
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
    font-size: 0.82rem;
    font-weight: 500;
    padding: 1px 4px;
    width: 100%;
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
    width: 18px;
    height: 18px;
  }

  .picker-row td {
    padding: 0 0 6px 0;
    border-bottom: 1px solid var(--color-border);
  }

  .picker-panel {
    padding: 6px 0;
  }

</style>
