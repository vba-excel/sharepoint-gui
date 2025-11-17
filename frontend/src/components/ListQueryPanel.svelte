<!-- sharepoint-gui/frontend/src/components/ListQueryPanel.svelte -->
<script lang="ts">
  import sp, { type ListResponse, type ListQuery } from '../api/api';
  import { toasts } from '../lib/toast';
  import { stamped } from '../lib/helpers/filename';
  import { exportXLSX, exportCSV, exportJSON } from '../lib/exports';
  import { normalizeErr } from '../lib/errors';

  export let setAppBusy: (v:boolean)=>void = () => {};

  // estado (tipado)
  let list: string = 'tblRegistos';
  let select: string = 'Id,Matricula,Operador,DataHora';
  let filter: string = '';
  let orderby: string = 'ID desc';
  let top: number = 5;
  let all: boolean = false;
  let latestOnly: boolean = false;

  let busy = false;
  let exportBusy = false; // NOVO: UX de export
  let error = '';
  let result: ListResponse | null = null;
  let elapsed: number = 0;

  function prettyJSON(v: any): string {
    try { return JSON.stringify(v, null, 2); }
    catch { return String(v ?? ''); }
  }

  // helpers para exports (select → cols)
  function collectAllKeys(rows: any[]): string[] {
    const seen = new Set<string>();
    for (const r of rows || []) for (const k of Object.keys(r || {})) if (!k.startsWith('__')) seen.add(k);
    return Array.from(seen);
  }
  function parseSelectToColumns(sel: string): string[] {
    return (sel || '')
      .split(',').map(s => s.trim()).filter(Boolean)
      .map(s => s.replace(/\s+/g, ''))
      .map(s => s.split('/').pop() || s)
      .map(s => s.replace(/\([^)]*\)/g, ''));
  }
  function columnsForExport(sel: string, rows: any[]): string[] {
    const colsFromSelect = parseSelectToColumns(sel);
    const keysInData = new Set(collectAllKeys(rows));
    const picked = colsFromSelect.filter(c => keysInData.has(c));
    const rest = Array.from(keysInData).filter(k => !picked.includes(k)).sort((a,b)=>a.localeCompare(b));
    return [...picked, ...rest];
  }

  export function refresh() { runList(); }

  async function runList() {
    busy = true; setAppBusy(true); error = ''; result = null;
    const t0 = performance.now();
    let ok = false;
    try {
      const q: ListQuery = { list, select, filter, orderby, top, all, latestOnly };
      result = await sp.listItems(q);
      ok = true;
    } catch (e:any) {
      error = normalizeErr(e);
    } finally {
      elapsed = Math.round(performance.now() - t0);
      busy = false; setAppBusy(false);
      if (ok) {
        const n = result?.items?.length ?? 0;
        toasts.push(`Leitura concluída (${n} itens)`, 'success');
      } else if (error) {
        toasts.push(error, /cancelada/i.test(error) ? 'info' : 'error');
      }
    }
  }

  async function doExportJSON() {
    if (!result?.items?.length || exportBusy) return;
    exportBusy = true;
    const cols = columnsForExport(select, result.items);
    const saved = await exportJSON(result.items, stamped(list, 'json'), { pretty: true, preferColumns: cols });
    exportBusy = false;
    toasts.push(saved === 'saved' ? 'Export JSON concluído' : 'Export JSON cancelado', saved === 'saved' ? 'success' : 'info');
  }

  async function doExportCSV() {
    if (!result?.items?.length || exportBusy) return;
    exportBusy = true;
    const cols = columnsForExport(select, result.items);
    const saved = await exportCSV(result.items, { filename: stamped(list, 'csv'), preferColumns: cols });
    exportBusy = false;
    toasts.push(saved === 'saved' ? 'Export CSV concluído' : 'Export CSV cancelado', saved === 'saved' ? 'success' : 'info');
  }

  async function doExportXLSX() {
    if (!result?.items?.length || exportBusy) return;
    exportBusy = true;
    const cols = columnsForExport(select, result.items);
    const saved = await exportXLSX(result.items, {
      filename: stamped(list, 'xlsx'),
      sheetName: 'Items',
      preferColumns: cols,
      autoDates: true,
      freezeHeader: true,
      // Exemplo (podes ajustar conforme a lista):
      // colFormats: { DataHora: 'yyyy-mm-dd hh:mm' }
    });
    exportBusy = false;
    toasts.push(saved === 'saved' ? 'Export XLSX concluído' : 'Export XLSX cancelado', saved === 'saved' ? 'success' : 'info');
  }
