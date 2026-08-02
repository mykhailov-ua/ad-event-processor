#!/usr/bin/env node
/**
 * Copy native ESM sources into dist/ and write HTML entry shells.
 * No bundler; no npm runtime dependencies.
 */
import { cpSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const DIST = join(ROOT, 'dist');
const SRC = join(ROOT, 'src');

const INDEX_HTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>BidShard Admin</title>
    <link rel="stylesheet" href="/src/styles/a11y.css" />
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.js"></script>
  </body>
</html>
`;

const LOGIN_HTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Sign in — BidShard Admin</title>
    <link rel="stylesheet" href="/src/styles/a11y.css" />
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/login.js"></script>
  </body>
</html>
`;

rmSync(DIST, { recursive: true, force: true });
mkdirSync(DIST, { recursive: true });
cpSync(SRC, join(DIST, 'src'), { recursive: true });
writeFileSync(join(DIST, 'index.html'), INDEX_HTML, 'utf8');
writeFileSync(join(DIST, 'login.html'), LOGIN_HTML, 'utf8');

console.log('dist: copied src/ and wrote index.html, login.html');
