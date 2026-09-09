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

function fixFile(content) {
  let next = content;
  next = next.replace(/^\s*\?: string;\s*\n/gm, '');
  next = next.replace(/^\s*wrapperClassName\?: string;\s*\n/gm, '');
  next = next.replace(/^\s*labelClassName\?: string;\s*\n/gm, '');
  next = next.replace(/^\s*breadcrumbsClassName\?: string;\s*\n/gm, '');
  next = next.replace(/export const (\w+) = '[^']*';/g, "export const $1 = '';");
  next = next.replace(/export const (\w+) =\s*\n\s*'[^']*';/g, "export const $1 = '';");
  return next;
}

let changed = 0;
for (const file of walk(ROOT)) {
  if (file.endsWith('.test.ts')) {
    continue;
  }
  const original = readFileSync(file, 'utf8');
  const fixed = fixFile(original);
  if (fixed !== original) {
    writeFileSync(file, fixed, 'utf8');
    changed += 1;
  }
}
console.log(`fix_type_classname: updated ${changed} files`);
