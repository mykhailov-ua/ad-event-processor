#!/usr/bin/env node
import { readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs';
import { join, extname } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(fileURLToPath(new URL('.', import.meta.url)), '..', 'src');

function walk(dir, out = []) {
  for (const name of readdirSync(dir)) {
    const path = join(dir, name);
    if (statSync(path).isDirectory()) walk(path, out);
    else if (extname(path) === '.tsx') out.push(path);
  }
  return out;
}

function ensureImports(content) {
  let next = content;
  const needsTypography =
    next.includes('adminTypography') &&
    !next.includes("from '@/lib/admin_kit'") &&
    !next.includes('from "@/lib/admin_kit"');
  const needsCn =
    next.includes(' cn(') &&
    !next.includes("from '@/lib/utils'") &&
    !next.includes('from "@/lib/utils"');
  if (!needsTypography && !needsCn) {
    return next;
  }
  const importBlockEnd = next.indexOf('\n\n');
  const insertAt = importBlockEnd === -1 ? 0 : importBlockEnd + 2;
  const lines = [];
  if (needsTypography) lines.push("import { adminTypography } from '@/lib/admin_kit';");
  if (needsCn) lines.push("import { cn } from '@/lib/utils';");
  return next.slice(0, insertAt) + lines.join('\n') + '\n' + next.slice(insertAt);
}

const subs = [
  [/\sclassName="[^"]*"\s+className="/g, ' className="'],
  [/className="text-base font-semibold"\s*>/g, 'className={adminTypography.sectionTitle}>'],
  [/className="text-base font-semibold"\s+/g, 'className={adminTypography.sectionTitle} '],
  [/className="text-base">/g, 'className={adminTypography.sectionTitle}>'],
  [
    /className="m-0 text-sm font-semibold text-foreground"\s*>/g,
    'className={cn("m-0", adminTypography.sectionTitle)}>',
  ],
  [
    /className="text-sm font-semibold text-foreground"\s*>/g,
    'className={adminTypography.sectionTitle}>',
  ],
  [
    /className="text-sm font-semibold text-foreground"\s+/g,
    'className={adminTypography.sectionTitle} ',
  ],
  [/className="m-0 text-sm font-medium"\s*>/g, 'className={cn("m-0", adminTypography.label)}>'],
  [/className="m-0 text-sm font-medium"\s+/g, 'className={cn("m-0", adminTypography.label)} '],
  [
    /className="text-sm font-medium text-destructive"\s*>/g,
    'className={cn(adminTypography.label, "text-destructive")}>',
  ],
  [/className="text-sm font-medium"\s*>/g, 'className={adminTypography.label}>'],
  [/className="text-sm font-medium"\s+/g, 'className={adminTypography.label} '],
  [
    /className="m-0 text-sm text-muted-foreground"\s*>/g,
    'className={cn("m-0", adminTypography.bodyMuted)}>',
  ],
  [
    /className="m-0 text-sm text-muted-foreground"\s+/g,
    'className={cn("m-0", adminTypography.bodyMuted)} ',
  ],
  [/className="text-sm text-muted-foreground"/g, 'className={adminTypography.bodyMuted}'],
  [/className="shrink-0 text-sm"/g, 'className={cn("shrink-0", adminTypography.body)}'],
  [
    /className="ui-surface grid gap-1 p-3 text-sm"/g,
    'className={cn("ui-surface grid gap-1 p-3", adminTypography.body)}',
  ],
  [
    /className="font-mono text-xs text-muted-foreground"/g,
    'className={cn(adminTypography.monoData, "text-muted-foreground")}',
  ],
  [
    /className="break-all font-mono text-xs text-muted-foreground"/g,
    'className={cn("break-all", adminTypography.monoData, "text-muted-foreground")}',
  ],
  [
    /className="break-all font-mono text-xs"/g,
    'className={cn("break-all", adminTypography.monoData)}',
  ],
  [/className="font-mono text-xs"/g, 'className={adminTypography.monoData}'],
  [
    /className="text-xs text-muted-foreground font-mono"/g,
    'className={cn(adminTypography.monoData, "text-muted-foreground")}',
  ],
  [/className="text-xs text-muted-foreground"/g, 'className={adminTypography.captionPlain}'],
  [
    /className="text-xs text-amber-700 dark:text-amber-400"/g,
    'className={cn(adminTypography.captionPlain, "text-amber-700 dark:text-amber-400")}',
  ],
  [
    /className="min-h-32 font-mono text-xs"/g,
    'className={cn("min-h-32", adminTypography.monoData)}',
  ],
  [
    /className="min-w-\[6rem\] font-mono text-xs"/g,
    'className={cn("min-w-[6rem]", adminTypography.monoData)}',
  ],
  [
    /className=\{cn\('flex flex-col gap-1 text-sm font-medium text-foreground', className\)\}/g,
    "className={cn('flex flex-col gap-1', adminTypography.label, className)}",
  ],
  [
    /className="text-\[13px\] font-semibold leading-\[18px\]"/g,
    'className={adminTypography.sectionTitle}',
  ],
];

let changed = 0;
for (const dir of ['domains', 'pages', 'shell'].map((p) => join(ROOT, p))) {
  for (const file of walk(dir)) {
    let content = readFileSync(file, 'utf8');
    let next = content;
    for (const [re, rep] of subs) next = next.replace(re, rep);
    next = ensureImports(next);
    if (next !== content) {
      writeFileSync(file, next, 'utf8');
      changed += 1;
      console.log(file.replace(ROOT + '/', 'web/src/'));
    }
  }
}
console.log(`fix_typography_slop: ${changed} files`);
