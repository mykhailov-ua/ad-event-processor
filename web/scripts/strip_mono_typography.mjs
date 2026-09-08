import { readFileSync, writeFileSync, readdirSync, statSync } from 'fs';
import { join } from 'path';

const ROOT = join(import.meta.dirname, '..', 'src');
const TOKEN = /\b(font-mono|font-numeric|tabular-nums)\b/g;

function walk(dir, out = []) {
  for (const name of readdirSync(dir)) {
    const path = join(dir, name);
    const st = statSync(path);
    if (st.isDirectory()) {
      walk(path, out);
    } else if (/\.(tsx?)$/.test(name)) {
      out.push(path);
    }
  }
  return out;
}

function cleanText(text) {
  let next = text.replace(TOKEN, '');
  next = next.replace(/className="([^"]*)"/g, (_, classes) => {
    const trimmed = classes.replace(/\s+/g, ' ').trim();
    return trimmed ? `className="${trimmed}"` : '';
  });
  next = next.replace(/cn\(\s*,/g, 'cn(');
  next = next.replace(/,\s*,/g, ',');
  next = next.replace(/\(\s*,/g, '(');
  next = next.replace(/,\s*\)/g, ')');
  return next;
}

let changed = 0;
for (const file of walk(ROOT)) {
  const before = readFileSync(file, 'utf8');
  if (!TOKEN.test(before)) {
    continue;
  }
  const after = cleanText(before);
  if (after !== before) {
    writeFileSync(file, after);
    changed += 1;
  }
}

console.log(`strip_mono_typography: ${changed} files updated`);
