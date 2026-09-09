#!/usr/bin/env node

import { readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs';
import { join, extname } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(fileURLToPath(new URL('.', import.meta.url)), '..', 'src');

function walk(dir, out = []) {
  for (const name of readdirSync(dir)) {
    const path = join(dir, name);
    const stat = statSync(path);
    if (stat.isDirectory()) {
      walk(path, out);
      continue;
    }
    if (extname(path) === '.tsx' || extname(path) === '.ts') {
      out.push(path);
    }
  }
  return out;
}

function fixContent(content) {
  let next = content;

  // strip script turned `classNames` into `s` in object literals and destructuring.
  next = next.replace(/\bclassNames\b/g, 'classNames');
  next = next.replace(/(\{[^}]*)\bs\b(?=\s*[:,}])/g, '$1classNames');
  next = next.replace(/(\.\.\.)s\b/g, '...classNames');
  next = next.replace(/\bs=\{/g, 'classNames={');

  // Remove style-related props from JSX (no-op for bare UI).
  next = next.replace(/\s+labelClassName=\{[^}]+\}/g, '');
  next = next.replace(/\s+labelClassName="[^"]*"/g, '');
  next = next.replace(/\s+wrapperClassName=\{[^}]+\}/g, '');
  next = next.replace(/\s+wrapperClassName="[^"]*"/g, '');
  next = next.replace(/\s+breadcrumbsClassName=\{[^}]+\}/g, '');
  next = next.replace(/\s+breadcrumbsClassName="[^"]*"/g, '');
  next = next.replace(/\s+bodyClassName=\{[^}]+\}/g, '');
  next = next.replace(/\s+bodyClassName="[^"]*"/g, '');
  next = next.replace(/\s+headerClassName=\{[^}]+\}/g, '');
  next = next.replace(/\s+headerClassName="[^"]*"/g, '');
  next = next.replace(/\s+valueClassName=\{[^}]+\}/g, '');
  next = next.replace(/\s+valueClassName="[^"]*"/g, '');
  next = next.replace(/\s+hostClassName=\{[^}]+\}/g, '');
  next = next.replace(/\s+hostClassName="[^"]*"/g, '');
  next = next.replace(/\s+tableClassName=\{[^}]+\}/g, '');
  next = next.replace(/\s+tableClassName="[^"]*"/g, '');

  return next;
}

let changed = 0;
for (const file of walk(ROOT)) {
  const original = readFileSync(file, 'utf8');
  const fixed = fixContent(original);
  if (fixed !== original) {
    writeFileSync(file, fixed, 'utf8');
    changed += 1;
  }
}
console.log(`fix_stripped_types: updated ${changed} files`);
