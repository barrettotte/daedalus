<script lang="ts">
  // Modal for managing card templates - create, edit, delete reusable card presets.

  import { templates, labelColors, addToast } from "../stores/board";
  import { SaveTemplates } from "../../wailsjs/go/main/App";
  import { backdropClose, labelColor, isFileIcon } from "../lib/utils";
  import { getIconNames } from "../lib/icons";
  import type { daedalus } from "../../wailsjs/go/models";
  import Icon from "./Icon.svelte";
  import CardIcon from "./CardIcon.svelte";

  let { onclose }: { onclose: () => void } = $props();

  // Local working copy of templates so we can edit without immediately persisting.
  let localTemplates = $state<daedalus.CardTemplate[]>([]);
  let editingIndex = $state<number | null>(null);
  let confirmingDelete = $state<number | null>(null);
  let dirty = $state(false);
  let iconFileNames: string[] = $state([]);
  let emojiValue = $state("");

  // Initialize local copy from store.
  $effect(() => {
    localTemplates = $templates.map(t => structuredClone(t));
  });

  // Load file icon names when component mounts.
  $effect(() => {
    getIconNames().then(names => { iconFileNames = names; });
  });

  // Currently editing template (derived from index).
  let editingTemplate = $derived(editingIndex !== null ? localTemplates[editingIndex] : null);

  // All known label names from the board registry.
  let allLabelNames = $derived(Object.keys($labelColors).sort());

  // Adds a new blank template and opens it for editing.
  function addTemplate(): void {
    const newTmpl = {
      name: "",
      labels: [],
      icon: "",
      estimate: undefined,
      counter: undefined,
      checklistTitle: "",
      checklist: [],
    } as unknown as daedalus.CardTemplate;
    localTemplates = [...localTemplates, newTmpl];
    editingIndex = localTemplates.length - 1;
    emojiValue = "";
    dirty = true;
  }

  // Removes a template by index and persists immediately.
  function deleteTemplate(idx: number): void {
    localTemplates = localTemplates.filter((_, i) => i !== idx);
    if (editingIndex === idx) {
      editingIndex = null;
    } else if (editingIndex !== null && editingIndex > idx) {
      editingIndex--;
    }
    confirmingDelete = null;
    save();
  }

  // Opens/closes the editor for a template row.
  function toggleEdit(idx: number): void {
    if (editingIndex === idx) {
      editingIndex = null;
    } else {
      editingIndex = idx;
      const tmpl = localTemplates[idx];
      emojiValue = (tmpl.icon && !isFileIcon(tmpl.icon)) ? tmpl.icon : "";
    }
    confirmingDelete = null;
  }

  // Updates a field on the editing template.
  function updateField(field: string, value: unknown): void {
    if (editingIndex === null) {
      return;
    }
    const updated = [...localTemplates];
    updated[editingIndex] = { ...updated[editingIndex], [field]: value } as daedalus.CardTemplate;
    localTemplates = updated;
    dirty = true;
  }

  // Toggles a label on/off for the editing template.
  function toggleLabel(label: string): void {
    if (!editingTemplate) {
      return;
    }
    const current = editingTemplate.labels || [];
    if (current.includes(label)) {
      updateField("labels", current.filter(l => l !== label));
    } else {
      updateField("labels", [...current, label]);
    }
  }

  // Sets the icon from emoji input.
  function commitEmoji(): void {
    const trimmed = emojiValue.trim();
    updateField("icon", trimmed);
  }

  // Sets the icon from file icon grid.
  function selectFileIcon(name: string): void {
    updateField("icon", name);
    emojiValue = "";
  }

  // Clears the icon.
  function clearIcon(): void {
    updateField("icon", "");
    emojiValue = "";
  }

  // Handles estimate input.
  function handleEstimateInput(e: Event): void {
    const raw = (e.target as HTMLInputElement).value.trim();
    if (raw === "") {
      updateField("estimate", undefined);
    } else {
      const num = parseFloat(raw);
      if (!isNaN(num)) {
        updateField("estimate", num);
      }
    }
  }

  // Adds a default counter to the editing template.
  function addCounter(): void {
    updateField("counter", { current: 0, max: 10, start: 0, step: 1, label: "" });
  }

  // Removes the counter from the editing template.
  function removeCounter(): void {
    updateField("counter", undefined);
  }

  // Updates a single counter field.
  function updateCounterField(field: string, value: unknown): void {
    if (!editingTemplate || !editingTemplate.counter) {
      return;
    }
    updateField("counter", { ...editingTemplate.counter, [field]: value });
  }

  // Adds a checklist item to the editing template.
  function addChecklistItem(): void {
    if (editingIndex === null) {
      return;
    }
    const tmpl = localTemplates[editingIndex];
    const checklist = [...(tmpl.checklist || [])];
    checklist.push({ idx: checklist.length, desc: "", done: false } as daedalus.CheckListItem);
    updateField("checklist", checklist);
  }

  // Updates a checklist item description.
  function updateChecklistItem(itemIdx: number, desc: string): void {
    if (editingIndex === null) {
      return;
    }
    const tmpl = localTemplates[editingIndex];
    const checklist = [...(tmpl.checklist || [])];
    checklist[itemIdx] = { ...checklist[itemIdx], desc } as daedalus.CheckListItem;
    updateField("checklist", checklist);
  }

  // Removes a checklist item.
  function removeChecklistItem(itemIdx: number): void {
    if (editingIndex === null) {
      return;
    }
    const tmpl = localTemplates[editingIndex];
    const checklist = (tmpl.checklist || []).filter((_, i) => i !== itemIdx);
    updateField("checklist", checklist);
  }

  // Strips Svelte 5 reactive proxies and undefined values so the data is safe for Wails RPC.
  function toPlainTemplates(tmpls: daedalus.CardTemplate[]): daedalus.CardTemplate[] {
    return JSON.parse(JSON.stringify($state.snapshot(tmpls)));
  }

  // Saves all templates to the backend and updates the store.
  async function save(): Promise<void> {
    const valid = toPlainTemplates(localTemplates.filter(t => t.name && t.name.trim()));
    try {
      await SaveTemplates(valid);
      templates.set(valid);
      localTemplates = valid.map(t => structuredClone(t));
      editingIndex = null;
      dirty = false;
      addToast("Templates saved", "success");
    } catch (e) {
      addToast(`Failed to save templates: ${e}`);
    }
  }

  // Builds a short summary of what a template sets.
  function templateSummary(tmpl: daedalus.CardTemplate): string {
    const parts: string[] = [];
    if (tmpl.labels && tmpl.labels.length > 0) {
      parts.push(`${tmpl.labels.length} label${tmpl.labels.length > 1 ? "s" : ""}`);
    }
    if (tmpl.icon) {
      parts.push("icon");
    }
    if (tmpl.estimate != null) {
      parts.push(`est: ${tmpl.estimate}`);
    }
    if (tmpl.counter) {
      const c = tmpl.counter;
      const label = c.label ? `${c.label}: ` : "";
      parts.push(`${label}${c.start || 0}-${c.max}`);
    }
    if (tmpl.checklist && tmpl.checklist.length > 0) {
      parts.push(`${tmpl.checklist.length} checklist item${tmpl.checklist.length > 1 ? "s" : ""}`);
    }
    return parts.length > 0 ? parts.join(", ") : "empty";
  }
