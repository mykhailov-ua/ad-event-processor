#!/usr/bin/env node
/** Replace banned typography tokens in domains/pages with adminTypography roles. */
import { readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs';
import { join, extname } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(fileURLToPath(new URL('.', import.meta.url)), '..', 'src');
const TARGETS = ['domains', 'pages'].map((part) => join(ROOT, part));

const REPLACEMENTS = [
  ['text-sm text-muted-foreground', '${adminTypography.bodyMuted}'],
  ['text-muted-foreground text-sm', '${adminTypography.bodyMuted}'],
  ['m-0 text-sm font-semibold', '${adminTypography.sectionTitle}'],
  ['text-sm font-semibold', '${adminTypography.sectionTitle}'],
  ['text-base font-semibold', '${adminTypography.sectionTitle}'],
  ['text-base', '${adminTypography.sectionTitle}'],
  [' text-sm', ' ${adminTypography.body}'],
];

function walk(dir, out = []) {
  if (!statSync(dir, { throwIfNoEntry: false })?.isDirectory()) {
    return out;
  }
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

function ensureTypographyImport(content) {
  if (content.includes('adminTypography')) {
    return content;
  }
  const importLine = "import { adminTypography } from '@/lib/admin_kit';\n";
  const lastImport = content.lastIndexOf('\nimport ');
  if (lastImport === -1) {
    return importLine + content;
  }
  const end = content.indexOf('\n', lastImport + 1);
  return content.slice(0, end + 1) + importLine + content.slice(end + 1);
}

function toTemplateLiteral(className) {
  if (className.includes('${adminTypography.')) {
    return `{${JSON.stringify(className)
      .slice(1, -1)
      .replace(/\\"/g, '"')
      .replace(/adminTypography\./g, '${adminTypography.')}}`;
  }
  return `"${className}"`;
}

let changed = 0;
for (const base of TARGETS) {
  for (const file of walk(base)) {
    let content = readFileSync(file, 'utf8');
    let next = content;
    let touched = false;
    for (const [from, to] of REPLACEMENTS) {
      if (next.includes(from)) {
        next = next.split(from).join(to);
        touched = true;
      }
    }
    if (!touched) {
      continue;
    }
    next = ensureTypographyImport(next);
    next = next.replace(/className="([^"]*\$\{adminTypography[^"]*)"/g, (_, value) => {
      const expr = value.replace(/\$\{adminTypography\.(\w+)\}/g, '${adminTypography.$1}');
      return `className={\`${expr}\`}`;
    });
    if (next !== content) {
      writeFileSync(file, next, 'utf8');
      changed += 1;
      console.log('typography:', file.replace(ROOT + '/', 'web/src/'));
    }
  }
}
console.log(`migrate_domain_typography: updated ${changed} files`);
