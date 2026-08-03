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

const ts = Date.now();
const FONT_LINKS = `    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@fontsource/geist-sans@5.2.5/400.css" />
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@fontsource/geist-sans@5.2.5/600.css" />
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@fontsource/geist-mono@5.2.5/400.css" />
    <link rel="stylesheet" href="/src/styles/tokens.css?v=${ts}" />
    <link rel="stylesheet" href="/src/styles/system.css?v=${ts}" />
`;

const INDEX_HTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>BidShard Admin</title>
${FONT_LINKS}    <link rel="stylesheet" href="/src/styles/main.css?v=${ts}" />
    <link rel="stylesheet" href="/src/styles/a11y.css?v=${ts}" />
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
${FONT_LINKS}    <link rel="stylesheet" href="/src/styles/main.css?v=${ts}" />
    <link rel="stylesheet" href="/src/styles/a11y.css?v=${ts}" />
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
