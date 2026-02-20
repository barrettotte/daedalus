<script lang="ts">
  // Shared URI sidebar widget. Shows three states: URL with actions,
  // "URI" header when editing, or "+ Add URI" button when empty.

  import { copyToClipboard } from "../lib/utils";
  import Icon from "./Icon.svelte";

  let {
    url,
    editing = false,
    onopen,
    onedit,
    onremove,
  }: {
    url: string;
    editing?: boolean;
    onopen: () => void;
    onedit: () => void;
    onremove: () => void;
  } = $props();
</script>

{#if url}
  <div class="sidebar-section">
    <div class="section-header">
      <h4 class="sidebar-title">URI</h4>
      <div class="section-header-actions">
        <button class="counter-header-btn" title={url} onclick={() => onopen()}>
          <Icon name="link" size={12} />
        </button>
        <button class="counter-header-btn" title="Copy URI" onclick={() => copyToClipboard(url, "URI")}>
          <Icon name="copy" size={12} />
        </button>
        <button class="counter-header-btn" title="Edit URI" onclick={() => onedit()}>
          <Icon name="pencil" size={12} />
        </button>
        <button class="counter-header-btn remove" title="Remove URI" onclick={() => onremove()}>
          <Icon name="trash" size={12} />
        </button>
      </div>
    </div>
  </div>
{:else if editing}
  <div class="sidebar-section">
    <div class="section-header">
      <h4 class="sidebar-title">URI</h4>
    </div>
  </div>
{:else}
  <div class="sidebar-section">
    <button class="add-counter-btn" title="Add URI" onclick={() => onedit()}>
      + Add URI
    </button>
  </div>
{/if}
