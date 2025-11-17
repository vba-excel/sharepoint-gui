<script lang="ts">
  import { toasts } from '../lib/toast';
</script>

<div class="toast-container" aria-live="polite">
  {#each $toasts as t (t.id)}
    <div class="toast {t.kind}">{t.text}</div>
  {/each}
</div>

<style>
  .toast-container { position: fixed; top: 16px; right: 16px; display: flex; flex-direction: column; gap: 8px; z-index: 1000; }
  .toast { padding: 10px 12px; border-radius: 10px; box-shadow: 0 4px 12px rgba(0,0,0,.12); color: #fff; background: #333; animation: slideIn .18s ease, fadeOut .25s ease 2.3s forwards; }
  .toast.success { background: #1f7a1f; }
  .toast.error   { background: #b00020; }
  .toast.info    { background: #2a5bd7; }
  @keyframes slideIn { from { opacity: 0; transform: translateY(-6px); } to { opacity: 1; transform: translateY(0); } }
  @keyframes fadeOut { to   { opacity: 0; transform: translateY(-6px); } }
</style>
