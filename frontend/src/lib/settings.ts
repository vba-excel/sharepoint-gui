// sharepoint-gui/frontend/src/lib/settings.ts
import { writable, derived, get } from 'svelte/store';

// ──────────────────────────────────────────────────────────────
// Attachments: accept (extensões/MIME)
// ──────────────────────────────────────────────────────────────
export const attachmentsAccept = writable<string>(
  localStorage.getItem('sp_att_accept') ?? ''
);
attachmentsAccept.subscribe((v) => {
  try { localStorage.setItem('sp_att_accept', v ?? ''); } catch {}
});

// ──────────────────────────────────────────────────────────────
// Attachments: limite de upload (MB) + bytes derivado
// ──────────────────────────────────────────────────────────────
const initialMB = Number(localStorage.getItem('sp_att_maxmb') ?? '50') || 50;
export const maxUploadMBStore = writable<number>(initialMB);
maxUploadMBStore.subscribe((v) => {
  const val = v > 0 ? v : 50;
  try { localStorage.setItem('sp_att_maxmb', String(val)); } catch {}
});

// Útil para validações/UX em bytes no UI (drag&drop, input[file], etc.)
export const maxUploadBytesStore = derived(maxUploadMBStore, (mb) =>
  Math.max(1, mb) * 1024 * 1024
);

// ──────────────────────────────────────────────────────────────
// CSV mode (para exportações). Delimitador derivado.
// ──────────────────────────────────────────────────────────────
export type CsvMode = 'standard' | 'pt';

const initialCsv: CsvMode =
  (localStorage.getItem('sp_csv_mode') === 'pt') ? 'pt' : 'standard';

export const csvModeStore = writable<CsvMode>(initialCsv);
csvModeStore.subscribe((v) => {
  try { localStorage.setItem('sp_csv_mode', v); } catch {}
});

// Delimitador derivado (se preferires reatividade no UI)
export const csvDelimiterStore = derived(csvModeStore, (m) => m === 'pt' ? ';' : ',');

// ──────────────────────────────────────────────────────────────
// Helpers (compatíveis com código existente)
// ──────────────────────────────────────────────────────────────
export function getCsvMode(): CsvMode { return get(csvModeStore); }
export function getCsvDelimiter(): string { return get(csvModeStore) === 'pt' ? ';' : ','; }
export function getAttachmentsAccept(): string { return get(attachmentsAccept); }
export function getMaxUploadMB(): number { return get(maxUploadMBStore); }

// Nota: o Download Mode passou para lib/download.ts (store própria lá).
