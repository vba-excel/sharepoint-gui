<script lang="ts">
  import { onMount } from 'svelte';
  import Overlay from './components/Overlay.svelte';
  import Toasts from './components/Toasts.svelte';
  import SettingsPanel from './components/SettingsPanel.svelte';
  import ListQueryPanel from './components/ListQueryPanel.svelte';
  import CrudPanel from './components/CrudPanel.svelte';
  import AttachmentsPanel from './components/AttachmentsPanel.svelte';
  import ThemeToggle from './components/ThemeToggle.svelte';
  import { toasts } from './lib/toast';
  import sp from './api/api';

  let appBusy = false;
  let cancelBtn: HTMLButtonElement | null = null;

  let cfgPath = localStorage.getItem('sp_cfg') || 'private.json';
  let siteURL = localStorage.getItem('sp_site') || '';
  let gt = Number(localStorage.getItem('sp_gt') ?? '60');
  let clean = (localStorage.getItem('sp_clean') ?? '0') === '1';

  $: if (appBusy && cancelBtn) cancelBtn.focus();

  function onKey(e: KeyboardEvent) {
    if (!appBusy) return;
    if (e.key === 'Escape' || e.key === 'Esc') { e.preventDefault(); cancelOp(); }
  }

  onMount(async () => {
    try { await sp.ping(); } catch {}
    await sp.setConfig({ ConfigPath: cfgPath, SiteURL: siteURL, GlobalTimeoutSec: Number(gt), CleanOutput: clean });
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  });

  function setAppBusy(v: boolean) { appBusy = v; }

  async function cancelOp() {
    try {
      const had = await sp.cancelCurrent();
      if (had) toasts.push('A cancelar…', 'info', 1500);
    } catch {}
  }

  let listPanelRef: any;
  function refreshListIfAny() { listPanelRef?.refresh(); }
</script>

<main>
  <div class="inline" style="justify-content: space-between; align-items: center;">
    <h1>SharePoint GUI (PoC)</h1>
    <ThemeToggle />
  </div>

  <section class="card">
    <SettingsPanel bind:cfgPath bind:siteURL bind:gt bind:clean />
  </section>

  <section class="card">
    <ListQueryPanel bind:this={listPanelRef} {setAppBusy} />
  </section>

  <section class="card">
    <CrudPanel {setAppBusy} onChanged={refreshListIfAny} />
  </section>

  <section class="card">
    <AttachmentsPanel />
  </section>

  <Overlay visible={appBusy} onCancel={cancelOp} bind:cancelBtn />
  <Toasts />
</main>
