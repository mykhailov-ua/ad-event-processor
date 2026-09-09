#!/usr/bin/env node
import { readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs';
import { join, extname } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(fileURLToPath(new URL('.', import.meta.url)), '..', 'src');

function walk(dir, out = []) {
  for (const name of readdirSync(dir)) {
    const path = join(dir, name);
    if (statSync(path).isDirectory()) {
      walk(path, out);
    } else if (extname(path) === '.tsx') {
      out.push(path);
    }
  }
  return out;
}

let changed = 0;
for (const file of walk(ROOT)) {
  const content = readFileSync(file, 'utf8');
  const next = content.replace(/<([A-Za-z][A-Za-z0-9]*)className=/g, '<$1 className=');
  if (next !== content) {
    writeFileSync(file, next, 'utf8');
    changed += 1;
  }
}
console.log(`fix_tag_classname_merge: updated ${changed} files`);
