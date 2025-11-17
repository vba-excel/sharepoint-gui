// sharepoint-gui/frontend/src/lib/exports.ts
// Utilitários de exportação: JSON, CSV e XLSX (auto-largura, locale CSV, formatos XLSX)
// Respeitam o modo de download através de ./download

import { saveBlob, saveJSON, type SaveResult } from './download';
import { getCsvDelimiter } from './settings';

// Constrói a ordem de colunas a partir da união de chaves de todas as linhas,
// respeitando preferências iniciais (se existirem).
function buildColumns(rows: any[], prefer: string[] = []): string[] {
  const set = new Set<string>();
  for (const r of rows) Object.keys(r ?? {}).forEach(k => set.add(k));
  const all = Array.from(set);
  const prefixed = prefer.filter(p => set.has(p));
  const rest = all.filter(k => !prefixed.includes(k)).sort();
  return [...prefixed, ...rest];
}

function reorderRows(rows: any[], columns: string[]): any[] {
  return (rows || []).map(r => {
    const o: any = {};
    for (const c of columns) if (Object.prototype.hasOwnProperty.call(r ?? {}, c)) o[c] = r[c];
    return o;
  });
}

function csvEscape(val: any, delimiter: string): string {
  if (val === null || val === undefined) return '';
  let s: string;
  if (typeof val === 'object') s = JSON.stringify(val);
  else s = String(val);
  s = s.replace(/\r?\n/g, '\r\n');
  const mustQuote = s.includes('"') || s.includes('\r') || s.includes('\n') || s.includes(delimiter);
  return mustQuote ? `"${s.replace(/"/g, '""')}"` : s;
}

// ---- JSON ----
export async function exportJSON(
  rows: any[],
  filename = 'export.json',
  opts?: { pretty?: boolean; preferColumns?: string[] }
): Promise<SaveResult> {
  if (!rows?.length) return 'cancelled';

  const cols = opts?.preferColumns?.length ? buildColumns(rows, opts.preferColumns) : null;
  const data = cols ? reorderRows(rows, cols) : rows;

  const pretty = opts?.pretty ?? true;
  return saveJSON(filename, data, pretty);
}

// ---- CSV ----
export async function exportCSV(
  rows: any[],
  opts?: {
    filename?: string;
    delimiter?: string;          // default: settings (',' | ';')
    preferColumns?: string[];    // colunas preferidas no início (se existirem)
    withBOM?: boolean;           // Excel-friendly (default true)
    eol?: '\r\n' | '\n';
  }
): Promise<SaveResult> {
  if (!rows?.length) return 'cancelled';

  const filename = opts?.filename ?? 'export.csv';
  const delimiter = opts?.delimiter ?? getCsvDelimiter();
  const eol = opts?.eol ?? '\r\n';
  const cols = buildColumns(rows, opts?.preferColumns ?? ['ID','Id','id','Matricula','Operador','DataHora']);

  const header = cols.join(delimiter);
  const body = rows
    .map(r => cols.map(c => csvEscape(r?.[c], delimiter)).join(delimiter))
    .join(eol);

  const bom = (opts?.withBOM ?? true) ? '\uFEFF' : '';
  const csv = bom + header + eol + body + eol;

  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' });
  return saveBlob(filename, blob);
}

// ---- XLSX ----
function toExcelValue(v: any, autoDates: boolean): any {
  if (v === null || v === undefined) return '';
  if (typeof v === 'number' || typeof v === 'boolean' || v instanceof Date) return v;
  if (typeof v === 'object') return JSON.stringify(v);

  const s = String(v);
  if (autoDates) {
    const isoDate = /^\d{4}-\d{2}-\d{2}$/;
    const isoDT   = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+\-]\d{2}:\d{2})?$/;
    if (isoDT.test(s) || isoDate.test(s)) {
      const d = new Date(s);
      if (!isNaN(d.getTime())) return d;
    }
  }
  if (!isNaN(Number(s)) && s.trim() !== '') return Number(s);
  return s;
}

function autoColWidths(matrix: any[][]): { wch: number }[] {
  const cols = matrix[0]?.length ?? 0;
  const widths: number[] = Array.from({ length: cols }, () => 10);
  for (const row of matrix) {
    row.forEach((cell, i) => {
      const str = cell instanceof Date ? cell.toISOString() : String(cell ?? '');
      widths[i] = Math.max(widths[i], Math.min(60, str.length + 2));
    });
  }
  return widths.map(w => ({ wch: w }));
}

export async function exportXLSX(
  rows: any[],
  opts?: {
    filename?: string;       // default 'export.xlsx'
    sheetName?: string;      // default 'Sheet1'
    preferColumns?: string[];// ordem preferida no início
    autoDates?: boolean;     // tenta converter ISO strings em Date (default true)
    freezeHeader?: boolean;  // congela a primeira linha (default true)
    colFormats?: Record<string,string>; // NOVO: {"DataHora":"yyyy-mm-dd hh:mm"}
  }
): Promise<SaveResult> {
  if (!rows?.length) return 'cancelled';

  const XLSX: typeof import('xlsx') = await import('xlsx');
  const prefer = opts?.preferColumns ?? ['ID','Id','id','Matricula','Operador','DataHora'];
  const cols = buildColumns(rows, prefer);

  const matrix: any[][] = [
    cols,
    ...rows.map(r => cols.map(c => toExcelValue(r?.[c], opts?.autoDates ?? true))),
  ];

  const ws = XLSX.utils.aoa_to_sheet(matrix);
  (ws as any)['!cols'] = autoColWidths(matrix);

  if (opts?.freezeHeader ?? true) {
    (ws as any)['!freeze'] = { xSplit: 0, ySplit: 1, topLeftCell: 'A2', activePane: 'bottomLeft', state: 'frozen' };
  }

  // NOVO: aplicar formatos por coluna (cell.z)
  if (opts?.colFormats && Object.keys(opts.colFormats).length) {
    const colIndex: Record<string, number> = {};
    cols.forEach((name, i) => { colIndex[name] = i; });

    for (const [colName, fmt] of Object.entries(opts.colFormats)) {
      const c = colIndex[colName];
      if (c == null) continue;
      // Percorre linhas de dados (r >= 1; r=0 é header)
      for (let r = 1; r < matrix.length; r++) {
        const addr = XLSX.utils.encode_cell({ r, c });
        const cell = (ws as any)[addr];
        if (cell) cell.z = fmt; // aplica formato
      }
    }
  }

  const wb = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(wb, ws, (opts?.sheetName ?? 'Sheet1').slice(0, 31));

  const out = XLSX.write(wb, { bookType: 'xlsx', type: 'array' }); // ArrayBuffer
  const blob = new Blob([out], {
    type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  });
  return saveBlob(opts?.filename ?? 'export.xlsx', blob);
}
