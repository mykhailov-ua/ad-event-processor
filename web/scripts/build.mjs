#!/usr/bin/env node
/**
 * Bundle admin UI with esbuild into web/dist for go:embed.
 * Entry shells keep /src/main.js and /src/login.js paths (controlplane static routes).
 */
import * as esbuild from 'esbuild';
import { cpSync, mkdirSync, rmSync, writeFileSync, existsSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const DIST = join(ROOT, 'dist');
const SRC = join(ROOT, 'src');

// Prefer local web/node_modules; fall back to repo root if present.
const require = createRequire(import.meta.url);
try {
  require.resolve('esbuild');
} catch {
  console.error('esbuild missing. Run: cd web && npm ci');
  process.exit(1);
}

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
    <script type="module" src="/src/main.js?v=${ts}"></script>
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
    <script type="module" src="/src/login.js?v=${ts}"></script>
  </body>
</html>
`;

rmSync(DIST, { recursive: true, force: true });
mkdirSync(join(DIST, 'src'), { recursive: true });

// Sources are *.worker.ts; esbuild entryNames keep basename → dist/src/workers/*.worker.js
const workerEntries = [
  join(SRC, 'workers', 'parse_json.worker.ts'),
  join(SRC, 'workers', 'report_aggregate.worker.ts'),
];

await esbuild.build({
  absWorkingDir: ROOT,
  entryPoints: [join(SRC, 'main.tsx'), join(SRC, 'login.tsx'), ...workerEntries],
  bundle: true,
  splitting: true,
  format: 'esm',
  platform: 'browser',
  target: ['es2022'],
  jsx: 'automatic',
  jsxImportSource: 'react',
  outdir: join(DIST, 'src'),
  outbase: SRC,
  entryNames: '[dir]/[name]',
  chunkNames: 'chunks/[name]-[hash]',
  assetNames: 'assets/[name]-[hash]',
  sourcemap: true,
  minify: true,
  logLevel: 'info',
  loader: {
    '.ts': 'ts',
    '.tsx': 'tsx',
    '.js': 'js',
  },
});

cpSync(join(SRC, 'styles'), join(DIST, 'src', 'styles'), { recursive: true });
if (existsSync(join(SRC, 'static'))) {
  cpSync(join(SRC, 'static'), join(DIST, 'src', 'static'), { recursive: true });
}

writeFileSync(join(DIST, 'index.html'), INDEX_HTML, 'utf8');
writeFileSync(join(DIST, 'login.html'), LOGIN_HTML, 'utf8');

console.log('dist: esbuild bundle → dist/src/{main,login,workers,chunks} + styles/static + HTML shells');
