import { writable } from 'svelte/store';

export type Theme = 'light' | 'dark';

function detectInitial(): Theme {
  const saved = localStorage.getItem('theme');
  if (saved === 'light' || saved === 'dark') return saved;
  return window.matchMedia?.('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

export const theme = writable<Theme>(detectInitial());

theme.subscribe((t) => {
  const root = document.documentElement;
  root.dataset.theme = t;
  // ajuda o browser a escolher cores de UI nativas (scrollbar, inputs)
  root.style.colorScheme = t;
  localStorage.setItem('theme', t);
});
