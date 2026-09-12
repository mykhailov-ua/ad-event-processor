#!/usr/bin/env node
/**
 * Re-insert className attributes removed by strip_styles.mjs using git diff vs baseline.
 * Baseline: 882abcdf (last commit with intact Tailwind classNames in web/src).
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

function isClassNameLine(line) {
  const t = line.trim();
  return (
    t.startsWith('className=') ||
    t.startsWith('className?:') ||
    t === 'className,' ||
    t.startsWith('className,')
  );
}

function isClassNameContinuation(line) {
  const t = line.trim();
  if (t === '' || t === ')' || t === ')}' || t.endsWith(')}')) {
    return true;
  }
  if (
    t.startsWith("'") ||
    t.startsWith('"') ||
    t.startsWith('`') ||
    t.startsWith('uiSurfaces.') ||
    t.startsWith('admin') ||
    t.startsWith('cn(') ||
    t.includes('adminSpacing') ||
    t.includes('adminTypography') ||
    t.includes('adminKit') ||
    t.startsWith('selected') ||
    t.startsWith(':') ||
    t.startsWith('?') ||
    t === ')' ||
    t === '),' ||
    t.endsWith(',') ||
    t.startsWith('//')
  ) {
    return true;
  }
  return false;
}

function collectRemovedClassBlock(removedLines, startIdx) {
  const block = [];
  let i = startIdx;
  while (i < removedLines.length) {
    const line = removedLines[i];
    block.push(line);
    if (line.trim().endsWith('/>') || line.trim().endsWith('>')) {
      break;
    }
    if (line.trim().endsWith(')}') || line.trim().endsWith('")') || line.trim().endsWith("')")) {
      break;
    }
    i += 1;
    if (
      i < removedLines.length &&
      !isClassNameContinuation(removedLines[i]) &&
      !isClassNameLine(removedLines[i])
    ) {
      if (
        removedLines[i].trim().startsWith('onClick') ||
        removedLines[i].trim().startsWith('type=')
      ) {
        break;
      }
    }
  }
  return block;
}

function parseDiff(diff) {
  const hunks = [];
  const lines = diff.split('\n');
  let i = 0;
  while (i < lines.length) {
    if (!lines[i].startsWith('@@')) {
      i += 1;
      continue;
    }
    const header = lines[i];
    i += 1;
    const removed = [];
    const added = [];
    while (i < lines.length && !lines[i].startsWith('@@')) {
      const line = lines[i];
      if (line.startsWith('-')) {
        removed.push(line.slice(1));
      } else if (line.startsWith('+')) {
        added.push(line.slice(1));
      }
      i += 1;
    }
    hunks.push({ header, removed, added });
  }
  return hunks;
}

function restoreFile(relPath) {
  let diff = '';
  try {
    diff = execSync(`git diff ${BASE} HEAD -- ${relPath}`, {
      cwd: REPO,
      encoding: 'utf8',
      stdio: ['pipe', 'pipe', 'pipe'],
    });
  } catch {
    return false;
  }
  if (!diff.includes('className')) {
    return false;
  }

  const filePath = join(REPO, relPath);
  let content = readFileSync(filePath, 'utf8');
  if (
    !/<[A-Za-z][A-Za-z0-9]*\s[^>]*>\s*$/.test(content) &&
    !/<[A-Za-z][A-Za-z0-9]*\s+>/.test(content)
  ) {
    return false;
  }

  const hunks = parseDiff(diff);
  let changed = false;

  for (const hunk of hunks) {
    const { removed, added } = hunk;
    let r = 0;
    let a = 0;
    while (r < removed.length) {
      if (!isClassNameLine(removed[r])) {
        r += 1;
        a += 1;
        continue;
      }
      const classBlock = collectRemovedClassBlock(removed, r);
      const blockText = classBlock.join('\n');
      r += classBlock.length;

      while (a < added.length && added[a].trim() === '') {
        a += 1;
      }
      if (a >= added.length) {
        continue;
      }

      const anchor = added[a]?.trim() ?? '';
      const insertLines = classBlock.map((line) => line);
      const insertText = insertLines.join('\n');

      const anchorIdx = content.indexOf(added[a]);
      if (anchorIdx === -1) {
        continue;
      }

      const before = content.slice(0, anchorIdx);
      const after = content.slice(anchorIdx);
      if (before.endsWith(insertText) || after.startsWith(insertText)) {
        a += 1;
        continue;
      }

      content = before + insertText + '\n' + after;
      changed = true;
      a += 1;
    }
  }

  content = content.replace(
    /\n\s+\n(\s+(?:onClick|type=|key=|aria-|role=|disabled|value=|id=))/g,
    '\n$1'
  );
  content = content.replace(/(<[A-Za-z][A-Za-z0-9]*)\s+>/g, '$1>');

  if (changed) {
    writeFileSync(filePath, content, 'utf8');
  }
  return changed;
}

let updated = 0;
for (const abs of walk(ROOT)) {
  const rel = relative(REPO, abs);
  if (restoreFile(rel)) {
    updated += 1;
    console.log('restored:', rel);
  }
}
console.log(`restore_classnames_from_git: updated ${updated} files`);
