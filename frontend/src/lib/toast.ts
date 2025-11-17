import { writable } from 'svelte/store';

export type ToastKind = 'success' | 'error' | 'info';
export type Toast = { id: number; text: string; kind: ToastKind };

function createToasts() {
  const { subscribe, update } = writable<Toast[]>([]);
  let nextId = 0;

  function push(text: string, kind: ToastKind = 'info', ms = 2500) {
    const id = ++nextId;
    update(ts => [...ts, { id, text, kind }]);
    setTimeout(() => update(ts => ts.filter(t => t.id !== id)), ms);
  }

  return { subscribe, push };
}

export const toasts = createToasts();
