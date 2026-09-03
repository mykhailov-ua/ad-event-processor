#!/usr/bin/env node
/** Strip var(--admin-*) and remaining admin-* tokens; map to Tailwind zinc utilities. */
import { readFileSync, writeFileSync, readdirSync, statSync } from 'node:fs';
import { join, extname } from 'node:path';
import { fileURLToPath } from 'node:url';

const SRC = join(fileURLToPath(new URL('.', import.meta.url)), '..', 'src');

const REPLACEMENTS = [
  ['rounded-[var(--admin-radius-sm)]', 'rounded-sm'],
  ['rounded-[var(--admin-radius)]', 'rounded-md'],
  ['border-[var(--admin-border)]', 'border-zinc-200 dark:border-zinc-800'],
  ['border-[var(--admin-border-light)]', 'border-zinc-100 dark:border-zinc-800'],
  ['border-[var(--admin-border-strong)]', 'border-zinc-300 dark:border-zinc-700'],
  ['bg-[var(--admin-surface-1)]', 'bg-white dark:bg-zinc-950'],
  ['bg-[var(--admin-surface-2)]', 'bg-zinc-50 dark:bg-zinc-900'],
  ['bg-[var(--admin-surface-3)]', 'bg-zinc-100 dark:bg-zinc-800'],
  ['bg-[var(--admin-surface-sunken)]', 'bg-zinc-50 dark:bg-zinc-900'],
  ['bg-[var(--admin-input-bg)]', 'bg-white dark:bg-zinc-950'],
  ['bg-[var(--admin-accent)]', 'bg-zinc-100 dark:bg-zinc-800'],
  ['bg-[var(--admin-item-highlight)]', 'bg-zinc-100 dark:bg-zinc-800'],
  ['bg-[var(--admin-brand-soft)]', 'bg-blue-50 dark:bg-blue-950/40'],
  ['bg-[var(--admin-brand)]', 'bg-blue-600'],
  ['bg-[var(--admin-btn-hover-bg)]', 'bg-zinc-100 dark:bg-zinc-800'],
  ['text-[var(--admin-fg-emphasis)]', 'font-semibold text-zinc-900 dark:text-zinc-50'],
  ['text-[var(--admin-fg-secondary)]', 'text-zinc-600 dark:text-zinc-400'],
  ['text-[var(--admin-fg)]', 'text-zinc-900 dark:text-zinc-100'],
  ['text-[var(--admin-muted)]', 'text-zinc-500 dark:text-zinc-400'],
  ['text-[var(--admin-brand)]', 'text-blue-600 dark:text-blue-400'],
  ['border-[var(--admin-brand)]', 'border-blue-600'],
  ['h-[var(--admin-control-height)]', 'h-8'],
  ['min-h-[var(--admin-control-height)]', 'min-h-8'],
  ['focus-visible:ring-[var(--admin-brand)]', 'focus-visible:ring-blue-600'],
  ['ring-[var(--admin-brand)]', 'ring-blue-600'],
  ['divide-[var(--admin-border)]', 'divide-zinc-200 dark:divide-zinc-800'],
  ['admin-table-th-inner', 'flex w-full items-center gap-1'],
  ['admin-header-search-input', ''],
  [
    'admin-header-search-trigger',
    'flex h-8 w-full items-center justify-between gap-2 rounded-md border border-zinc-200 bg-white px-3 text-sm text-zinc-500 hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-950 dark:hover:bg-zinc-900',
  ],
  ['admin-header-search-placeholder', 'truncate text-left'],
  [
    'admin-header-search-kbd',
    'hidden rounded border border-zinc-200 px-1.5 py-0.5 text-[10px] font-medium text-zinc-500 sm:inline dark:border-zinc-700',
  ],
  ['hsl(var(--primary))', '#3b82f6'],
  ['hsl(var(--muted-foreground) / 0.65)', 'rgb(113 113 122 / 0.65)'],
];

function walk(dir, files = []) {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) {
      if (name === 'styles') continue;
      walk(p, files);
    } else if (['.ts', '.tsx'].includes(extname(name))) {
      files.push(p);
    }
  }
  return files;
}

for (const file of walk(SRC)) {
  let content = readFileSync(file, 'utf8');
  let changed = false;
  for (const [from, to] of REPLACEMENTS) {
    if (content.includes(from)) {
      content = content.split(from).join(to);
      changed = true;
    }
  }
  if (changed) {
    writeFileSync(file, content, 'utf8');
  }
}

console.log('strip_admin_vars: done');
