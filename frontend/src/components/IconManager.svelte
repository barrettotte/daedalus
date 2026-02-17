<script lang="ts">
  // Modal for managing uploaded icons -- upload new SVG/PNG files, preview,
  // and delete existing icons.

  import { addToast, boardPath, saveWithToast } from "../stores/board";
  import { getIconNames, saveCustomIcon, deleteIcon } from "../lib/icons";
  import { OpenFileExternal } from "../../wailsjs/go/main/App";
  import { backdropClose } from "../lib/utils";
  import Icon from "./Icon.svelte";
  import CardIcon from "./CardIcon.svelte";

  let { onclose }: { onclose: () => void } = $props();

  let iconNames: string[] = $state([]);
  let confirmingDelete = $state<string | null>(null);
  let fileInput: HTMLInputElement | undefined = $state();

  function loadIcons(): void {
    getIconNames().then(names => { iconNames = names; });
  }

  $effect(() => { loadIcons(); });

  function triggerUpload(): void {
    fileInput?.click();
  }

  async function handleFileChange(e: Event): Promise<void> {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) {
      return;
    }

    const ext = file.name.split(".").pop()?.toLowerCase();
    if (ext !== "svg" && ext !== "png") {
      addToast("Only .svg and .png files are supported");
      input.value = "";
      return;
    }

    try {
      let content: string;
      if (ext === "svg") {
        content = await file.text();
      } else {
        const dataUrl = await readFileAsDataURL(file);
        const commaIdx = dataUrl.indexOf(",");
        content = commaIdx >= 0 ? dataUrl.slice(commaIdx + 1) : dataUrl;
      }

      await saveCustomIcon(file.name, content);
      loadIcons();
      addToast("Icon uploaded", "success");

    } catch (err) {
      addToast(`Failed to save icon: ${err}`);
    }

    input.value = "";
  }

  async function handleDelete(name: string): Promise<void> {
    try {
      await deleteIcon(name);
      confirmingDelete = null;
      loadIcons();
      addToast("Icon deleted", "success");
    } catch (err) {
      addToast(`Failed to delete icon: ${err}`);
    }
  }

  function openInExplorer(): void {
    saveWithToast(OpenFileExternal($boardPath + "/assets/icons"), "open icons folder");
  }

  function readFileAsDataURL(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(reader.result as string);
      reader.onerror = () => reject(reader.error);
      reader.readAsDataURL(file);
    });
  }
</script>

<div class="modal-backdrop centered z-high" role="presentation" use:backdropClose={onclose}>
  <div class="modal-dialog size-md" role="dialog">
    <div class="modal-header">
      <h2 class="modal-title">Icon Manager</h2>
      <div class="header-actions">
        <button class="upload-btn" onclick={openInExplorer} title="Open icons folder in file manager">
          <Icon name="folder" size={12} />
        </button>
        <button class="upload-btn" onclick={triggerUpload}>Upload</button>
        <button class="modal-close" onclick={onclose} title="Close">
          <Icon name="close" size={16} />
        </button>
      </div>
    </div>
    <div class="editor-body">
      {#if iconNames.length === 0}
        <p class="empty-msg">No icons uploaded. Click Upload to add .svg or .png files.</p>
      {:else}
        <table class="icon-table">
          <thead>
            <tr>
              <th class="col-preview"></th>
              <th class="col-name">Name</th>
              <th class="col-actions"></th>
            </tr>
          </thead>
          <tbody>
            {#each iconNames as name}
              <tr class="icon-row">
                <td class="col-preview">
                  <span class="icon-preview"><CardIcon name={name} size={20} /></span>
                </td>
                <td class="col-name">
                  <span class="icon-filename">{name}</span>
                </td>
                <td class="col-actions">
                  {#if confirmingDelete === name}
                    <button class="delete-btn confirming" onclick={() => handleDelete(name)}>confirm?</button>
                  {:else}
                    <button class="delete-btn" onclick={() => { confirmingDelete = name; }} title="Delete icon">
                      <Icon name="trash" size={12} />
                    </button>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    </div>
  </div>
</div>

<input bind:this={fileInput} type="file" accept=".svg,.png" class="hidden-input" onchange={handleFileChange}/>

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

  .icon-table {
    width: 100%;
    border-collapse: collapse;
    border-spacing: 0;
  }

  .icon-table th {
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

  .col-preview {
    width: 36px;
  }

  .col-name {
    width: 100%;
  }

  .col-actions {
    text-align: right !important;
  }

  .icon-row td {
    padding: 6px 16px 6px 0;
    vertical-align: middle;
    border-bottom: 1px solid var(--color-border);
  }

  .icon-preview {
    display: inline-flex;
    color: var(--color-text-secondary);
  }

  .icon-filename {
    font-size: 0.82rem;
    color: var(--color-text-primary);
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

  .header-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .upload-btn {
    all: unset;
    font-size: 0.75rem;
    color: var(--color-text-primary);
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    border-radius: 4px;
    padding: 3px 10px;
    cursor: pointer;

    &:hover {
      background: var(--overlay-hover);
      border-color: var(--color-text-tertiary);
    }
  }

  .hidden-input {
    display: none;
  }
</style>
