// sharepoint-gui/frontend/src/lib/download.ts
//
// Helpers unificados de download/gravação que:
// - Mantêm uma store reativa de "Download Mode" (auto | dialog) neste módulo
// - Em "dialog": usam os métodos do wrapper sp (saveBytesPick / saveAttachmentPick / saveByURLPick)
// - Em "auto": fazem download imediato via <a download>
// - Devolvem sempre 'saved' | 'cancelled' para gerir toasts corretamente.

import { writable, get } from 'svelte/store';
import sp from '../api/api';
import { toU8, u8ToArrayBuffer } from './bytes';

export type DownloadMode = 'auto' | 'dialog';

// ---- Store do Download Mode (neste módulo) ----
const STORAGE_KEY = 'sp_dl_mode';

function detectInitial(): DownloadMode {
  const v = (typeof localStorage !== 'undefined')
    ? localStorage.getItem(STORAGE_KEY)
    : null;
  return v === 'dialog' ? 'dialog' : 'auto';
}

// Store pública (se precisares ler/mostrar no UI noutros pontos)
export const downloadMode = writable<DownloadMode>(detectInitial());

// Cache interno para *snapshot* estável por operação
let currentMode: DownloadMode = detectInitial();

// Sincroniza store -> cache e persiste no localStorage
downloadMode.subscribe((v) => {
  currentMode = v;
  try { localStorage.setItem(STORAGE_KEY, v); } catch {}
});

// Setter conveniente para o SettingsPanel (ou outros)
export function setDownloadMode(v: DownloadMode) {
  downloadMode.set(v);
}

// Getter (snapshot) se precisares do valor corrente
export function getDownloadMode(): DownloadMode {
  return currentMode;
}

// ---- API de gravação/download ----

export type SaveResult = 'saved' | 'cancelled';

/** Download direto (sem diálogo) via âncora temporária. */
function downloadViaAnchor(filename: string, blob: Blob): SaveResult {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename || 'download.bin';
  // Algumas webviews pedem que o elemento esteja no DOM:
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
  return 'saved';
}

/** Guarda um Blob segundo o modo atual. */
export async function saveBlob(filename: string, blob: Blob): Promise<SaveResult> {
  // snapshot do modo para esta operação
  const mode = currentMode;
  if (mode === 'dialog') {
    // Usa diálogo nativo via backend (sp.saveBytesPick)
    const ab = await blob.arrayBuffer(); // ArrayBuffer garantido
    const path = await sp.saveBytesPick(
      filename,
      Array.from(new Uint8Array(ab)),
      blob.type || 'application/octet-stream'
    );
    return path ? 'saved' : 'cancelled';
  }
  // Modo auto: download imediato
  return downloadViaAnchor(filename, blob);
}

/** Guarda texto simples. */
export async function saveText(
  filename: string,
  text: string,
  mime = 'text/plain'
): Promise<SaveResult> {
  return saveBlob(filename, new Blob([text], { type: mime }));
}

/** Guarda JSON (pretty=true por defeito). */
export async function saveJSON(
  filename: string,
  obj: any,
  pretty = true
): Promise<SaveResult> {
  const text = pretty ? JSON.stringify(obj, null, 2) : JSON.stringify(obj);
  return saveText(filename, text, 'application/json');
}

/** Guarda bytes (number[] ou base64 string). */
export async function saveBytes(
  filename: string,
  data: number[] | string,
  mime = 'application/octet-stream'
): Promise<SaveResult> {
  const u8 = toU8(data);
  const ab = u8ToArrayBuffer(u8); // garante ArrayBuffer “puro” (evita TS2322)
  return saveBlob(filename, new Blob([ab], { type: mime }));
}

/** Download de anexo (respeita modo). */
export async function saveAttachment(
  list: string,
  id: number,
  fileName: string
): Promise<SaveResult> {
  const mode = currentMode;
  if (mode === 'dialog') {
    const path = await sp.saveAttachmentPick(list, id, fileName);
    return path ? 'saved' : 'cancelled';
  }
  // Auto: baixa os bytes e faz download direto
  const data = await sp.downloadAttachment(list, id, fileName); // number[] | base64 string
  const u8 = toU8(data);
  const ab = u8ToArrayBuffer(u8);
  const blob = new Blob([ab], { type: 'application/octet-stream' });
  return downloadViaAnchor(fileName, blob);
}

/** Download por URL/Server-Relative (respeita modo). */
export async function saveByURL(urlOrPath: string): Promise<SaveResult> {
  const mode = currentMode;
  if (mode === 'dialog') {
    const path = await sp.saveByURLPick(urlOrPath);
    return path ? 'saved' : 'cancelled';
  }
  // Auto: baixa os bytes e faz download direto
  const data = await sp.downloadByURL(urlOrPath); // number[] | base64 string
  const u8 = toU8(data);
  const ab = u8ToArrayBuffer(u8);
  const blob = new Blob([ab], { type: 'application/octet-stream' });
  const suggested = (urlOrPath.split('/').pop() || 'download.bin').split('?')[0];
  return downloadViaAnchor(suggested, blob);
}
