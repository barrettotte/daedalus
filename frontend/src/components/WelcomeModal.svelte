<script lang="ts">
  // Welcome modal for board selection -- shown on startup when no default board is set,
  // or opened via the switch-board button/Ctrl+O.

  import {
    GetAppConfig, SetDefaultBoard, RemoveRecentBoard, OpenDirectoryDialog, LoadBoard,
  } from "../../wailsjs/go/main/App";
  import { main } from "../../wailsjs/go/models";
  import { backdropClose } from "../lib/utils";
  import Icon from "./Icon.svelte";
  import appIcon from "../assets/images/daedalus.svg";

  let {
    isOverlay = false,
    onclose,
    onboard,
  }: {
    isOverlay?: boolean;
    onclose?: () => void;
    onboard: (response: main.BoardResponse) => void;
  } = $props();

  let recentBoards: Array<{ path: string; title: string; lastOpened: string }> = $state([]);
  let defaultBoard = $state("");
  let loading = $state(false);

  // Loads the app config and populates the recent boards list.
  async function loadConfig(): Promise<void> {
    try {
      const cfg = await GetAppConfig();
      recentBoards = cfg.recentBoards || [];
      defaultBoard = cfg.defaultBoard || "";
    } catch (e) {
      console.error("Failed to load app config:", e);
    }
  }

  // Opens a board by path -- calls LoadBoard, then notifies parent.
  async function openBoard(path: string): Promise<void> {
    if (loading || !path) {
      return;
    }
    loading = true;
    try {
      const response = await LoadBoard(path);
      if (response) {
        onboard(response);
      }
    } catch (e) {
      console.error("Failed to open board:", e);
    } finally {
      loading = false;
    }
  }

  // Opens the native directory picker and loads the selected board.
  async function handleOpenBoard(): Promise<void> {
    const path = await OpenDirectoryDialog();
    if (path) {
      await openBoard(path);
    }
  }

  // Opens a directory picker for creating a new board (same as open -- empty dirs get initialized).
  async function handleCreateBoard(): Promise<void> {
    const path = await OpenDirectoryDialog();
    if (path) {
      await openBoard(path);
    }
  }

  // Removes a board from the recent list.
  async function handleRemoveRecent(path: string, e: MouseEvent): Promise<void> {
    e.stopPropagation();
    try {
      await RemoveRecentBoard(path);
      recentBoards = recentBoards.filter(b => b.path !== path);
    } catch (err) {
      console.error("Failed to remove recent board:", err);
    }
  }

  // Toggles a board as the default.
  async function handleToggleDefault(path: string, e: MouseEvent): Promise<void> {
    e.stopPropagation();
    try {
      const newDefault = defaultBoard === path ? "" : path;
      await SetDefaultBoard(newDefault);
      defaultBoard = newDefault;
    } catch (err) {
      console.error("Failed to set default board:", err);
    }
  }

  // Extracts the directory name from a full path for display.
  function dirName(path: string): string {
    const parts = path.replace(/\/+$/, "").split("/");
    return parts[parts.length - 1] || path;
  }

  loadConfig();
</script>

{#if isOverlay}
  <div class="modal-backdrop centered z-high" role="presentation" use:backdropClose={onclose!}>
    <div class="modal-dialog size-md welcome-modal" role="dialog">
      <div class="modal-header">
        <h2 class="modal-title">Switch Board</h2>
        <button class="modal-close" onclick={onclose} title="Close">
          <Icon name="close" size={16} />
        </button>
      </div>
      {@render modalBody()}
    </div>
  </div>
{:else}
  <div class="modal-backdrop centered" role="presentation">
    <div class="modal-dialog size-md welcome-modal" role="dialog">
      <div class="welcome-header">
        <img src={appIcon} alt="" class="welcome-icon" />
        <h1 class="welcome-title">Daedalus</h1>
      </div>
      {@render modalBody()}
    </div>
  </div>
{/if}

{#snippet modalBody()}
  <div class="welcome-body">
    {#if recentBoards.length > 0}
      <div class="recent-section">
        <span class="recent-label">Recent boards</span>
        <div class="recent-list">
          {#each recentBoards as board}
            <div class="recent-row" class:disabled={loading} role="button" tabindex="0"
              onclick={() => openBoard(board.path)}
              onkeydown={(e) => (e.key === "Enter" || e.key === " ") && openBoard(board.path)}
            >
              <Icon name="folder" size={16} />
              <div class="recent-info">
                <span class="recent-name">{board.title || dirName(board.path)}</span>
                <span class="recent-path">{board.path}</span>
              </div>
              <button class="recent-action" title={defaultBoard === board.path ? "Unset default" : "Set as default"}
                onclick={(e) => handleToggleDefault(board.path, e)}
              >
                <Icon name={defaultBoard === board.path ? "star-filled" : "star"} size={14} />
              </button>
              <button class="recent-action" title="Remove from recent" onclick={(e) => handleRemoveRecent(board.path, e)}>
                <Icon name="close" size={12} />
              </button>
            </div>
          {/each}
        </div>
      </div>
    {:else}
      <div class="empty-state">
        <span class="empty-text">No recent boards</span>
      </div>
    {/if}
    <div class="welcome-actions">
      <button class="welcome-btn" onclick={handleOpenBoard} disabled={loading}>
        <Icon name="folder" size={16} />
        Open Board
      </button>
      <button class="welcome-btn" onclick={handleCreateBoard} disabled={loading}>
        <Icon name="plus" size={16} />
        Create Board
      </button>
    </div>
  </div>
{/snippet}

<style lang="scss">
  .welcome-modal {
    user-select: none;
  }

  .welcome-header {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    padding: 28px 20px 12px 20px;
  }

  .welcome-icon {
    width: 64px;
    height: 64px;
  }

  .welcome-title {
    margin: 0;
    font-size: 1.6rem;
    font-weight: 700;
    color: var(--color-text-primary);
  }

  .welcome-body {
    padding: 12px 20px 20px 20px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .recent-section {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .recent-label {
    font-size: 0.68rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-text-muted);
  }

  .recent-list {
    display: flex;
    flex-direction: column;
    gap: 2px;
    max-height: 320px;
    overflow-y: auto;
  }

  .recent-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 10px;
    border-radius: 6px;
    cursor: pointer;
    color: var(--color-text-secondary);
    transition: background 0.1s;

    &:hover {
      background: var(--overlay-hover-light);
    }

    &.disabled {
      opacity: 0.5;
      cursor: not-allowed;
      pointer-events: none;
    }
  }

  .recent-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }

  .recent-name {
    font-size: 0.88rem;
    font-weight: 600;
    color: var(--color-text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .recent-path {
    font-size: 0.72rem;
    color: var(--color-text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .recent-action {
    all: unset;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    border-radius: 4px;
    cursor: pointer;
    color: var(--color-text-muted);
    flex-shrink: 0;

    &:hover {
      color: var(--color-text-primary);
      background: var(--overlay-hover-medium);
    }
  }

  .empty-state {
    text-align: center;
    padding: 20px 0;
  }

  .empty-text {
    font-size: 0.85rem;
    color: var(--color-text-muted);
  }

  .welcome-actions {
    display: flex;
    gap: 10px;
  }

  .welcome-btn {
    all: unset;
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 10px 16px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--color-text-secondary);
    background: var(--overlay-hover-light);
    border: 1px solid var(--color-border);
    transition: background 0.15s, color 0.15s;

    &:hover {
      background: var(--overlay-hover-medium);
      color: var(--color-text-primary);
    }

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  }
</style>
