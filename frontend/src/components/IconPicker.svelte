<script lang="ts">
  // Upload-based icon picker. Shows a grid of uploaded icons and an upload button.

  import { getIconNames, saveCustomIcon, downloadIcon } from "../lib/icons";
  import { addToast } from "../stores/board";
  import CardIcon from "./CardIcon.svelte";

  let {
    currentIcon = "",
    onselect,
  }: {
    currentIcon?: string;
    onselect?: (name: string) => void;
  } = $props();

  let iconNames: string[] = $state([]);
  let loadGeneration = 0;

  function loadIcons(): void {
    const gen = ++loadGeneration;
    getIconNames().then(names => {
      if (gen === loadGeneration) {
        iconNames = names;
      }
    });
  }

  $effect(() => {
    loadIcons();
  });

  let fileInput: HTMLInputElement | undefined = $state();

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
        // Strip the data:image/png;base64, prefix -- backend expects raw base64
        const commaIdx = dataUrl.indexOf(",");
        content = commaIdx >= 0 ? dataUrl.slice(commaIdx + 1) : dataUrl;
      }

      await saveCustomIcon(file.name, content);
      loadIcons();
      onselect?.(file.name);
    } catch (err) {
      addToast(`Failed to save icon: ${err}`);
    }

    input.value = "";
  }

  // Which input row is open: "url", "lucide", or null
  let activeInput: "url" | "lucide" | null = $state(null);
  let urlValue = $state("");
  let lucideSlug = $state("");
  let downloading = $state(false);

  function toggleInput(mode: "url" | "lucide"): void {
    activeInput = activeInput === mode ? null : mode;
  }

  async function handleUrlDownload(): Promise<void> {
    const url = urlValue.trim();
    if (!url) {
      return;
    }

    downloading = true;
    try {
      const filename = await downloadIcon(url);
      loadIcons();
      onselect?.(filename);
      urlValue = "";
      activeInput = null;
    } catch (err) {
      addToast(`Failed to download icon: ${err}`);
    } finally {
      downloading = false;
    }
  }

  async function handleLucideDownload(): Promise<void> {
    const slug = lucideSlug.trim().toLowerCase();
    if (!slug) {
      return;
    }
    if (!/^[a-z0-9-]+$/.test(slug)) {
      addToast("Invalid Lucide slug -- use lowercase letters, numbers, and dashes");
      return;
    }

    downloading = true;
    try {
      const url = `https://unpkg.com/lucide-static/icons/${slug}.svg`;
      const filename = await downloadIcon(url);
      loadIcons();
      onselect?.(filename);
      lucideSlug = "";
      activeInput = null;
    } catch (err) {
      addToast(`Lucide icon "${slug}" not found or download failed`);
    } finally {
      downloading = false;
    }
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

<div class="icon-picker">
  {#if iconNames.length > 0}
    <div class="icon-grid">
      {#each iconNames as iconName}
        <button class="icon-option" class:active={iconName === currentIcon} title={iconName} onclick={() => onselect?.(iconName)}>
          <CardIcon name={iconName} size={16} />
        </button>
      {/each}
    </div>
  {:else}
    <div class="empty-state">No icons uploaded</div>
  {/if}

  <div class="picker-actions">
    <button class="action-link" onclick={triggerUpload}>Upload</button>
    <button class="action-link" onclick={() => toggleInput("url")}>URL</button>
    <button class="action-link" onclick={() => toggleInput("lucide")}>Lucide</button>
    {#if currentIcon}
      <button class="action-link remove" onclick={() => onselect?.("")}>
        Remove
      </button>
    {/if}
  </div>

  {#if activeInput === "url"}
    <div class="input-row">
      <input class="slug-input" type="text" placeholder="https://example.com/icon.svg" bind:value={urlValue}
        onkeydown={(e) => e.key === "Enter" && handleUrlDownload()} disabled={downloading}
      />
      <button class="action-link" onclick={handleUrlDownload} disabled={downloading || !urlValue.trim()}>
        {downloading ? "..." : "Go"}
      </button>
    </div>
  {:else if activeInput === "lucide"}
    <div class="input-row">
      <input class="slug-input" type="text" placeholder="lucide slug" bind:value={lucideSlug}
        onkeydown={(e) => e.key === "Enter" && handleLucideDownload()} disabled={downloading}
      />
      <button class="action-link" onclick={handleLucideDownload} disabled={downloading || !lucideSlug.trim()}>
        {downloading ? "..." : "Go"}
      </button>
    </div>
  {/if}

  <input bind:this={fileInput} type="file" accept=".svg,.png" class="hidden-input" onchange={handleFileChange}/>
</div>

<style lang="scss">
  .icon-picker {
    margin-top: 4px;
  }

  .icon-grid {
    display: grid;
    grid-template-columns: repeat(5, 1fr);
    gap: 2px;
    margin-bottom: 8px;
  }

  .icon-option {
    all: unset;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 6px;
    border-radius: 4px;
    color: var(--color-text-tertiary);
    cursor: pointer;

    &:hover {
      background: var(--overlay-hover);
      color: var(--color-text-primary);
    }

    &.active {
      background: var(--overlay-hover-medium);
      color: var(--color-accent);
    }
  }

  .empty-state {
    font-size: 0.75rem;
    color: var(--color-text-muted);
    font-style: italic;
    padding: 4px 0;
    margin-bottom: 8px;
  }

  .picker-actions {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .action-link {
    all: unset;
    font-size: 0.75rem;
    color: var(--color-text-primary);
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    border-radius: 4px;
    padding: 3px 8px;
    cursor: pointer;
    box-sizing: border-box;

    &:hover {
      background: var(--overlay-hover);
      border-color: var(--color-text-tertiary);
    }

    &.remove:hover {
      color: var(--color-error);
    }

    &:disabled {
      opacity: 0.4;
      cursor: not-allowed;
    }
  }

  .input-row {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-top: 4px;
  }

  .slug-input {
    flex: 1;
    min-width: 0;
    background: var(--color-bg-base);
    border: 1px solid var(--color-border);
    color: var(--color-text-primary);
    font-size: 0.8rem;
    padding: 2px 6px;
    border-radius: 4px;
    outline: none;
    box-sizing: border-box;

    &::placeholder {
      color: var(--color-text-muted);
    }

    &:focus {
      border-color: var(--color-accent);
    }

    &:disabled {
      opacity: 0.5;
    }
  }

  .hidden-input {
    display: none;
  }
</style>
