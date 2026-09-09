#!/usr/bin/env node

import { readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs';
import { join, extname } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(fileURLToPath(new URL('.', import.meta.url)), '..', 'src');

const STYLE_PROPS = [
  'labelClassName',
  'wrapperClassName',
  'breadcrumbsClassName',
  'bodyClassName',
  'headerClassName',
  'valueClassName',
  'hostClassName',
  'tableClassName',
];

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

function fixContent(content) {
  let next = content;
  for (const prop of STYLE_PROPS) {
    next = next.replace(new RegExp(`\\s+${prop}=\\{[^}]+\\}`, 'g'), '');
    next = next.replace(new RegExp(`\\s+${prop}="[^"]*"`, 'g'), '');
  }
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
console.log(`remove_style_props: updated ${changed} files`);
