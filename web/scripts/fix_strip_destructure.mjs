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
    if (extname(path) === '.tsx') {
      out.push(path);
    }
  }
  return out;
}

function fixBrokenDestructuring(content) {
  let next = content;
  next = next.replace(/\{\s*,\s*/g, '{ ');
  next = next.replace(/\(\s*,\s*/g, '(');
  next = next.replace(/,\s*,/g, ',');
  return next;
}

let changed = 0;
for (const file of walk(ROOT)) {
  const original = readFileSync(file, 'utf8');
  const fixed = fixBrokenDestructuring(original);
  if (fixed !== original) {
    writeFileSync(file, fixed, 'utf8');
    changed += 1;
  }
}
console.log(`fix_destructure: updated ${changed} files`);
