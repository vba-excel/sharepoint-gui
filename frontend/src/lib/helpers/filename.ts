// helpers/filename.ts
export function stamped(name: string, ext: string) {
  const pad = (n: number) => n.toString().padStart(2, '0');
  const d = new Date();
  const ts = `${d.getFullYear()}${pad(d.getMonth()+1)}${pad(d.getDate())}-${pad(d.getHours())}${pad(d.getMinutes())}${pad(d.getSeconds())}`;
  const safe = (name || 'export').replace(/[^\w.-]+/g, '_');
  return `${safe}-${ts}.${ext}`;
}