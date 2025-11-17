<script lang="ts">
  import { theme } from '../lib/theme';
  import { get } from 'svelte/store';

  function toggle() {
    const t = get(theme);
    theme.set(t === 'dark' ? 'light' : 'dark');
  }
</script>

<button on:click={toggle} aria-pressed={$theme === 'dark'} title="Alternar tema">
  {$theme === 'dark' ? '🌙 Dark' : '☀️ Light'}
</button>

<style>
  button { padding: 6px 10px; border-radius: 8px; border: 1px solid var(--btn-border); background: var(--btn-bg); color: var(--text); }
</style>