</script>

<div class="modal-backdrop centered z-high" role="presentation" use:backdropClose={onclose}>
  <div class="modal-dialog size-md" role="dialog">
    <div class="modal-header">
      <h2 class="modal-title">Template Manager</h2>
      <div class="header-actions">
        <button class="modal-close" onclick={onclose} title="Close">
          <Icon name="close" size={16} />
        </button>
      </div>
    </div>
    <div class="editor-body">
      {#if localTemplates.length === 0}
        <p class="empty-msg">No templates yet. Add one to get started.</p>
      {:else}
        <table class="manager-table">
          <thead>
            <tr>
              <th class="col-name">Name</th>
              <th class="col-summary">Sets</th>
              <th class="col-actions"></th>
            </tr>
          </thead>
          <tbody>
            {#each localTemplates as tmpl, idx}
              <tr class="template-row" class:active={editingIndex === idx}>
                <td class="col-name">
                  <button class="template-name-btn" onclick={() => toggleEdit(idx)}>
                    <span class="row-chevron" class:open={editingIndex === idx}>
                      <Icon name="chevron-down" size={10} />
                    </span>
                    {tmpl.name || "(untitled)"}
                  </button>
                </td>
                <td class="col-summary">
                  <span class="template-summary">{templateSummary(tmpl)}</span>
                </td>
                <td class="col-actions">
                  <div class="actions-inner">
                    {#if editingIndex === idx}
                      <button class="btn-icon edit-btn" onclick={save} title="Save template">
                        <Icon name="check" size={12} />
                      </button>
                    {:else}
                      <button class="btn-icon edit-btn" onclick={() => toggleEdit(idx)} title="Edit template">
                        <Icon name="pencil" size={12} />
                      </button>
                    {/if}
                    {#if confirmingDelete === idx}
                      <button class="btn-icon delete-btn confirming" onclick={() => deleteTemplate(idx)} title="Click again to confirm">confirm?</button>
                    {:else}
                      <button class="btn-icon delete-btn" onclick={() => { confirmingDelete = idx; }} title="Delete template">
                        <Icon name="trash" size={12} />
                      </button>
                    {/if}
                  </div>
                </td>
              </tr>
              {#if editingIndex === idx && editingTemplate}
                <tr class="editor-row">
                  <td colspan="3">
                    <div class="template-editor">
                      <div class="editor-top-row">
                        <div class="editor-field grow">
                          <span class="field-label">Name</span>
                          <input type="text" class="form-input" value={editingTemplate.name} placeholder="Template name"
                            oninput={(e) => updateField("name", (e.target as HTMLInputElement).value)}
                          />
                        </div>
                        <div class="editor-field est-field">
                          <span class="field-label">Estimate</span>
                          <input type="text" inputmode="numeric" class="form-input" oninput={handleEstimateInput} placeholder="Hrs"
                            value={editingTemplate.estimate != null ? String(editingTemplate.estimate) : ""}
                          />
                        </div>
                      </div>
                      <div class="editor-divider"></div>

                      <div class="editor-section">
                        <span class="field-label">Labels</span>
                        {#if allLabelNames.length > 0}
                          <div class="label-picker">
                            {#each allLabelNames as label}
                              {@const active = (editingTemplate.labels || []).includes(label)}
                              <button class="label-chip" class:active style="background: {labelColor(label, $labelColors)}" title={active ? `Remove ${label}` : `Add ${label}`}
                                onclick={() => toggleLabel(label)}
                              >{label}</button>
                            {/each}
                          </div>
                        {:else}
                          <span class="no-items-hint">No labels on this board yet</span>
                        {/if}
                      </div>
                      <div class="editor-divider"></div>

                      <div class="editor-section">
                        <div class="icon-header-row">
                          <span class="field-label">
                            Icon
                            {#if editingTemplate.icon}
                              <span class="icon-badge">
                                {#if isFileIcon(editingTemplate.icon)}
                                  <CardIcon name={editingTemplate.icon} size={12} />
                                {:else}
                                  {editingTemplate.icon}
                                {/if}
                              </span>
                              <button class="icon-clear-btn" onclick={clearIcon} title="Remove icon">
                                <Icon name="close" size={10} />
                              </button>
                            {/if}
                          </span>
                          <div class="emoji-row">
                            <input class="form-input emoji-input" type="text" placeholder="Emoji..." bind:value={emojiValue} onkeydown={e => e.key === 'Enter' && commitEmoji()}/>
                            <button class="emoji-save-btn" onclick={commitEmoji}>Set</button>
                          </div>
                        </div>
                        {#if iconFileNames.length > 0}
                          <div class="icon-grid">
                            {#each iconFileNames as name}
                              <button class="icon-grid-option" class:active={name === editingTemplate.icon} title={name} onclick={() => selectFileIcon(name)}>
                                <CardIcon name={name} size={16} />
                              </button>
                            {/each}
                          </div>
                        {/if}
                      </div>
                      <div class="editor-divider"></div>

                      <div class="editor-section">
                        {#if editingTemplate.counter}
                          <div class="counter-row">
                            <span class="field-label">Counter</span>
                            <input type="text" class="form-input counter-label-input" value={editingTemplate.counter.label || ""} placeholder="Label"
                              oninput={(e) => updateCounterField("label", (e.target as HTMLInputElement).value)}
                            />
                            <input type="text" inputmode="numeric" class="form-input counter-num-input" value={String(editingTemplate.counter.start || 0)} title="From"
                              oninput={(e) => {
                                const n = parseInt((e.target as HTMLInputElement).value, 10);
                                if (!isNaN(n)) { updateCounterField("start", n); }
                              }}
                            />
                            <span class="counter-sep">to</span>
                            <input type="text" inputmode="numeric" class="form-input counter-num-input" value={String(editingTemplate.counter.max)} title="To"
                              oninput={(e) => {
                                const n = parseInt((e.target as HTMLInputElement).value, 10);
                                if (!isNaN(n)) { updateCounterField("max", n); }
                              }}
                            />
                            <span class="counter-sep">by</span>
                            <input type="text" inputmode="numeric" class="form-input counter-num-input" value={String(editingTemplate.counter.step || 1)} title="Step"
                              oninput={(e) => {
                                const n = parseInt((e.target as HTMLInputElement).value, 10);
                                if (!isNaN(n) && n >= 1) { updateCounterField("step", n); }
                              }}
                            />
                            <button class="btn-icon checklist-remove" title="Remove counter" onclick={removeCounter}>
                              <Icon name="close" size={10} />
                            </button>
                          </div>
                        {:else}
                          <button class="add-checklist-btn" onclick={addCounter}>
                            <Icon name="plus" size={10} /> Add counter
                          </button>
                        {/if}
                      </div>

                      {#if (editingTemplate.checklist && editingTemplate.checklist.length > 0) || true}
                        <div class="editor-divider"></div>

                        <div class="editor-section">
                          <span class="field-label">Checklist</span>
                          <div class="checklist-editor">
                            {#each editingTemplate.checklist || [] as item, itemIdx}
                              <div class="checklist-row">
                                <span class="checklist-num">{itemIdx + 1}.</span>
                                <input type="text" class="form-input checklist-input" value={item.desc} placeholder="Checklist item"
                                  oninput={(e) => updateChecklistItem(itemIdx, (e.target as HTMLInputElement).value)}
                                />
                                <button class="btn-icon checklist-remove" title="Remove item" onclick={() => removeChecklistItem(itemIdx)}>
                                  <Icon name="close" size={10} />
                                </button>
                              </div>
                            {/each}
                            <button class="add-checklist-btn" onclick={addChecklistItem}>
                              <Icon name="plus" size={10} /> Add item
                            </button>
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

      <div class="template-actions">
        <button class="add-template-btn" onclick={addTemplate}>
          <Icon name="plus" size={12} /> New Template
        </button>
        {#if dirty}
          <button class="save-template-btn" onclick={save}>Save</button>
        {/if}
      </div>
    </div>
  </div>
</div>

<style lang="scss">
  .col-name {
    width: 40%;
  }

  .col-summary {
    width: 100%;
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

  .template-row.active td {
    border-bottom-color: transparent;
  }

  .template-name-btn {
    all: unset;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 0.82rem;
    color: var(--color-text-primary);
    font-weight: 500;
    cursor: pointer;
    border-radius: 3px;
    padding: 2px 4px;

    &:hover {
      background: var(--overlay-hover-light);
    }
  }

  .row-chevron {
    display: inline-flex;
    color: var(--color-text-muted);
    transition: transform 0.15s;
    transform: rotate(-90deg);

    &.open {
      transform: rotate(0deg);
    }
  }

  .template-summary {
    font-size: 0.72rem;
    color: var(--color-text-muted);
    font-style: italic;
  }

  .edit-btn {
    width: 20px;
    height: 20px;

    &:hover {
      color: var(--color-accent);
    }
  }

  .delete-btn {
    width: 20px;
    height: 20px;

    &:hover {
      color: var(--color-error);
    }

    &.confirming {
      width: auto;
      font-size: 0.68rem;
      font-weight: 600;
      color: var(--color-error);
      padding: 2px 6px;
    }
  }

  .editor-row td {
    padding: 0;
    border-bottom: 1px solid var(--color-border);
  }

  .template-editor {
    display: flex;
    flex-direction: column;
    gap: 0;
    background: var(--color-bg-base);
    border-radius: 6px;
    margin: 4px 0 6px 0;
    padding: 2px 0;
    border: 1px solid var(--color-border);
  }

  .editor-top-row {
    display: flex;
    gap: 10px;
    padding: 10px 14px 8px;
  }

  .editor-field {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }

  .grow {
    flex: 1;
  }

  .est-field {
    width: 64px;
    flex-shrink: 0;
  }

  .editor-section {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 8px 14px;
  }

  .icon-header-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .icon-header-row .emoji-row {
    flex: 1;
    min-width: 0;
  }

  .editor-divider {
    height: 1px;
    background: var(--color-border);
    margin: 0 10px;
  }

  .field-label {
    font-size: 0.68rem;
    font-weight: 600;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .no-items-hint {
    font-size: 0.75rem;
    color: var(--color-text-muted);
    font-style: italic;
  }

  .label-picker {
    display: flex;
    flex-wrap: wrap;
    gap: 5px;
  }

  .label-chip {
    all: unset;
    font-size: 0.68rem;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 3px;
    color: var(--color-text-inverse);
    cursor: pointer;
    opacity: 0.3;
    transition: opacity 0.12s, transform 0.1s;

    &:hover {
      opacity: 0.65;
    }

    &.active {
      opacity: 1;
      box-shadow: 0 0 0 2px var(--color-bg-base), 0 0 0 3px currentColor;
    }
  }

  .icon-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: var(--overlay-hover-medium);
    border-radius: 3px;
    padding: 1px 4px;
    font-size: 0.8rem;
    line-height: 1;
  }

  .icon-clear-btn {
    all: unset;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 14px;
    height: 14px;
    cursor: pointer;
    color: var(--color-text-muted);
    border-radius: 2px;

    &:hover {
      color: var(--color-error);
    }
  }

  .icon-grid {
    grid-template-columns: repeat(auto-fill, minmax(30px, 1fr));
    background: var(--overlay-subtle);
    border-radius: 4px;
    padding: 4px;
  }

  .icon-grid-option.active {
    background: var(--overlay-accent);
    box-shadow: inset 0 0 0 1px var(--overlay-accent-border);
  }

  .counter-row {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .counter-label-input {
    flex: 1;
    min-width: 0;
    font-size: 0.75rem;
    padding: 3px 6px;
  }

  .counter-num-input {
    width: 36px;
    flex-shrink: 0;
    text-align: center;
    font-family: var(--font-mono);
    font-size: 0.75rem;
    padding: 3px 4px;
  }

  .counter-sep {
    font-size: 0.68rem;
    color: var(--color-text-muted);
    flex-shrink: 0;
  }

  .checklist-editor {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .checklist-row {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .checklist-num {
    font-size: 0.7rem;
    font-family: var(--font-mono);
    color: var(--color-text-muted);
    width: 16px;
    text-align: right;
    flex-shrink: 0;
  }

  .checklist-input {
    flex: 1;
  }

  .checklist-remove {
    flex-shrink: 0;
    width: 18px;
    height: 18px;
    color: var(--color-text-muted);

    &:hover {
      color: var(--color-error);
    }
  }

  .add-checklist-btn {
    all: unset;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 0.75rem;
    color: var(--color-text-muted);
    cursor: pointer;
    padding: 3px 6px;
    border-radius: 3px;
    margin-left: 22px;

    &:hover {
      color: var(--color-text-primary);
      background: var(--overlay-hover-light);
    }
  }

  .template-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    padding-top: 12px;
  }

  .add-template-btn {
    all: unset;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 0.75rem;
    color: var(--color-text-primary);
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    cursor: pointer;
    padding: 5px 12px;
    border-radius: 4px;

    &:hover {
      background: var(--overlay-hover);
      border-color: var(--color-text-tertiary);
    }
  }

  .save-template-btn {
    all: unset;
    display: inline-flex;
    align-items: center;
    font-size: 0.75rem;
    color: var(--color-text-primary);
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    cursor: pointer;
    padding: 5px 12px;
    border-radius: 4px;
    margin-left: auto;

    &:hover {
      background: var(--overlay-hover);
      border-color: var(--color-text-tertiary);
    }
  }
</style>
