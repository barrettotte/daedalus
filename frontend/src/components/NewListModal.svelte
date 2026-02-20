<script lang="ts">
  // Modal dialog for creating a new list on the kanban board.

  import { CreateList, SaveListConfig, SaveListOrder } from "../../wailsjs/go/main/App";
  import { listOrder, addToast } from "../stores/board";
  import { autoFocus, backdropClose, formatListName } from "../lib/utils";
  import Icon from "./Icon.svelte";
  import ColorPicker from "./ColorPicker.svelte";

  let { onclose, onreload }: { onclose: () => void; onreload: () => Promise<void> } = $props();

  let name = $state("");
  let title = $state("");
  let limit = $state(0);
  let position = $state($listOrder.length + 1);
  let color = $state("");
  let submitting = $state(false);

  let formatPreview = $derived(name.trim() ? formatListName(name.trim()) : "Auto-generated from name");

  let nameError = $derived.by(() => {
    const trimmed = name.trim();
    if (!trimmed) {
      return "";
    }
    if (/[/\\]/.test(trimmed) || trimmed.includes("..")) {
      return "Name cannot contain slashes or '..'";
    }
    if (trimmed.startsWith(".")) {
      return "Name cannot start with '.'";
    }
    if (trimmed === "_assets") {
      return "Name '_assets' is reserved";
    }
    return "";
  });

  async function submit(): Promise<void> {
    const trimmed = name.trim();
    if (!trimmed) {
      return;
    }

    submitting = true;
    try {
      await CreateList(trimmed);

      if (title.trim() || limit || color) {
        await SaveListConfig(trimmed, title.trim(), limit, color, "");
      }
      await onreload();

      const currentOrder = $listOrder;
      const totalLists = currentOrder.length;
      const clampedPos = Math.max(1, Math.min(position, totalLists));

      if (clampedPos !== totalLists) {
        const newOrder = [...currentOrder];
        const idx = newOrder.indexOf(trimmed);
        if (idx !== -1) {
          newOrder.splice(idx, 1);
          newOrder.splice(clampedPos - 1, 0, trimmed);
          listOrder.set(newOrder);
          await SaveListOrder(newOrder);
        }
      }

      addToast(`List "${trimmed}" created`, "success");
      onclose();

    } catch (e) {
      addToast(`Failed to create list: ${e}`);
    } finally {
      submitting = false;
    }
  }

  function handleColorChange(hex: string): void {
    color = hex;
  }
</script>

<svelte:window onkeydown={(e) => { if (e.key === 'Escape') { onclose(); } }} />

<div class="modal-backdrop centered" role="presentation" use:backdropClose={onclose}>
  <div class="modal-dialog size-md" role="dialog">
    <div class="modal-header">
      <h2 class="modal-title">New List</h2>
      <button class="modal-close" onclick={onclose} title="Close">
        <Icon name="close" size={16} />
      </button>
    </div>
    <div class="modal-body-form">
      <div class="form-row">
        <label class="form-label" for="new-list-name">Name <span class="required">*</span></label>
        <input id="new-list-name" class="form-input name-input" type="text" placeholder="e.g. backlog, in-progress, done"
          bind:value={name}
          onkeydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); submit(); } }}
          use:autoFocus
        />
        <span class="form-hint">Directory name. Use lowercase with dashes.</span>
        {#if nameError}
          <div class="form-error">{nameError}</div>
        {/if}
      </div>

      <div class="form-row">
        <label class="form-label" for="new-list-title">Display title</label>
        <input id="new-list-title" class="form-input title-input" type="text" placeholder={formatPreview}
          bind:value={title}
          onkeydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); submit(); } }}
        />
      </div>

      <div class="form-row-inline">
        <div class="form-field">
          <label class="form-label" for="new-list-limit">Card limit</label>
          <input id="new-list-limit" class="form-input limit-input" type="number" min="0" bind:value={limit} />
        </div>
        <div class="form-field">
          <label class="form-label pos-label" for="new-list-position">
            Position
            <span class="pos-presets">
              <button type="button" class="pos-preset" class:active={position === 1} title="First position (leftmost)"
                onclick={() => position = 1}>L</button>
              <button type="button" class="pos-preset" class:active={position === $listOrder.length + 1}
                onclick={() => position = $listOrder.length + 1} title="Last position (rightmost)">R</button>
            </span>
          </label>
          <input id="new-list-position" class="form-input position-input" type="number" min="1" max={$listOrder.length + 1} bind:value={position}/>
        </div>
      </div>

      <div class="form-row">
        <span class="form-label">Color</span>
        <ColorPicker bind:color onchange={handleColorChange} />
      </div>

      <div class="form-actions">
        <button class="cancel-btn" onclick={onclose}>Cancel</button>
        <button class="create-btn" onclick={submit} disabled={!!nameError || submitting || !name.trim()}>
          {submitting ? "Creating..." : "Create"}
        </button>
      </div>

    </div>
  </div>
</div>

<style lang="scss">
  .modal-body-form {
    padding: 12px 20px 20px 20px;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .form-row {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .form-row-inline {
    display: flex;
    gap: 12px;
  }

  .form-field {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .form-label {
    font-size: 0.68rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-text-muted);
  }

  .required {
    color: var(--color-error);
  }

  .form-hint {
    font-size: 0.68rem;
    color: var(--color-text-muted);
  }

  .name-input {
    font-size: 0.85rem;
    padding: 6px 10px;
  }

  .title-input {
    font-size: 0.85rem;
    padding: 6px 10px;
  }

  .limit-input {
    font-family: var(--font-mono);
    font-size: 0.8rem;
    padding: 6px 10px;
    text-align: center;
  }

  .position-input {
    font-family: var(--font-mono);
    font-size: 0.8rem;
    padding: 6px 10px;
    text-align: center;
  }

  .pos-label {
    display: flex;
    align-items: center;
  }


  .form-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 4px;
  }

  .create-btn {
    background: var(--color-accent);
    color: var(--color-text-inverse);
    border: none;
    padding: 8px 20px;
    border-radius: 4px;
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;

    &:hover:not(:disabled) {
      background: var(--color-accent-hover);
    }

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  }

  .form-error {
    color: var(--danger);
    font-size: 0.8rem;
    margin-top: 0.25rem;
  }

</style>
