<script lang="ts">
  export let visible = false;
  export let onCancel: () => void = () => {};
  export let cancelBtn: HTMLButtonElement | null; // bind:this
</script>

{#if visible}
  <div class="overlay" aria-live="assertive" role="dialog" aria-modal="true" aria-labelledby="overlay-title">
    <div class="spinner" role="status" aria-label="A executar…"></div>
    <div id="overlay-title" class="overlay-text">A executar…</div>
    <div class="overlay-actions">
      <button class="btn-cancel" on:click={onCancel} bind:this={cancelBtn}>Cancelar</button>
      <div class="overlay-hint">Atalho: Esc</div>
    </div>
  </div>
{/if}

<style>
  .overlay { position: fixed; inset: 0; background: rgba(255,255,255,.65); display: flex; flex-direction: column; align-items: center; justify-content: center; z-index: 999; }
  .spinner { width: 42px; height: 42px; border: 4px solid rgba(0,0,0,.18); border-top-color: rgba(0,0,0,.6); border-radius: 50%; animation: spin 1s linear infinite; }
  .overlay-text { margin-top: 10px; color: #333; font-weight: 600; }
  .overlay-actions { margin-top: 12px; display: flex; flex-direction: column; align-items: center; gap: 6px; }
  .btn-cancel { padding: 8px 14px; border-radius: 10px; border: 1px solid #c33; background: #e55; color: #fff; font-weight: 600; cursor: pointer; }
  .btn-cancel:hover { filter: brightness(0.95); }
  .overlay-hint { font-size: 12px; color: #555; }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
