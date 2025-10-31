<script lang="ts">
  import { onMount } from 'svelte';
  // Os bindings serão gerados após "wails dev/build":
  // caminho típico: wailsjs/go/main/SPGUI.ts
  import { Ping, ListItems, JSON as JSONFmt } from '../wailsjs/go/main/SPGUI';

  let list = "tblRegistos";
  let select = "Id,Matricula,Operador,DataHora";
  let filter = "";
  let orderby = "ID desc";
  let top = 5;
  let all = false;
  let latestOnly = false;

  let busy = false;
  let error = "";
  let result: any = null;

  onMount(async () => {
    try { await Ping(); } catch { /* ignore */ }
  });

  async function runList() {
    busy = true; error = "";
    try {
      result = await ListItems({ list, select, filter, orderby, top, all, latestOnly });
    } catch (e:any) {
      error = e?.message || String(e);
    } finally {
      busy = false;
    }
  }
</script>

<main>
  <h1>SharePoint GUI</h1>

  <div class="card">
    <div class="row">
      <label>List</label>
      <input bind:value={list} />
    </div>
    <div class="row">
      <label>Select</label>
      <input bind:value={select} />
    </div>
    <div class="row">
      <label>Filter</label>
      <input bind:value={filter} placeholder="ex.: Matricula eq '57RT01'" />
    </div>
    <div class="row">
      <label>OrderBy</label>
      <input bind:value={orderby} />
    </div>
    <div class="row">
      <label>Top</label>
      <input type="number" bind:value={top} min="0" />
    </div>
    <div class="row">
      <label><input type="checkbox" bind:checked={all} /> All</label>
      <label><input type="checkbox" bind:checked={latestOnly} /> Latest only</label>
    </div>
    <button on:click={runList} disabled={busy}>{busy ? "A ler..." : "Listar"}</button>
  </div>

  {#if error}
    <pre class="error">{error}</pre>
  {/if}

  {#if result}
    <h3>Resumo</h3>
    <pre>{JSON.stringify(result.summary, null, 2)}</pre>

    <h3>Items</h3>
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

    <details>
      <summary>JSON completo</summary>
      <pre>{JSONFmt(result)}</pre>
    </details>
  {/if}
</main>

<style>
  main { font-family: ui-sans-serif, system-ui; padding: 16px; }
  .card { border: 1px solid #ddd; border-radius: 8px; padding: 12px; margin-bottom: 16px; }
  .row { display: grid; grid-template-columns: 120px 1fr; gap: 8px; margin: 6px 0; }
  input { padding: 6px; }
  button { padding: 8px 12px; }
  table { border-collapse: collapse; width: 100%; margin-top: 8px; }
  th, td { border: 1px solid #eee; padding: 6px; text-align: left; }
  .error { color: #b00020; white-space: pre-wrap; }
</style>
