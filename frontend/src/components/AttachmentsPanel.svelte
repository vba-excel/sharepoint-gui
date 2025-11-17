<!-- sharepoint-gui/frontend/src/components/AttachmentsPanel.svelte -->
<script lang="ts">
  import { onDestroy } from 'svelte';
  import sp, { type AttachmentInfo } from '../api/api';
  import { toasts } from '../lib/toast';
  import { normalizeErr } from '../lib/errors';
  import { saveAttachment, saveByURL, type SaveResult } from '../lib/download';
  import { attachmentsAccept, maxUploadMBStore, maxUploadBytesStore } from '../lib/settings';

  export let defaultList = '';
  export let defaultItemId: number | null = null;

  let attList: string = defaultList || '';
  let attItemID: number = defaultItemId ?? 0;

  let attBusy = false;
  let attErr = '';
  let attItems: AttachmentInfo[] = [];

  // Upload múltiplo + DnD + limites
  let attFiles: File[] = [];
  let fileInput: HTMLInputElement | null = null;
  let dndOver = false;
  let sizeExceeded = false;
  let totalSizeBytes = 0;

  // progress simples (por ficheiros)
  let uploadProg = 0;

  $: if (defaultList && attList !== defaultList) attList = defaultList;
  $: if (defaultItemId && attItemID !== defaultItemId) attItemID = defaultItemId;

  // Limites (MB para UI; BYTES para validação)
  let maxMB = 0;
  let maxBytes = 0;
  $: maxMB   = $maxUploadMBStore;
  $: maxBytes = $maxUploadBytesStore;

  // ===== Ícones (escolha da variante) =====
  type DeleteIcon = 'trash2' | 'trash' | 'bin' | 'bucket';
  let deleteIcon: DeleteIcon = 'trash2';

  type DownloadIcon = 'arrow' | 'tray' | 'cloud' | 'save';
  let downloadIcon: DownloadIcon = 'arrow';

  // Helpers
  function updateSizeState() {
    totalSizeBytes = attFiles.reduce((sum, f) => sum + (f?.size ?? 0), 0);
    sizeExceeded = totalSizeBytes > maxBytes;
  }

  function parseAcceptList(list: string): string[] {
    return (list || '').split(',').map(s => s.trim()).filter(Boolean);
  }
  function matchesAccept(f: File, acceptList: string[]): boolean {
    if (!acceptList.length) return true;
    const name = f.name.toLowerCase();
    const type = (f.type || '').toLowerCase();
    for (const rule of acceptList) {
      const r = rule.toLowerCase();
      if (r.endsWith('/*')) {
        const base = r.slice(0, -2);
        if (type.startsWith(base)) return true;
      } else if (r.startsWith('.')) {
        if (name.endsWith(r)) return true;
      } else {
        if (type === r) return true; // MIME exato
      }
    }
    return false;
  }

  // merge sem duplicados (chave: name|size|lastModified)
  function mergeFiles(existing: File[], incoming: File[]): File[] {
    const key = (f: File) => `${f.name}|${f.size}|${f.lastModified}`;
    const map = new Map<string, File>();
    for (const f of existing) map.set(key(f), f);
    for (const f of incoming) {
      const k = key(f);
      if (!map.has(k)) map.set(k, f);
    }
    return Array.from(map.values());
  }

  function handleFilesPicked(fs: File[]) {
    const acc = parseAcceptList($attachmentsAccept);
    const filtered = fs.filter(f => matchesAccept(f, acc));
    const rejected = fs.length - filtered.length;
    if (rejected > 0) toasts.push(`Alguns ficheiros foram ignorados pelo filtro "Attachments accept".`, 'info');
    attFiles = mergeFiles(attFiles, filtered); // APPEND + dedupe
    updateSizeState();
  }

  function handleFilesChange(e: Event) {
    const input = e.currentTarget as HTMLInputElement | null;
    handleFilesPicked(Array.from(input?.files ?? []));
  }

  function clearSelection() {
    attFiles = [];
    if (fileInput) fileInput.value = '';
    updateSizeState();
  }
  function removeFileAt(idx: number) {
    attFiles = attFiles.filter((_, i) => i !== idx);
    updateSizeState();
  }

  function onDragEnter(e: DragEvent) { e.preventDefault(); dndOver = true; }
  function onDragOver(e: DragEvent)  { e.preventDefault(); if (!dndOver) dndOver = true; }
  function onDragLeave(_e: DragEvent) { dndOver = false; }
  function onDrop(e: DragEvent) {
    e.preventDefault();
    dndOver = false;
    const files = Array.from(e.dataTransfer?.files ?? []);
    if (!files.length) return;
    handleFilesPicked(files);
  }

  async function refreshAtt() {
    if (!attList || !attItemID) { attErr = 'Indique List e ID.'; return; }
    attBusy = true; attErr = '';
    try {
      clearThumbs();
      attItems = await sp.listAttachments(attList, attItemID);
      toasts.push(`Encontrados ${attItems.length} anexos`, 'success');
      // prewarm
      for (const a of attItems) if (isThumbable(a)) ensureThumb(a).catch(()=>{});
    } catch (e:any) {
      attErr = normalizeErr(e);
      toasts.push(attErr, /cancelada/i.test(attErr) ? 'info' : 'error');
    } finally {
      attBusy = false;
    }
  }

  async function uploadAtts() {
    if (!attList || !attItemID || attFiles.length === 0) { attErr = 'Falta List, ID ou ficheiros.'; return; }
    if (sizeExceeded) { toasts.push(`Tamanho total excede ${maxMB} MB.`, 'error'); return; }
    attBusy = true; attErr = ''; uploadProg = 0;
    let ok = 0, fail = 0;
    const failed: { name: string; err: string }[] = [];
    try {
      const total = attFiles.length;
      for (let i = 0; i < total; i++) {
        const f = attFiles[i];
        try {
          const ab = await f.arrayBuffer();
          const bytes: number[] = Array.from(new Uint8Array(ab));
          await sp.addAttachment(attList, attItemID, f.name, bytes);
          ok++;
        } catch (e:any) {
          fail++; failed.push({ name: f.name, err: normalizeErr(e) });
        } finally {
          uploadProg = Math.round(((i + 1) / total) * 100);
        }
      }
      await refreshAtt();
      clearSelection();
      if (fail === 0) toasts.push(`Enviados ${ok} ficheiro(s) com sucesso`, 'success');
      else if (ok > 0) { toasts.push(`Parcial: ${ok} OK, ${fail} falhou`, 'info'); attErr = failed.map(x => `• ${x.name}: ${x.err}`).join('\n'); }
      else { toasts.push(`Falhou o envio de ${fail} ficheiro(s)`, 'error'); attErr = failed.map(x => `• ${x.name}: ${x.err}`).join('\n'); }
    } finally {
      attBusy = false;
      setTimeout(()=> uploadProg = 0, 600);
    }
  }

  async function downloadAtt(a: AttachmentInfo) {
    attBusy = true; attErr = '';
    try {
      const res: SaveResult = await saveAttachment(attList, attItemID, a.fileName);
      if (res === 'saved') toasts.push(`Download "${a.fileName}" concluído`, 'success');
      else toasts.push(`Download "${a.fileName}" cancelado`, 'info');
    } catch (e:any) {
      attErr = normalizeErr(e);
      toasts.push(attErr, /cancelada/i.test(attErr) ? 'info' : 'error');
    } finally {
      attBusy = false;
    }
  }

  async function deleteAtt(a: AttachmentInfo) {
    if (!confirm(`Remover o anexo "${a.fileName}"?`)) return;
    attBusy = true; attErr = '';
    try {
      await sp.deleteAttachment(attList, attItemID, a.fileName);
      toasts.push(`Eliminado: ${a.fileName}`, 'success');
      await refreshAtt();
    } catch (e:any) {
      attErr = normalizeErr(e);
      toasts.push(attErr, /cancelada/i.test(attErr) ? 'info' : 'error');
    } finally {
      attBusy = false;
    }
  }

  let attUrl = '';
  async function downloadByUrl() {
    attBusy = true; attErr = '';
    try {
      const res: SaveResult = await saveByURL(attUrl);
      if (res === 'saved') toasts.push('Download por URL concluído', 'success');
      else toasts.push('Download por URL cancelado', 'info');
    } catch (e:any) {
      attErr = normalizeErr(e);
      toasts.push(attErr, /cancelada/i.test(attErr) ? 'info' : 'error');
    } finally {
      attBusy = false;
    }
  }

  function fmtSize(n: number) {
    if (!Number.isFinite(n)) return '';
    const units = ['B','KB','MB','GB','TB'];
    let u = 0; let v = n;
    while (v >= 1024 && u < units.length-1) { v /= 1024; u++; }
    return `${v.toFixed(v >= 10 ? 0 : 1)} ${units[u]}`;
  }

  // ===== Preview Modal =====
  let previewOpen = false;
  let previewName = '';
  let previewType: 'image' | 'pdf' | 'text' | 'unknown' = 'unknown';
  let previewSrc = '';
  let previewText = '';
  let previewErr = '';

  function extOf(name: string): string {
    const i = name.lastIndexOf('.');
    return i >= 0 ? name.slice(i+1).toLowerCase() : '';
  }
  function mimeForExt(ext: string): string {
    switch (ext) {
      case 'png': return 'image/png';
      case 'jpg':
      case 'jpeg': return 'image/jpeg';
      case 'gif': return 'image/gif';
      case 'webp': return 'image/webp';
      case 'svg': return 'image/svg+xml';
      case 'pdf': return 'application/pdf';
      case 'txt': return 'text/plain; charset=utf-8';
      case 'csv': return 'text/csv; charset=utf-8';
      case 'json': return 'application/json; charset=utf-8';
      default: return 'application/octet-stream';
    }
  }
  function classifyForPreview(ext: string): 'image'|'pdf'|'text'|'unknown' {
    if (['png','jpg','jpeg','gif','webp','svg'].includes(ext)) return 'image';
    if (ext === 'pdf') return 'pdf';
    if (['txt','csv','json','log','md'].includes(ext)) return 'text';
    return 'unknown';
  }
  function closePreview() {
    if (previewSrc) URL.revokeObjectURL(previewSrc);
    previewOpen = false;
    previewSrc = '';
    previewText = '';
    previewErr = '';
  }
  async function previewAtt(a: AttachmentInfo) {
    previewOpen = true;
    previewName = a.fileName;
    previewErr = '';
    const ext = extOf(a.fileName);
    previewType = classifyForPreview(ext);

    try {
      const data = await sp.downloadAttachment(attList, attItemID, a.fileName);
      const arr = Array.isArray(data)
        ? new Uint8Array(data as number[])
        : (() => { const bin = atob(String(data)); const u8 = new Uint8Array(bin.length); for (let i=0;i<bin.length;i++) u8[i] = bin.charCodeAt(i); return u8; })();
      const mime = mimeForExt(ext);
      const blob = new Blob([arr], { type: mime });

      if (previewType === 'image' || previewType === 'pdf') {
        previewSrc = URL.createObjectURL(blob);
      } else if (previewType === 'text') {
        const text = await blob.text();
        previewText = text.length > 200_000 ? (text.slice(0, 200_000) + '\n\n…(truncado)…') : text;
      } else {
        previewErr = 'Pré-visualização não suportada para este tipo de ficheiro.';
      }
    } catch (e:any) {
      previewErr = normalizeErr(e);
    }
  }

  // ===== Thumbnails =====
  const thumbURLs = new Map<string, string>();
  const thumbPromises = new Map<string, Promise<void>>();

  function isThumbable(a: AttachmentInfo): boolean {
    const ext = extOf(a.fileName);
    return ['png','jpg','jpeg','gif','webp','svg'].includes(ext);
  }
  function thumbKey(a: AttachmentInfo): string {
    return `${a.serverRelativeUrl}|${a.fileName}`;
  }
  function getThumbURL(a: AttachmentInfo): string | null {
    return thumbURLs.get(thumbKey(a)) ?? null;
  }
  function ensureThumb(a: AttachmentInfo): Promise<void> {
    const key = thumbKey(a);
    if (thumbURLs.has(key)) return Promise.resolve();
    if (!isThumbable(a)) return Promise.resolve();
    const inflight = thumbPromises.get(key);
    if (inflight) return inflight;

    const p = (async () => {
      try {
        const data = await sp.downloadAttachment(attList, attItemID, a.fileName);
        const arr = Array.isArray(data)
          ? new Uint8Array(data as number[])
          : (() => { const bin = atob(String(data)); const u8 = new Uint8Array(bin.length); for (let i=0;i<bin.length;i++) u8[i] = bin.charCodeAt(i); return u8; })();
        const blob = new Blob([arr], { type: mimeForExt(extOf(a.fileName)) });
        const url  = URL.createObjectURL(blob);
        thumbURLs.set(key, url);
      } catch {
        // falhou gerar a thumb — apenas não define URL; o UI mostrará o placeholder 'broken'
      }
    })();

    thumbPromises.set(key, p);
    p.finally(() => { thumbPromises.delete(key); });
    return p;
  }

  function clearThumbs() {
    for (const url of thumbURLs.values()) { try { URL.revokeObjectURL(url); } catch {} }
    thumbURLs.clear();
    thumbPromises.clear();
  }
  onDestroy(clearThumbs);

  // ===== Placeholder icons (por extensão) =====
  function fileIconForExt(name: string): string {
    const e = extOf(name);
    if (['png','jpg','jpeg','gif','webp','svg','bmp','tiff','ico'].includes(e)) return 'file-image';
    if (e === 'pdf') return 'file-pdf';
    if (['txt','log','md','rtf'].includes(e)) return 'file-text';
    if (['csv','tsv','xls','xlsx','ods'].includes(e)) return 'file-table';
    if (['doc','docx','odt'].includes(e)) return 'file-doc';
    if (['ppt','pptx','odp'].includes(e)) return 'file-ppt';
    if (['zip','7z','rar','gz','tgz','tar','bz2'].includes(e)) return 'file-zip';
    if (['json','xml','yaml','yml','ini','cfg'].includes(e)) return 'file-code';
    return 'file';
  }
