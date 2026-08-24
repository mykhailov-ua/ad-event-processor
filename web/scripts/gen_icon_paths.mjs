#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const webRoot = path.resolve(__dirname, '..');
const phosphorDir = path.join(webRoot, 'node_modules/@phosphor-icons/core/assets/regular');
const mapPath = path.join(__dirname, 'icon_phosphor_map.json');
const outPath = path.join(webRoot, 'src/lib/icon_paths.ts');

const map = JSON.parse(fs.readFileSync(mapPath, 'utf8'));


function parseSvgPrimitives(svg) {
  const inner = svg
    .trim()
    .replace(/<\?xml[^?]*\?>/g, '')
    .replace(/^<svg[^>]*>/, '')
    .replace(/<\/svg>\s*$/, '');

  const tags = [];
  const re = /<(path|circle|rect|line|polyline|polygon|ellipse)([^>]*?)(\/?)>/gi;
  let match;
  while ((match = re.exec(inner)) !== null) {
    const tagName = match[1];
    const attrs = match[2].trim();
    tags.push(`<${tagName}${attrs ? ` ${attrs}` : ''} />`);
  }
  return tags;
}


const paths = {};
const missing = [];

for (const [name, phosphorFile] of Object.entries(map)) {
  const file = path.join(phosphorDir, `${phosphorFile}.svg`);
  if (!fs.existsSync(file)) {
    missing.push(`${name} -> ${phosphorFile}.svg`);
    continue;
  }
  const svg = fs.readFileSync(file, 'utf8');
  const tags = parseSvgPrimitives(svg);
  if (!tags.length) {
    missing.push(`${name} -> ${phosphorFile}.svg (no primitives)`);
    continue;
  }
  paths[name] = tags;
}

if (missing.length) {
  console.error('gen_icon_paths: missing phosphor assets:\n', missing.join('\n'));
  process.exit(1);
}

const names = Object.keys(paths).sort();
const lines = [
  '/** Phosphor regular icon primitives (@phosphor-icons/core, MIT). Generated - do not edit. */',
  '/** Run: npm run icons:gen */',
  'export const ICON_PATHS: Record<string, string[]> = {',
];

for (const name of names) {
  lines.push(`  "${name}": [`);
  for (const tag of paths[name]) {
    lines.push(`    ${JSON.stringify(tag)},`);
  }
  lines.push('  ],');
}
lines.push('};');
lines.push('');

fs.writeFileSync(outPath, lines.join('\n'));
console.log(`gen_icon_paths: wrote ${names.length} icons -> ${path.relative(webRoot, outPath)}`);
