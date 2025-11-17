<script lang="ts">
  import sp from '../api/api';
  import { toasts } from '../lib/toast';
  import { normalizeErr } from '../lib/errors';

  export let setAppBusy: (v:boolean)=>void = () => {};
  export let onChanged: () => void = () => {};

  let wo_list = 'tblRegTestes';
  let wo_select = 'Id,Matricula,Operador,DataHora';
  let wo_id: number = 0;
  let wo_fields_json = '{ "Matricula": "ABCD01", "Operador": "999999" }';
  let lastWrite: any = null;
  let busy = false;
  let error = '';

  function prettyJSON(v: any): string {
    try { return JSON.stringify(v, null, 2); } catch { return String(v ?? ''); }
  }
  function parseFieldsJSON(): Record<string, any> {
    try { return wo_fields_json.trim() ? JSON.parse(wo_fields_json) : {}; }
    catch { throw new Error('Campos (JSON) inválidos'); }
  }
  function idOf(obj:any) { return obj?.ID ?? obj?.Id ?? obj?.id; }

  async function doAdd() {
    busy = true; setAppBusy(true); error = ''; lastWrite = null;
    let ok = false;
    try {
      const fields = parseFieldsJSON();
      lastWrite = await sp.addItem(wo_list, fields, wo_select);
      ok = true; onChanged();
    } catch (e:any) { error = normalizeErr(e); }
    finally {
      busy = false; setAppBusy(false);
      if (ok) toasts.push(`Adicionado${idOf(lastWrite) ? ' (ID '+idOf(lastWrite)+')' : ''}`, 'success');
      else if (error) toasts.push(error, /cancelada/i.test(error) ? 'info' : 'error');
    }
  }

  async function doUpdate() {
    busy = true; setAppBusy(true); error = ''; lastWrite = null;
    let ok = false;
    try {
      if (!wo_id || wo_id <= 0) throw new Error('ID inválido');
      const fields = parseFieldsJSON();
      lastWrite = await sp.updateItem(wo_list, wo_id, fields, wo_select);
      ok = true; onChanged();
    } catch (e:any) { error = normalizeErr(e); }
    finally {
      busy = false; setAppBusy(false);
      if (ok) toasts.push(`Atualizado (ID ${idOf(lastWrite) ?? wo_id})`, 'success');
      else if (error) toasts.push(error, /cancelada/i.test(error) ? 'info' : 'error');
    }
  }

  async function doDelete() {
    busy = true; setAppBusy(true); error = ''; lastWrite = null;
    let ok = false;
    try {
      if (!wo_id || wo_id <= 0) throw new Error('ID inválido');
      const okDel = await sp.deleteItem(wo_list, wo_id);
      lastWrite = { deleted: okDel, id: wo_id };
      ok = okDel; onChanged();
    } catch (e:any) { error = normalizeErr(e); }
    finally {
      busy = false; setAppBusy(false);
      if (ok) toasts.push(`Eliminado (ID ${wo_id})`, 'success');
      else if (error) toasts.push(error, /cancelada/i.test(error) ? 'info' : 'error');
    }
  }
</script>

<!-- markup igual ao teu -->
<h2>Write (Add / Update / Delete)</h2>
<div class="row">
  <label for="wo_list">List</label>
  <input id="wo_list" bind:value={wo_list} />
</div>
<div class="row">
  <label for="wo_select">Select (readback)</label>
  <input id="wo_select" bind:value={wo_select} />
</div>
<div class="row">
  <label for="wo_id">ID</label>
  <input id="wo_id" type="number" bind:value={wo_id} min="0" />
</div>
<div class="row">
  <label for="wo_fields">Campos (JSON)</label>
  <textarea id="wo_fields" bind:value={wo_fields_json} rows="4" spellcheck="false"></textarea>
</div>
<div class="row">
  <span></span>
  <div class="inline">
    <button on:click={doAdd} disabled={busy}>Add</button>
    <button on:click={doUpdate} disabled={busy}>Update</button>
    <button on:click={doDelete} disabled={busy}>Delete</button>
  </div>
</div>

{#if lastWrite}
  <h3>Resultado</h3>
  <pre>{prettyJSON(lastWrite)}</pre>
{/if}
{#if error}
  <pre class="error">{error}</pre>
{/if}
