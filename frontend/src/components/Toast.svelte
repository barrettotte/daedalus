<script lang="ts">
  // Displays a stack of temporary error/success/info notifications that auto-dismiss after a timeout.

  import { toasts } from "../stores/board";
</script>

{#if $toasts.length > 0}
  <div class="toast-container">
    {#each $toasts as toast (toast.id)}
      <div class="toast {toast.type}">{toast.message}</div>
    {/each}
  </div>
{/if}

<style lang="scss">
  .toast-container {
    position: fixed;
    bottom: 16px;
    right: 16px;
    z-index: 2000;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .toast {
    font-size: 0.8rem;
    padding: 10px 16px;
    border-radius: 6px;
    box-shadow: var(--shadow-md);
    max-width: 360px;
    word-break: break-word;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, sans-serif;

    &.error {
      background: var(--color-toast-bg);
      border: 1px solid var(--overlay-error-border);
      color: var(--color-error);
    }

    &.success {
      background: var(--overlay-success);
      border: 1px solid var(--color-success);
      color: var(--color-success);
    }

    &.info {
      background: var(--color-bg-elevated);
      border: 1px solid var(--color-border);
      color: var(--color-text-secondary);
    }
  }
</style>
