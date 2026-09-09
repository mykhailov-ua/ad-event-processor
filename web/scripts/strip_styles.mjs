#!/usr/bin/env node

import { readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs';
import { join, extname } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(fileURLToPath(new URL('.', import.meta.url)), '..', 'src');
const ATTR = 'className';

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

function skipAttributeValue(content, start) {
  let i = start;
  if (content[i] !== '=') {
    return start;
  }
  i += 1;
  while (content[i] === ' ') {
    i += 1;
  }
  if (content[i] === '"') {
    i += 1;
    while (i < content.length && content[i] !== '"') {
      if (content[i] === '\\') {
        i += 2;
        continue;
      }
      i += 1;
    }
    return i + 1;
  }
  if (content[i] === "'") {
    i += 1;
    while (i < content.length && content[i] !== "'") {
      if (content[i] === '\\') {
        i += 2;
        continue;
      }
      i += 1;
    }
    return i + 1;
  }
  if (content[i] === '{') {
    let depth = 0;
    while (i < content.length) {
      const ch = content[i];
      if (ch === '{') {
        depth += 1;
      } else if (ch === '}') {
        depth -= 1;
        if (depth === 0) {
          return i + 1;
        }
      } else if (ch === '"' || ch === "'" || ch === '`') {
        const quote = ch;
        i += 1;
        while (i < content.length && content[i] !== quote) {
          if (content[i] === '\\') {
            i += 2;
            continue;
          }
          i += 1;
        }
      }
      i += 1;
    }
  }
  return i;
}

function stripAttributeName(content, attrName) {
  let result = '';
  let i = 0;
  while (i < content.length) {
    const idx = content.indexOf(attrName, i);
    if (idx === -1) {
      result += content.slice(i);
      break;
    }
    const prev = content[idx - 1];
    if (prev && /[A-Za-z0-9_$.]/.test(prev)) {
      result += content.slice(i, idx + attrName.length);
      i = idx + attrName.length;
      continue;
    }
    const next = content[idx + attrName.length];
    if (next === '?' || next === ':') {
      result += content.slice(i, idx + attrName.length);
      i = idx + attrName.length;
      continue;
    }
    result += content.slice(i, idx);
    const end = skipAttributeValue(content, idx + attrName.length);
    i = end;
  }
  return result;
}

function stripFile(path) {
  let content = readFileSync(path, 'utf8');
  const original = content;
  content = stripAttributeName(content, ATTR);
  content = stripAttributeName(content, 'style');
  if (content !== original) {
    writeFileSync(path, content, 'utf8');
    return true;
  }
  return false;
}

const files = walk(ROOT);
let changed = 0;
for (const file of files) {
  if (stripFile(file)) {
    changed += 1;
  }
}
console.log(`strip_styles: updated ${changed} of ${files.length} tsx files`);
