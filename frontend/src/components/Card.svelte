<script>
  export let card;
  $: meta = card.metadata;
  $: isOverdue = meta.due ? new Date(meta.due) < new Date() : false;
</script>

<div class="card">
  <div class="card-header">
    <span class="id">#{meta.id}</span>
    {#if meta.due}
      <span class="due-badge" class:overdue={isOverdue}>
        {new Date(meta.due).toLocaleDateString()}
      </span>
    {/if}
  </div>
  <div class="title">{meta.title}</div>

  {#if meta.tags && meta.tags.length > 0}
    <div class="tags">
      {#each meta.tags as tag}
        <span class="tag">{tag}</span>
      {/each}
    </div>
  {/if}

  <div class="preview">
    {card.previewText.slice(0, 80)}...
  </div>
</div>

<style>
  .card {
    background: #2b303b;
    border-radius: 6px;
    padding: 10px;
    margin: 0 4px 8px 0;
    border-left: 4px solid #4a90e2;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
    height: 90%;
    color: #dcdcdc;
    font-family: "Segoe UI", Tahoma, Geneva, Verdana, sans-serif;
    overflow: hidden;
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    font-size: 0.75rem;
    color: #888;
    margin-bottom: 4px;
  }

  .title {
    font-weight: 600;
    font-size: 0.95rem;
    margin-bottom: 6px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .tags {
    display: flex;
    gap: 4px;
    margin-bottom: 6px;
  }

  .tag {
    background: #3e4451;
    font-size: 0.7rem;
    padding: 2px 6px;
    border-radius: 4px;
  }

  .preview {
    font-size: 0.8rem;
    color: #aaa;
    line-height: 1.2;
  }

  .due-badge {
    background: #2ecc71;
    color: #fff;
    padding: 0 4px;
    border-radius: 3px;
  }
  .due-badge.overdue {
    background: #e74c3c;
  }
</style>
