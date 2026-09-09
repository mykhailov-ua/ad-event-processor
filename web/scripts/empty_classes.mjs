#!/usr/bin/env node

import { readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(fileURLToPath(new URL('.', import.meta.url)), '..', 'src');

const TARGETS = [
  'shell/shell_chrome.ts',
  'shell/filter_panel_classes.ts',
  'shell/searchable_filter_select_classes.ts',
  'lib/campaign_picker_classes.ts',
  'domains/campaigns/list/campaign_list_classes.ts',
  'domains/customers/customer_detail_classes.ts',
  'domains/dashboards/dashboard_classes.ts',
  'domains/dashboards/dashboard_preferences_classes.ts',
  'domains/dashboards/dashboard_metrics.ts',
];

function emptyExportedStrings(content) {
  let next = content.replace(
    /export const (\w+)(?::[^=]+)?\s*=\s*(?:cn\([\s\S]*?\)|`[\s\S]*?`|'[\s\S]*?'|"[\s\S]*?");/g,
    "export const $1 = '';"
  );
  next = next.replace(
    /(\w+):\s*(?:cn\([\s\S]*?\)|`[\s\S]*?`|'[\s\S]*?'|"[\s\S]*?")/g,
    "$1: ''"
  );
  return next;
}

function walkClasses(dir, out = []) {
  for (const name of readdirSync(dir)) {
    const path = join(dir, name);
    const stat = statSync(path);
    if (stat.isDirectory()) {
      walkClasses(path, out);
      continue;
    }
    if (name.includes('classes') && name.endsWith('.ts') && !name.endsWith('.test.ts')) {
      out.push(path);
    }
  }
  return out;
}

const files = new Set([...TARGETS.map((rel) => join(ROOT, rel)), ...walkClasses(ROOT)]);
let changed = 0;
for (const file of files) {
  if (!statSync(file).isFile()) {
    continue;
  }
  const original = readFileSync(file, 'utf8');
  const next = emptyExportedStrings(original);
  if (next !== original) {
    writeFileSync(file, next, 'utf8');
    changed += 1;
  }
}
console.log(`empty_classes: updated ${changed} files`);
