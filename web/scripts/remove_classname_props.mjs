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
  return content.replace(/^\s*className:\s*[^,\n]+,?\s*$/gm, '');
}

let changed = 0;
for (const file of walk(ROOT)) {
  const original = readFileSync(file, 'utf8');
  const fixed = fixFile(original);
  if (fixed !== original) {
    writeFileSync(file, fixed, 'utf8');
    changed += 1;
  }
}
console.log(`remove_classname_props: updated ${changed} files`);
