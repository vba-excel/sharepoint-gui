<script lang="ts">
  import sp from '../api/api';
  import { toasts } from '../lib/toast';
  import {
    attachmentsAccept,
    maxUploadMBStore,
    csvModeStore,
    type CsvMode
  } from '../lib/settings';
  import { setDownloadMode, type DownloadMode } from '../lib/download';

  export let cfgPath = localStorage.getItem('sp_cfg') || 'private.json';
  export let siteURL = localStorage.getItem('sp_site') || '';
  export let gt: number = Number(localStorage.getItem('sp_gt') ?? '60');
  export let clean: boolean = (localStorage.getItem('sp_clean') ?? '0') === '1';

  // Download mode (vive em lib/download.ts)
  export let dlMode: DownloadMode =
    (localStorage.getItem('sp_dl_mode') === 'dialog') ? 'dialog' : 'auto';

  // CSV mode (ligado à store para refletir imediatamente)
  export let csvMode: CsvMode =
    (localStorage.getItem('sp_csv_mode') === 'pt') ? 'pt' : 'standard';

  // Attachments settings
  export let attAccept: string = localStorage.getItem('sp_att_accept') ?? '';
  export let attMaxMB: number = Number(localStorage.getItem('sp_att_maxmb') ?? '50');

  async function pickConfig() {
    const p = await sp.openConfigDialog();
    if (p) cfgPath = p as string;
  }

  async function saveConfig() {
    await sp.setConfig({
      ConfigPath: cfgPath,
      SiteURL: siteURL,
      GlobalTimeoutSec: Number(gt),
      CleanOutput: clean
    });

    // Persistência (mantida por compatibilidade)
    localStorage.setItem('sp_cfg', cfgPath);
    localStorage.setItem('sp_site', siteURL);
    localStorage.setItem('sp_gt', String(gt));
    localStorage.setItem('sp_clean', clean ? '1' : '0');
    localStorage.setItem('sp_dl_mode', dlMode);
    localStorage.setItem('sp_csv_mode', csvMode);
    localStorage.setItem('sp_att_accept', attAccept);
    localStorage.setItem('sp_att_maxmb', String(attMaxMB > 0 ? attMaxMB : 50));

    // Aplicar imediatamente nas stores (evita reiniciar a app)
    setDownloadMode(dlMode);                           // Download Mode
    csvModeStore.set(csvMode);                         // CSV delimiter live
    attachmentsAccept.set(attAccept);                  // filtro anexos live
    maxUploadMBStore.set(attMaxMB > 0 ? attMaxMB : 50);// limite upload live

    toasts.push('Config guardada', 'success');
  }
</script>

<h2>Settings</h2>
<div class="row3">
  <div class="row">
    <label for="cfg">private.json</label>
    <input id="cfg" bind:value={cfgPath} />
    <button on:click={pickConfig}>Procurar…</button>
  </div>
  <div class="row">
    <label for="site">Site URL (override)</label>
    <input id="site" bind:value={siteURL} placeholder="(opcional)" />
  </div>
  <div class="row">
    <label for="gt">Global timeout (s)</label>
    <input id="gt" type="number" min="0" bind:value={gt} />
  </div>
  <div class="row">
    <span id="global-flags-label">Global flags</span>
    <div class="inline" role="group" aria-labelledby="global-flags-label">
      <label for="clean"><input id="clean" type="checkbox" bind:checked={clean} /> Clean output (__…)</label>
    </div>
  </div>

  <div class="row">
    <label for="dlmode">Download</label>
    <select id="dlmode" bind:value={dlMode}>
      <option value="auto">Auto (sem diálogo)</option>
      <option value="dialog">Pedir localização (save dialog)</option>
    </select>
  </div>

  <div class="row">
    <label for="csvmode">CSV</label>
    <select id="csvmode" bind:value={csvMode}>
      <option value="standard">Padrão (vírgula “,”)</option>
      <option value="pt">PT/Excel (ponto e vírgula “;”)</option>
    </select>
  </div>

  <div class="row">
    <label for="attaccept">Attachments accept</label>
    <input id="attaccept" bind:value={attAccept} placeholder=".pdf,.png,.xlsx,image/*" />
  </div>
  <div class="row">
    <label for="attmax">Max upload (MB)</label>
    <input id="attmax" type="number" min="1" bind:value={attMaxMB} />
  </div>

  <div class="row">
    <span></span>
    <button on:click={saveConfig}>Guardar</button>
  </div>
</div>
