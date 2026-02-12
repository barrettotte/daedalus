<script lang="ts">
  import { toasts } from "../stores/board";
</script>

{#if $toasts.length > 0}
  <div class="toast-container">
    {#each $toasts as toast (toast.id)}
      <div class="toast">{toast.message}</div>
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
    background: #3b1c1c;
    border: 1px solid rgba(248, 113, 104, 0.3);
    color: #f87168;
    font-size: 0.8rem;
    padding: 10px 16px;
    border-radius: 6px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
    max-width: 360px;
    word-break: break-word;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, sans-serif;
  }
</style>