</script>

<!-- Sprite de ícones (inline) -->
<svg aria-hidden="true" style="position:absolute;width:0;height:0;overflow:hidden">
  <!-- ===== DELETE VARIANTS ===== -->
  <symbol id="ic-trash2" viewBox="0 0 24 24">
    <path d="M9.5 4A1.5 1.5 0 0 1 11 2.5h2A1.5 1.5 0 0 1 14.5 4V5H20a1 1 0 1 1 0 2h-1.1l-1 12.1A3 3 0 0 1 14.9 22H9.1a3 3 0 0 1-2.99-2.9L5.1 7H4a1 1 0 1 1 0-2h5.5V4zM7.11 7l.98 12.02c.06.71.65 1.25 1.36 1.25h5.1c.71 0 1.3-.54 1.36-1.25L16.89 7H7.11zM10 9a1 1 0 0 1 1 1v7a1 1 0 1 1-2 0v-7a1 1 0 0 1 1-1zm4 0a1 1 0 0 1 1 1v7a1 1 0 1 1-2 0v-7a1 1 0 0 1 1-1z" fill="currentColor"/>
  </symbol>
  <symbol id="ic-trash" viewBox="0 0 24 24">
    <path d="M9 3a1 1 0 0 0-1 1v1H5.5a1 1 0 1 0 0 2H6v11a3 3 0 0 0 3 3h6a3 3 0 0 0 3-3V7h.5a1 1 0 1 0 0-2H16V4a1 1 0 0 0-1-1H9zm2 2h2v1h-2V5zM8 7h10v11a1 1 0 0 1-1 1H9a1 1 0 0 1-1-1V7zm3 3a1 1 0 1 1 2 0v7a1 1 0 1 1-2 0v-7z" fill="currentColor"/>
  </symbol>
  <symbol id="ic-bin" viewBox="0 0 24 24">
    <path d="M8 4h8v2h4v2h-2v11a3 3 0 0 1-3 3H9a3 3 0 0 1-3-3V8H4V6h4V4zm1 4v11a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1V8H9z" fill="currentColor"/>
  </symbol>
  <symbol id="ic-bucket" viewBox="0 0 24 24">
    <path d="M6 7a6 6 0 0 1 12 0h2a1 1 0 1 1 0 2h-1.03l-1.3 9.09A3 3 0 0 1 14.69 21H9.31a3 3 0 0 1-2.98-2.91L5.03 9H4a1 1 0 1 1 0-2h2zm2 0h8a4 4 0 0 0-8 0z" fill="currentColor"/>
  </symbol>

  <!-- ===== DOWNLOAD VARIANTS ===== -->
  <symbol id="ic-dl-arrow" viewBox="0 0 24 24">
    <path d="M12 3a1 1 0 0 1 1 1v8.586l2.293-2.293a1 1 0 1 1 1.414 1.414l-4 4a1.5 1.5 0 0 1-2.121 0l-4-4A1 1 0 1 1 7.999 10.293L10.3 12.586V4a1 1 0 0 1 1-1zM5 17a1 1 0 0 1 1 1v1h12v-1a1 1 0 1 1 2 0v2a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1v-2a1 1 0 0 1 1-1h1z" fill="currentColor"/>
  </symbol>
  <symbol id="ic-dl-tray" viewBox="0 0 24 24">
    <path d="M12 2a1 1 0 0 1 1 1v7.586l2.293-2.293a1 1 0 1 1 1.414 1.414l-3.999 4a1.5 1.5 0 0 1-2.122 0l-4-4A1 1 0 1 1 7.999 8.293L10 10.586V3a1 1 0 0 1 1-1zM4 14h16l2 6H2l2-6zm2.618 2l-.667 2h11.098l-.667-2H6.618z" fill="currentColor"/>
  </symbol>
  <symbol id="ic-dl-cloud" viewBox="0 0 24 24">
    <path d="M7.5 10a4.5 4.5 0 0 1 8.72-1.5 4 4 0 0 1 1.78 7.667V16a1 1 0 1 1 2 0v1.5A2.5 2.5 0 0 1 17.5 20H7a4 4 0 0 1 .5-8zM12 9a1 1 0 0 1 1 1v4.586l1.293-1.293a1 1 0 1 1 1.414 1.414l-3 3a1.5 1.5 0 0 1-2.121 0l-3-3a1 1 0 1 1 1.414-1.414L11 14.586V10a1 1 0 0 1 1-1z" fill="currentColor"/>
  </symbol>
  <symbol id="ic-dl-save" viewBox="0 0 24 24">
    <path d="M5 3h11l3 3v13a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2zm9 2H6v6h8V5zm2 0v6h2V7.828L16 5zM6 19h12v-6H6v6z" fill="currentColor"/>
  </symbol>

  <!-- ====== FILE/PLACEHOLDER ICONS ====== -->
  <!-- página genérica -->
  <symbol id="ic-file" viewBox="0 0 24 24">
    <path d="M6 2h8l4 4v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2zm8 2v4h4" fill="currentColor"/>
  </symbol>
  <!-- texto (linhas) -->
  <symbol id="ic-file-text" viewBox="0 0 24 24">
    <path d="M6 2h8l4 4v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2zm8 2v4h4M8 10h8M8 14h8M8 18h6" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round"/>
  </symbol>
  <!-- imagem (montanha + sol) -->
  <symbol id="ic-file-image" viewBox="0 0 24 24">
    <path d="M6 2h8l4 4v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2zm8 2v4h4" fill="currentColor"/>
    <path d="M7 16l3-3 2 2 3-3 2 2v3H7zM9.5 9a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3z" fill="currentColor"/>
  </symbol>
  <!-- tabela/folha (xlsx/csv) -->
  <symbol id="ic-file-table" viewBox="0 0 24 24">
    <path d="M6 2h8l4 4v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2zm8 2v4h4" fill="currentColor"/>
    <path d="M7 11h10M7 15h10M9 9v10M13 9v10" stroke="currentColor" stroke-width="2" fill="none"/>
  </symbol>
  <!-- pdf (página + badge) -->
  <symbol id="ic-file-pdf" viewBox="0 0 24 24">
    <path d="M6 2h8l4 4v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2zm8 2v4h4" fill="currentColor"/>
    <circle cx="17.5" cy="17.5" r="3.5" fill="currentColor"/>
  </symbol>
  <!-- doc -->
  <symbol id="ic-file-doc" viewBox="0 0 24 24">
    <path d="M6 2h8l4 4v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2zm8 2v4h4" fill="currentColor"/>
    <path d="M8 12h8M8 16h6" stroke="currentColor" stroke-width="2"/>
  </symbol>
  <!-- ppt -->
  <symbol id="ic-file-ppt" viewBox="0 0 24 24">
    <path d="M6 2h8l4 4v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2zm8 2v4h4" fill="currentColor"/>
    <path d="M12 12a4 4 0 1 0 0 8v-4h4a4 4 0 0 0-4-4z" fill="currentColor"/>
  </symbol>
  <!-- zip -->
  <symbol id="ic-file-zip" viewBox="0 0 24 24">
    <path d="M6 2h8l4 4v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2zm8 2v4h4" fill="currentColor"/>
    <path d="M10 8v8m0-8h2m-2 2h2m-2 2h2m-2 2h2" stroke="currentColor" stroke-width="2"/>
  </symbol>
  <!-- code/json -->
  <symbol id="ic-file-code" viewBox="0 0 24 24">
    <path d="M6 2h8l4 4v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2zm8 2v4h4" fill="currentColor"/>
    <path d="M9 14l-2-2 2-2M15 10l2 2-2 2" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round"/>
  </symbol>
  <!-- broken (erro ao gerar miniatura) -->
  <symbol id="ic-file-broken" viewBox="0 0 24 24">
    <path d="M6 2h8l4 4v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2zm8 2v4h4" fill="currentColor"/>
    <path d="M8 16l8-8M16 16L8 8" stroke="currentColor" stroke-width="2"/>
  </symbol>
  <!-- broken específico imagem -->
  <symbol id="ic-image-broken" viewBox="0 0 24 24">
    <rect x="3" y="5" width="18" height="14" rx="2" ry="2" fill="none" stroke="currentColor" stroke-width="2"/>
    <path d="M7 15l3-3 2 2 3-3 2 2" stroke="currentColor" stroke-width="2" fill="none"/>
    <path d="M8 8l8 8" stroke="currentColor" stroke-width="2"/>
  </symbol>
