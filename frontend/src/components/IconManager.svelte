<script lang="ts">
  // Modal for managing uploaded icons -- upload new SVG/PNG files, preview,
  // and delete existing icons.

  import { addToast, boardPath, boardData, saveWithToast } from "../stores/board";
  import { getIconNames, saveCustomIcon, deleteIcon } from "../lib/icons";
  import { OpenFileExternal } from "../../wailsjs/go/main/App";
  import { backdropClose } from "../lib/utils";
  import Icon from "./Icon.svelte";
  import CardIcon from "./CardIcon.svelte";

  let { onclose, onreload }: { onclose: () => void; onreload: () => Promise<void> } = $props();

  let iconFileNames: string[] = $state([]);
  let confirmingDelete = $state<string | null>(null);
  let fileInput: HTMLInputElement | undefined = $state();

  function loadIcons(): void {
    getIconNames().then(names => { iconFileNames = names; });
  }

  let iconCounts = $derived.by(() => {
    const counts: Record<string, number> = {};
    for (const cards of Object.values($boardData)) {
      for (const card of cards) {
        const icon = card.metadata.icon;
        if (icon) {
          counts[icon] = (counts[icon] || 0) + 1;
        }
      }
    }
    return counts;
  });

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
      await onreload();
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
      {#if iconFileNames.length === 0}
        <p class="empty-msg">No icons uploaded. Click Upload to add .svg or .png files.</p>
      {:else}
        <table class="manager-table">
          <thead>
            <tr>
              <th class="col-preview"></th>
              <th class="col-name">Name</th>
              <th class="col-count">Cards</th>
              <th class="col-actions"></th>
            </tr>
          </thead>
          <tbody>
            {#each iconFileNames as name}
              <tr class="icon-row">
                <td class="col-preview">
                  <span class="icon-preview"><CardIcon name={name} size={20} /></span>
                </td>
                <td class="col-name">
                  <span class="icon-filename">{name}</span>
                </td>
                <td class="col-count">
                  <span class="icon-count">{iconCounts[name] || 0}</span>
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

  .col-preview {
    width: 36px;
  }

  .col-name {
    width: 100%;
  }

  .col-count {
    text-align: left;
    white-space: nowrap;
  }

  .col-actions {
    text-align: right !important;
  }


  .icon-preview {
    display: inline-flex;
    color: var(--color-text-secondary);
  }

  .icon-filename {
    font-size: 0.82rem;
    color: var(--color-text-primary);
  }

  .icon-count {
    font-size: 0.78rem;
    color: var(--color-text-muted);
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