</script>

<h2>Listar</h2>
<div class="row">
  <label for="list">List</label>
  <input id="list" bind:value={list} />
</div>
<div class="row">
  <label for="select">Select</label>
  <input id="select" bind:value={select} />
</div>
<div class="row">
  <label for="filter">Filter</label>
  <input id="filter" bind:value={filter} placeholder="ex.: Matricula eq '57RT01'" />
</div>
<div class="row">
  <label for="orderby">OrderBy</label>
  <input id="orderby" bind:value={orderby} />
</div>
<div class="row">
  <label for="top">Top</label>
  <input id="top" type="number" bind:value={top} min="0" />
</div>
<div class="row">
  <span id="flags-label">Flags</span>
  <div class="inline" role="group" aria-labelledby="flags-label">
    <label for="flag-all"><input id="flag-all" type="checkbox" bind:checked={all} /> All</label>
    <label for="flag-latest"><input id="flag-latest" type="checkbox" bind:checked={latestOnly} /> Latest only</label>
  </div>
</div>
<div class="row">
  <span></span>
  <div class="inline">
    <button on:click={runList} disabled={busy || exportBusy}>{busy ? 'A ler…' : 'Listar'}</button>
    <button on:click={doExportJSON} disabled={!result?.items?.length || busy || exportBusy}>
      {exportBusy ? 'A exportar…' : 'Export JSON'}
    </button>
    <button on:click={doExportCSV} disabled={!result?.items?.length || busy || exportBusy}>
      {exportBusy ? 'A exportar…' : 'Export CSV'}
    </button>
    <button on:click={doExportXLSX} disabled={!result?.items?.length || busy || exportBusy}>
      {exportBusy ? 'A exportar…' : 'Export XLSX'}
    </button>
  </div>
</div>

{#if error}
  <pre class="error">{error}</pre>
{/if}

{#if result}
  <h3>Resumo</h3>
  <div class="summary">
    <div><b>items</b><div>{result.summary?.items ?? 0}</div></div>
    <div><b>pages</b><div>{result.summary?.pages ?? 0}</div></div>
    <div><b>throttled</b><div>{String(result.summary?.throttled ?? false)}</div></div>
    <div><b>partial</b><div>{String(result.summary?.partial ?? false)}</div></div>
    <div><b>fallback</b><div>{String(result.summary?.fallback ?? false)}</div></div>
    <div><b>stoppedEarly</b><div>{String(result.summary?.stoppedEarly ?? false)}</div></div>
    <div><b>elapsed</b><div>{elapsed} ms</div></div>
  </div>

  <h3>Items</h3>
  <div class="tablewrap">
    <table>
      <thead>
        <tr>
          {#if result.items && result.items[0]}
            {#each Object.keys(result.items[0]) as k}
              <th>{k}</th>
            {/each}
          {/if}
        </tr>
      </thead>
      <tbody>
        {#each result.items as row}
          <tr>
            {#each Object.keys(result.items[0] || {}) as k}
              <td>{row[k]}</td>
            {/each}
          </tr>
        {/each}
      </tbody>
    </table>
  </div>

  <details>
    <summary>JSON completo</summary>
    <pre>{prettyJSON(result)}</pre>
  </details>
{/if}