</svg>

<h2>Attachments</h2>

<div class="row">
  <label for="att_list">List</label>
  <input id="att_list" bind:value={attList} />
</div>
<div class="row">
  <label for="att_id">Item ID</label>
  <input id="att_id" type="number" min="1" bind:value={attItemID} />
</div>

<!-- DnD + file picker (ACCEPT reativo) -->
<div class="row">
  <label for="att_file">Ficheiros</label>
  <div class="dnd-wrap">
    <div
      class="dnd-zone {dndOver ? 'over' : ''}"
      on:dragenter|preventDefault={onDragEnter}
      on:dragover|preventDefault={onDragOver}
      on:dragleave={onDragLeave}
      on:drop={onDrop}
      aria-label="Arraste & Largue ficheiros aqui"
    >
      Arraste & Largue aqui
      <small class="muted">ou</small>
      <label class="pick-btn" for="att_file">escolha ficheiros…</label>
      <input
        id="att_file"
        type="file"
        multiple
        on:change={handleFilesChange}
        bind:this={fileInput}
        accept={$attachmentsAccept}
        hidden
      />
      {#if $attachmentsAccept}<div class="accept">Filtro: <code>{$attachmentsAccept}</code></div>{/if}
      <div class="limit">Limite total: {maxMB} MB</div>
    </div>
  </div>
</div>

{#if attFiles.length}
  <div class="row">
    <span></span>
    <div class="files-box">
      <div class="files-head">
        {attFiles.length} ficheiro(s) selecionado(s)
        <span class="size">({fmtSize(totalSizeBytes)})</span>
        {#if sizeExceeded}<span class="exceeded">excede limite</span>{/if}
        <button class="link" on:click={clearSelection} disabled={attBusy}>limpar</button>
      </div>
      <ul class="filelist">
        {#each attFiles as f, i}
          <li>
            <span class="fname">{f.name}</span>
            <span class="size">({fmtSize(f.size)})</span>
            <button class="remove" title="remover" on:click={() => removeFileAt(i)} disabled={attBusy}>✕</button>
          </li>
        {/each}
      </ul>
      {#if uploadProg > 0}
        <div class="progress"><div class="bar" style="width: {uploadProg}%;"></div></div>
      {/if}
    </div>
  </div>
{/if}

<div class="row">
  <span></span>
  <div class="inline">
    <button on:click={refreshAtt} disabled={attBusy || !attList || !attItemID}>
      {attBusy ? 'A carregar…' : 'Listar anexos'}
    </button>
    <button
      on:click={uploadAtts}
      disabled={attBusy || attFiles.length === 0 || !attList || !attItemID || sizeExceeded}
      title={sizeExceeded ? `Excede {maxMB} MB` : ''}
    >
      {attBusy ? 'A enviar…' : `Enviar (${attFiles.length}) / Substituir`}
    </button>
  </div>
</div>

<div class="row">
  <label for="url_dl">Download por URL</label>
  <input id="url_dl" bind:value={attUrl} placeholder="/sites/X/… ou https://…" />
</div>
<div class="row">
  <span></span>
  <div class="inline">
    <button on:click={downloadByUrl} disabled={attBusy || !attUrl.trim()}>
      {attBusy ? 'A descarregar…' : 'Download URL'}
    </button>
  </div>
</div>

{#if attErr}
  <pre class="error">{attErr}</pre>
{/if}

{#if attItems?.length}
  <div class="tablewrap">
    <table>
      <thead>
        <tr>
          <th style="width:56px;">Preview</th>
          <th>Nome</th>
          <th>Path</th>
          <th>Ações</th>
        </tr>
      </thead>
      <tbody>
        {#each attItems as a}
          <tr>
            <td class="thumb-cell">
              <button
                class="thumb-btn"
                on:click={() => previewAtt(a)}
                title="Pré-visualizar"
                aria-label={`Pré-visualizar ${a.fileName}`}
                disabled={attBusy}
              >
                {#if isThumbable(a)}
                  {#await ensureThumb(a)}
                    <div class="thumb thumb-loading" aria-label="a carregar…"></div>
                  {:then}
                    {#if getThumbURL(a)}
                      <img class="thumb" src={getThumbURL(a) || ''} alt="thumbnail" />
                    {:else}
                      <div class="thumb thumb-fallback" title="não foi possível gerar a miniatura">
                        <svg viewBox="0 0 24 24" aria-hidden="true">
                          <use href={`#ic-image-broken`}></use>
                        </svg>
                      </div>
                    {/if}
                  {:catch}
                    <div class="thumb thumb-fallback" title="erro ao gerar a miniatura">
                      <svg viewBox="0 0 24 24" aria-hidden="true">
                        <use href="#ic-file-broken"></use>
                      </svg>
                    </div>
                  {/await}
                {:else}
                  <div class="thumb thumb-icon" title="sem preview">
                    <svg viewBox="0 0 24 24" aria-hidden="true">
                      <use href={`#ic-${fileIconForExt(a.fileName)}`}></use>
                    </svg>
                  </div>
                {/if}
              </button>
            </td>
            <td>{a.fileName}</td>
            <td style="white-space:nowrap">{a.serverRelativeUrl}</td>
            <td>
              <div class="actions">
                <button
                  class="icon-btn"
                  title="Download"
                  aria-label={`Download ${a.fileName}`}
                  on:click={() => downloadAtt(a)}
                  disabled={attBusy}
                >
                  <svg viewBox="0 0 24 24" aria-hidden="true">
                    <use href={`#ic-dl-${downloadIcon}`}></use>
                  </svg>
                </button>
                <button
                  class="icon-btn danger"
                  title="Eliminar"
                  aria-label={`Eliminar ${a.fileName}`}
                  on:click={() => deleteAtt(a)}
                  disabled={attBusy}
                >
                  <svg viewBox="0 0 24 24" aria-hidden="true">
                    <use href={`#ic-${deleteIcon}`}></use>
                  </svg>
                </button>
              </div>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}

{#if previewOpen}
  <div class="preview-overlay" role="dialog" aria-modal="true" aria-label="Pré-visualização">
    <div class="preview-card">
      <div class="preview-head">
        <div class="name" title={previewName}>{previewName}</div>
        <button class="close" on:click={closePreview} aria-label="Fechar">✕</button>
      </div>

      <div class="preview-body">
        {#if previewErr}
          <pre class="error" style="margin:0">{previewErr}</pre>
        {:else if previewType === 'image'}
          <img class="preview-img" src={previewSrc} alt="preview" />
        {:else if previewType === 'pdf'}
          <iframe class="preview-frame" src={previewSrc} title="PDF"></iframe>
        {:else if previewType === 'text'}
          <pre class="preview-text">{previewText}</pre>
        {:else}
          <div class="muted">Pré-visualização não suportada.</div>
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .dnd-wrap { width: 100%; }
  .dnd-zone {
    border: 2px dashed var(--border);
    border-radius: 10px; padding: 14px;
    text-align: center; background: var(--card-bg);
    transition: background .15s ease, border-color .15s ease, box-shadow .15s ease;
  }
  .dnd-zone.over {
    border-color: var(--accent);
    background: rgba(0,0,0,0.05);
    box-shadow: 0 0 0 3px rgba(100, 149, 237, .35);
  }
  .pick-btn { color: var(--accent); cursor: pointer; text-decoration: underline; }
  .accept, .limit, .muted { color: var(--muted); margin-top: 4px; }
  .muted { font-size: 12px; }
  .files-box { border: 1px solid var(--border); border-radius: 8px; padding: 8px; background: var(--card-bg); }
  .files-head { display:flex; align-items:center; gap:8px; color: var(--muted); }
  .filelist { margin: 6px 0 0; padding-left: 0; max-height: 160px; overflow:auto; list-style: none; }
  .filelist li { margin: 2px 0; display:flex; align-items:center; gap:8px; }
  .filelist .size { color: var(--muted); }
  .filelist .fname { overflow: hidden; text-overflow: ellipsis; }
  .remove {
    background: none; border: 1px solid var(--btn-border);
    color: var(--text); border-radius: 6px; padding: 0 6px; cursor: pointer;
  }
  .link { background: none; border: none; color: var(--accent); cursor: pointer; padding: 0; }
  .link:disabled { opacity: .6; cursor: default; }
  .exceeded { color: #b00020; font-weight: 600; }
  .progress { height: 8px; background: var(--border); border-radius: 999px; margin-top: 8px; overflow: hidden; }
  .bar { height: 100%; background: var(--accent); }

  /* Tabela de anexos + thumbnails */
  .thumb-cell { width: 56px; }
  .thumb-btn {
    padding: 0; border: none; background: none; cursor: pointer;
    width: 44px; height: 44px; display: inline-flex; align-items: center; justify-content: center;
    border-radius: 6px;
  }
  .thumb-btn:focus-visible { outline: none; box-shadow: 0 0 0 2px var(--accent); }
  .thumb {
    width: 44px; height: 44px; border-radius: 6px; object-fit: cover;
    border: 1px solid var(--border); background: #fff; display: inline-block;
    pointer-events: none;
  }
  .thumb-loading {
    animation: pulse 1.2s ease-in-out infinite;
    background: linear-gradient(90deg, rgba(0,0,0,.06), rgba(0,0,0,.12), rgba(0,0,0,.06));
    background-size: 200% 100%;
  }
  .thumb-fallback, .thumb-icon {
    width: 44px; height: 44px;
    display:flex; align-items:center; justify-content:center; font-size: 14px;
    border: 1px solid var(--border); border-radius: 6px; background: var(--card-bg);
  }
  .thumb-fallback svg, .thumb-icon svg { width: 20px; height: 20px; }
  @keyframes pulse { 0% { background-position: 200% 0; } 100% { background-position: -200% 0; } }

  /* Icon buttons nas ações */
  .actions {
    display: flex;
    gap: 6px;
    align-items: center;
  }
  .icon-btn {
    width: 32px; height: 32px;
    display: inline-flex; align-items: center; justify-content: center;
    border-radius: 8px; border: 1px solid var(--btn-border); background: var(--btn-bg);
    cursor: pointer; padding: 0;
  }
  .icon-btn:disabled { opacity: .6; cursor: default; }
  .icon-btn:focus-visible { outline: none; box-shadow: 0 0 0 2px var(--accent); }
  .icon-btn svg { width: 18px; height: 18px; fill: currentColor; }
  .icon-btn.danger { color: #b00020; border-color: #b00020; }
  .icon-btn.danger:hover { filter: brightness(0.95); }

  /* Preview modal */
  .preview-overlay { position: fixed; inset: 0; background: rgba(0,0,0,.45); display:flex; align-items:center; justify-content:center; z-index: 1200; }
  .preview-card { width: min(960px, 92vw); height: min(80vh, 780px); min-height: 260px; background: var(--card-bg); border: 1px solid var(--border); border-radius: 12px; padding: 10px; display:flex; flex-direction:column; }
  .preview-head { display:flex; align-items:center; justify-content:space-between; gap:8px; }
  .preview-head .name { font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .preview-head .close { border: 1px solid var(--btn-border); background: var(--btn-bg); border-radius: 8px; padding: 4px 8px; cursor: pointer; }

  .preview-body {
    flex: 1; min-height: 0;
    display: flex; align-items: center; justify-content: center;
    overflow: hidden;
    margin-top: 8px;
    background: var(--card-bg);
  }

  .preview-img {
    max-width: 100%;
    max-height: 100%;
    width: auto; height: auto;
    display: block;
    object-fit: contain;
  }

  .preview-frame { width: 100%; height: 100%; border: 0; background: #fff; }
  .preview-text  { margin: 0; white-space: pre-wrap; overflow: auto; flex: 1; width: 100%; border: 1px solid var(--border); border-radius: 8px; padding: 8px; background: var(--card-bg); }
</style>
