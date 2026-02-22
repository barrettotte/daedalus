<script lang="ts">
  // Modal for managing uploaded icons -- upload new SVG/PNG files, preview,
  // and delete existing icons.

  import { addToast, boardPath, boardData, boardConfig, saveWithToast } from "../stores/board";
  import { getIconNames, saveCustomIcon, deleteIcon } from "../lib/icons";
  import { OpenFileExternal, SaveListConfig } from "../../wailsjs/go/main/App";
  import { backdropClose, joinPath } from "../lib/utils";
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

  let listCounts = $derived.by(() => {
    const counts: Record<string, number> = {};
    for (const cfg of Object.values($boardConfig)) {
      if (cfg.icon) {
        counts[cfg.icon] = (counts[cfg.icon] || 0) + 1;
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

      // Clear the icon from any lists that reference it
      const cfg = $boardConfig;
      for (const [listKey, listCfg] of Object.entries(cfg)) {
        if (listCfg.icon === name) {
          await SaveListConfig(listKey, listCfg.title || "", listCfg.limit || 0, listCfg.color || "", "");
          boardConfig.update(c => {
            c[listKey] = { ...c[listKey], icon: "" };
            return c;
          });
        }
      }

      confirmingDelete = null;
      loadIcons();
      await onreload();
      addToast("Icon deleted", "success");
    } catch (err) {
      addToast(`Failed to delete icon: ${err}`);
    }
  }

  function openInExplorer(): void {
    saveWithToast(OpenFileExternal(joinPath($boardPath, "_assets", "icons")), "open icons folder");
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
        <button class="header-btn" onclick={openInExplorer} title="Open icons folder in file manager">
          <Icon name="folder" size={12} />
        </button>
        <button class="header-btn" onclick={triggerUpload}>Upload</button>
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
              <th class="col-count">Lists</th>
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
                  <span class="icon-count">{listCounts[name] || 0}</span>
                </td>
                <td class="col-count">
                  <span class="icon-count">{iconCounts[name] || 0}</span>
                </td>
                <td class="col-actions">
                  <div class="actions-inner">
                    {#if confirmingDelete === name}
                      <button class="delete-btn confirming" onclick={() => handleDelete(name)}>confirm?</button>
                    {:else}
                      <button class="delete-btn" onclick={() => { confirmingDelete = name; }} title="Delete icon">
                        <Icon name="trash" size={12} />
                      </button>
                    {/if}
                  </div>
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

  .hidden-input {
    display: none;
  }
</style>
