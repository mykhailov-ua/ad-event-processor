#!/usr/bin/env node

import { cpSync, existsSync, mkdirSync, rmSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const WEB_ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const REPO_ROOT = resolve(WEB_ROOT, '..');
const DIST = join(WEB_ROOT, 'dist');
const STUB = join(REPO_ROOT, 'internal/controlplane/admin_static_stub');

const required = [
  'index.html',
  'login.html',
  'src/main.js',
  'src/login.js',
  'src/styles/app.css',
  'src/static/track.js',
];

if (!existsSync(DIST)) {
  console.error(`Error: ${DIST} missing. Run: cd web && npm run build`);
  process.exit(1);
}

for (const rel of required) {
  const path = join(DIST, rel);
  if (!existsSync(path)) {
    console.error(`Error: dist missing ${rel} (required before embed sync)`);
    process.exit(1);
  }
}

rmSync(STUB, { recursive: true, force: true });
mkdirSync(STUB, { recursive: true });
cpSync(DIST, STUB, { recursive: true });

console.log(`embed sync: ${DIST} -> ${STUB}`);
