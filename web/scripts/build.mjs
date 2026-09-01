#!/usr/bin/env node

import * as esbuild from 'esbuild';
import { cpSync, mkdirSync, readFileSync, rmSync, writeFileSync, existsSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';
import postcss from 'postcss';
import tailwindcss from 'tailwindcss';
import autoprefixer from 'autoprefixer';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const DIST = join(ROOT, 'dist');
const SRC = join(ROOT, 'src');

const require = createRequire(import.meta.url);
try {
  require.resolve('esbuild');
} catch {
  console.error('esbuild missing. Run: cd web && npm install');
  process.exit(1);
}

const ts = Date.now();
const FONT_LINKS = `    <link rel="stylesheet" href="/src/styles/globals.css?v=${ts}" />
`;

function buildHtmlShell(sourceName, scriptSrc) {
  const sourcePath = join(ROOT, sourceName);
  const base = existsSync(sourcePath)
    ? readFileSync(sourcePath, 'utf8')
    : `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>ad-event-processor Admin</title>
  </head>
  <body>
    <div id="root"></div>
  </body>
</html>
`;

  const withFonts = base.replace('</head>', `${FONT_LINKS}  </head>`);
  return withFonts.replace(
    '</body>',
    `    <script type="module" src="${scriptSrc}"></script>\n  </body>`,
  );
}

const INDEX_HTML = buildHtmlShell('index.html', `/src/main.js?v=${ts}`);
const LOGIN_HTML = buildHtmlShell('login.html', `/src/login.js?v=${ts}`);

function aliasAtPlugin() {
  const exts = ['.tsx', '.ts', '.jsx', '.js'];
  return {
    name: 'alias-at',
    setup(build) {
      build.onResolve({ filter: /^@\// }, (args) => {
        const rel = args.path.slice(2);
        const base = join(SRC, rel);
        for (const ext of exts) {
          const candidate = base + ext;
          if (existsSync(candidate)) {
            return { path: candidate };
          }
        }
        if (existsSync(base) && !base.endsWith('/')) {
          return { path: base };
        }
        return { path: base + '.tsx' };
      });
    },
  };
}

function loadFontCss(packageName, fontsOutDir) {
  const cssPath = require.resolve(`${packageName}/index.css`);
  const cssDir = dirname(cssPath);
  const filesDir = join(cssDir, 'files');
  const slug = packageName.split('/').pop();
  const targetDir = join(fontsOutDir, 'fonts', slug);
  mkdirSync(targetDir, { recursive: true });

  let css = readFileSync(cssPath, 'utf8');
  css = css.replace(/url\((['"]?)(\.\/files\/([^)'"]+))\1\)/g, (_match, _quote, _rel, file) => {
    const src = join(filesDir, file);
    if (existsSync(src)) {
      cpSync(src, join(targetDir, file));
    }
    return `url('./fonts/${slug}/${file}')`;
  });
  return css;
}

async function buildGlobalsCss() {
  const inputPath = join(SRC, 'styles', 'globals.css');
  const outDir = join(DIST, 'src', 'styles');
  let input = readFileSync(inputPath, 'utf8');
  for (const pkg of [
    '@fontsource-variable/inter',
    '@fontsource-variable/jetbrains-mono',
    '@fontsource-variable/geist',
    '@fontsource-variable/geist-mono',
  ]) {
    const token = `@import '${pkg}';`;
    if (input.includes(token)) {
      input = input.replace(token, loadFontCss(pkg, outDir));
    }
  }
  const result = await postcss([
    tailwindcss(join(ROOT, 'tailwind.config.ts')),
    autoprefixer,
  ]).process(input, { from: inputPath });
  mkdirSync(outDir, { recursive: true });
  writeFileSync(join(outDir, 'globals.css'), result.css, 'utf8');
}

rmSync(DIST, { recursive: true, force: true });
mkdirSync(join(DIST, 'src'), { recursive: true });

await buildGlobalsCss();

await esbuild.build({
  absWorkingDir: ROOT,
  entryPoints: [join(SRC, 'main.tsx'), join(SRC, 'login.tsx')],
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
  plugins: [aliasAtPlugin()],
  loader: {
    '.ts': 'ts',
    '.tsx': 'tsx',
    '.js': 'js',
    '.css': 'empty',
  },
});

const staticDir = join(SRC, 'static');
const trackSrc = join(staticDir, 'track.js');
if (existsSync(trackSrc)) {
  mkdirSync(join(DIST, 'src', 'static'), { recursive: true });
  cpSync(trackSrc, join(DIST, 'src', 'static', 'track.js'));
} else {
  console.error('Error: missing web/src/static/track.js (required for go:embed admin UI)');
  process.exit(1);
}

writeFileSync(join(DIST, 'index.html'), INDEX_HTML, 'utf8');
writeFileSync(join(DIST, 'login.html'), LOGIN_HTML, 'utf8');

console.log('dist: esbuild bundle -> dist/src/{main,login,chunks} + tailwind globals.css + HTML shells');
