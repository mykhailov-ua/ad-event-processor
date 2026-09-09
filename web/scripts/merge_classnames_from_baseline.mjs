#!/usr/bin/env node
/**
 * Merge className attributes from git baseline (882abcdf) into current TSX when strip_styles removed them.
 */
import { execSync } from 'node:child_process';
import { readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs';
import { join, extname, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const BASE = '882abcdf';
const ROOT = join(fileURLToPath(new URL('.', import.meta.url)), '..', 'src');
const REPO = join(fileURLToPath(new URL('.', import.meta.url)), '..', '..');

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

function isIdentStart(ch) {
  return /[A-Za-z_$]/.test(ch);
}

function isIdentPart(ch) {
  return /[A-Za-z0-9_$:-]/.test(ch);
}

/** Parse JSX opening/self-closing tags in source order. */
function parseOpeningTags(content) {
  const tags = [];
  let i = 0;
  while (i < content.length) {
    if (content[i] !== '<') {
      i += 1;
      continue;
    }
    if (content[i + 1] === '!' || content[i + 1] === '?') {
      i += 1;
      continue;
    }
    if (content[i + 1] === '/') {
      i += 1;
      continue;
    }

    const start = i;
    i += 1;
    let name = '';
    while (i < content.length && isIdentPart(content[i])) {
      name += content[i];
      i += 1;
    }
    if (!name) {
      continue;
    }

    while (i < content.length && /\s/.test(content[i])) {
      i += 1;
    }

    let className = null;
    let classStart = -1;
    let classEnd = -1;
    const attrsStart = i;

    while (i < content.length) {
      if (content[i] === '>' && content[i - 1] !== '=') {
        break;
      }
      if (content[i] === '/' && content[i + 1] === '>') {
        break;
      }

      if (content.slice(i, i + 9) === 'className') {
        classStart = i;
        i += 9;
        while (i < content.length && /\s/.test(content[i])) {
          i += 1;
        }
        if (content[i] === '=') {
          i += 1;
          while (i < content.length && /\s/.test(content[i])) {
            i += 1;
          }
          const valueEnd = readJsxAttributeValueEnd(content, i);
          className = content.slice(i, valueEnd).trim();
          classEnd = valueEnd;
          i = valueEnd;
          continue;
        }
      }

      i = skipJsxAttribute(content, i);
    }

    const selfClosing = content[i] === '/' && content[i + 1] === '>';
    const end = selfClosing ? i + 2 : content.indexOf('>', i) + 1;
    if (end <= 0) {
      break;
    }

    tags.push({
      name,
      start,
      end,
      attrsStart,
      className,
      classStart,
      classEnd,
      selfClosing,
      snippet: content.slice(start, Math.min(start + 80, end)),
    });
    i = end;
  }
  return tags;
}

function readJsxAttributeValueEnd(content, start) {
  let i = start;
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
        i = skipQuoted(content, i, ch) + 1;
        continue;
      }
      i += 1;
    }
    return i;
  }
  if (content[i] === '"' || content[i] === "'") {
    return skipQuoted(content, i, content[i]) + 1;
  }
  while (i < content.length && !/[\s/>]/.test(content[i])) {
    i += 1;
  }
  return i;
}

function skipQuoted(content, start, quote) {
  let i = start + 1;
  while (i < content.length) {
    if (content[i] === '\\') {
      i += 2;
      continue;
    }
    if (content[i] === quote) {
      return i;
    }
    i += 1;
  }
  return i;
}

function skipJsxAttribute(content, start) {
  let i = start;
  if (!isIdentStart(content[i])) {
    return start + 1;
  }
  while (i < content.length && isIdentPart(content[i])) {
    i += 1;
  }
  while (i < content.length && /\s/.test(content[i])) {
    i += 1;
  }
  if (content[i] === '=') {
    i += 1;
    while (i < content.length && /\s/.test(content[i])) {
      i += 1;
    }
    return readJsxAttributeValueEnd(content, i);
  }
  return i;
}

function tagHasClassName(content, tag) {
  const slice = content.slice(tag.start, tag.end);
  return slice.includes('className');
}

function insertClassName(content, tag, classNameExpr) {
  const insertAt = tag.attrsStart;
  const before = content.slice(0, insertAt);
  const after = content.slice(insertAt);
  const spacer = before.endsWith(' ') || before.endsWith('\n') || before.endsWith('\t') ? '' : ' ';
  return `${before}${spacer}className=${classNameExpr} ${after}`;
}

function mergeFile(relPath) {
  const abs = join(REPO, relPath);
  const current = readFileSync(abs, 'utf8');
  if (!/<[A-Za-z][A-Za-z0-9]*(?:\s[^>]*)?\s+>/.test(current)) {
    return false;
  }

  let baseline = '';
  try {
    baseline = execSync(`git show ${BASE}:${relPath}`, {
      cwd: REPO,
      encoding: 'utf8',
      stdio: ['pipe', 'pipe', 'pipe'],
    });
  } catch {
    return false;
  }

  const oldTags = parseOpeningTags(baseline);
  const curTags = parseOpeningTags(current);
  if (oldTags.length === 0) {
    return false;
  }

  let next = current;
  let offset = 0;
  let merged = 0;
  const limit = Math.min(oldTags.length, curTags.length);

  for (let idx = 0; idx < limit; idx += 1) {
    const oldTag = oldTags[idx];
    const curTag = curTags[idx];
    if (oldTag.name !== curTag.name) {
      continue;
    }
    if (!oldTag.className || tagHasClassName(next, curTag)) {
      continue;
    }

    const adjustedTag = {
      ...curTag,
      start: curTag.start + offset,
      end: curTag.end + offset,
      attrsStart: curTag.attrsStart + offset,
    };

    next = insertClassName(next, adjustedTag, oldTag.className);
    offset += `className=${oldTag.className} `.length;
    merged += 1;
  }

  next = next.replace(/(<[A-Za-z][A-Za-z0-9]*)\s+>/g, '$1>');

  if (merged > 0 && next !== current) {
    writeFileSync(abs, next, 'utf8');
    console.log(`merged ${merged} classNames: ${relPath}`);
    return true;
  }
  return false;
}

let updated = 0;
for (const abs of walk(ROOT)) {
  const rel = relative(REPO, abs);
  if (mergeFile(rel)) {
    updated += 1;
  }
}
console.log(`merge_classnames_from_baseline: updated ${updated} files`);
