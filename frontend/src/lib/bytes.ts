export function toU8(data: number[] | string): Uint8Array {
  if (typeof data === 'string') {
    const bin = atob(data);
    const u8 = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) u8[i] = bin.charCodeAt(i);
    return u8;
  }
  return Array.isArray(data) ? new Uint8Array(data as number[]) : new Uint8Array();
}

// Garante ArrayBuffer “puro” (não-Shared) compatível com Blob
export function u8ToArrayBuffer(u8: Uint8Array): ArrayBuffer {
  return u8.slice().buffer as ArrayBuffer;
}

export function saveBytesAsFile(
  filename: string,
  data: number[] | string,
  mime = 'application/octet-stream'
) {
  const u8 = toU8(data);
  const ab = u8ToArrayBuffer(u8);
  const blob = new Blob([ab], { type: mime });
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = filename || 'download.bin';
  a.click();
  URL.revokeObjectURL(a.href);
}
