export function normalizeErr(e: any): string {
  const msg = e?.message || String(e || '');
  if (/context (canceled|cancelled)/i.test(msg)) return 'Operação cancelada.';
  if (/deadline exceeded/i.test(msg)) return 'Operação cancelada (timeout).';
  return msg;
}
