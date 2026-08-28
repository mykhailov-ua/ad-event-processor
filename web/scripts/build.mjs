#!/usr/bin/env node

import * as esbuild from 'esbuild';
import { cpSync, mkdirSync, readFileSync, rmSync, writeFileSync, existsSync } from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const DIST = join(ROOT, 'dist');
const SRC = join(ROOT, 'src');

const require = createRequire(import.meta.url);
try {
  require.resolve('esbuild');
} catch {
  console.error('esbuild missing. Run: cd web && npm ci');
  process.exit(1);
}

const ts = Date.now();
function baseFontLinks() {
  return `    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@fontsource/geist-sans@5.2.5/400.css" />
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@fontsource/geist-mono@5.2.5/400.css" />
    <link rel="stylesheet" href="/src/styles/tokens.css?v=${ts}" />
    <link rel="stylesheet" href="/src/styles/base.css?v=${ts}" />
`;
}

function distCssHref(cssBundlePath) {
  const webPath = cssBundlePath.replace(/^dist\//, '/');
  return `${webPath}?v=${ts}`;
}

function stylesheetLink(href) {
  return `    <link rel="stylesheet" href="${href}" />\n`;
}

function htmlShell({ title, scriptSrc, bundleCssHref }) {
  const bundleLink = bundleCssHref ? stylesheetLink(bundleCssHref) : '';
  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>${title}</title>
${baseFontLinks()}${bundleLink}  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="${scriptSrc}"></script>
  </body>
</html>
`;
}

const HTML_CSS_ENTRIES = new Set(['src/main.tsx', 'src/login.tsx']);

function cssImportPath(jsOutPath, cssBundlePath) {
  const rel = relative(dirname(jsOutPath), cssBundlePath).replace(/\\/g, '/');
  return rel.startsWith('.') ? rel : `./${rel}`;
}

function cssLinkLoaderSnippet(cssRelPath) {
  return `(()=>{const href=new URL(${JSON.stringify(cssRelPath)},import.meta.url).href;if(document.querySelector(\`link[data-esbuild-css="\${href}"]\`))return;const link=document.createElement("link");link.rel="stylesheet";link.href=href;link.setAttribute("data-esbuild-css",href);document.head.appendChild(link)})();\n`;
}

function attachCssBundles(metafile) {
  for (const [jsOutPath, output] of Object.entries(metafile.outputs)) {
    if (!output.cssBundle || !jsOutPath.endsWith('.js')) continue;
    if (output.entryPoint && HTML_CSS_ENTRIES.has(output.entryPoint)) continue;
    const jsAbs = join(ROOT, jsOutPath);
    const cssRelPath = cssImportPath(jsOutPath, output.cssBundle);
    const loader = cssLinkLoaderSnippet(cssRelPath);
    let source = readFileSync(jsAbs, 'utf8');
    if (source.includes(`data-esbuild-css`)) continue;
    writeFileSync(jsAbs, `${loader}${source}`);
  }
}

function entryCssHref(metafile, entryPoint) {
  const entrySuffix = entryPoint.replace(`${ROOT}/`, '');
  for (const output of Object.values(metafile.outputs)) {
    if (output.entryPoint === entrySuffix && output.cssBundle) {
      return distCssHref(output.cssBundle);
    }
  }
  return null;
}

rmSync(DIST, { recursive: true, force: true });
mkdirSync(join(DIST, 'src'), { recursive: true });

const bundleResult = await esbuild.build({
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
  metafile: true,
  logLevel: 'info',
  loader: {
    '.ts': 'ts',
    '.tsx': 'tsx',
    '.js': 'js',
  },
});

attachCssBundles(bundleResult.metafile);

cpSync(join(SRC, 'styles'), join(DIST, 'src', 'styles'), { recursive: true });

const systemCssPath = join(SRC, 'styles', 'system.css');
const mainCssPath = join(SRC, 'styles', 'main.css');
if (existsSync(systemCssPath)) {
  cpSync(systemCssPath, join(DIST, 'src', 'styles', 'system.css'));
} else {
  writeFileSync(
    join(DIST, 'src', 'styles', 'system.css'),
    '.truncate{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;min-width:0}\n',
    'utf8'
  );
}
if (existsSync(mainCssPath)) {
  cpSync(mainCssPath, join(DIST, 'src', 'styles', 'main.css'));
} else {
  writeFileSync(
    join(DIST, 'src', 'styles', 'main.css'),
    "@import url('./tokens.css');\n@import url('./base.css');\n",
    'utf8'
  );
}

const staticDir = join(SRC, 'static');
const trackSrc = join(staticDir, 'track.js');
if (existsSync(trackSrc)) {
  mkdirSync(join(DIST, 'src', 'static'), { recursive: true });
  cpSync(trackSrc, join(DIST, 'src', 'static', 'track.js'));
} else {
  console.error('Error: missing web/src/static/track.js (required for go:embed admin UI)');
  process.exit(1);
}

const workersDir = join(SRC, 'workers');
const workerEntries = ['parse_json.worker.ts', 'report_aggregate.worker.ts'].map((name) =>
  join(workersDir, name)
);
for (const entry of workerEntries) {
  if (!existsSync(entry)) {
    console.error(`Error: missing ${entry} (required for go:embed admin UI)`);
    process.exit(1);
  }
}
await esbuild.build({
  absWorkingDir: ROOT,
  entryPoints: workerEntries,
  bundle: true,
  format: 'esm',
  platform: 'browser',
  target: ['es2022'],
  outdir: join(DIST, 'src', 'workers'),
  outbase: workersDir,
  entryNames: '[name]',
  minify: true,
  logLevel: 'info',
  loader: {
    '.ts': 'ts',
  },
});

const mainCssHref = entryCssHref(bundleResult.metafile, join(SRC, 'main.tsx'));
const loginCssHref = entryCssHref(bundleResult.metafile, join(SRC, 'login.tsx'));

writeFileSync(
  join(DIST, 'index.html'),
  htmlShell({
    title: 'ad-event-processor Admin',
    scriptSrc: `/src/main.js?v=${ts}`,
    bundleCssHref: mainCssHref,
  }),
  'utf8'
);
writeFileSync(
  join(DIST, 'login.html'),
  htmlShell({
    title: 'Sign in - ad-event-processor Admin',
    scriptSrc: `/src/login.js?v=${ts}`,
    bundleCssHref: loginCssHref,
  }),
  'utf8'
);

console.log('dist: esbuild bundle -> dist/src/{main,login,chunks} + styles + HTML shells');

const HYDRATOR_OUT = resolve(ROOT, '..', 'internal', 'ingestion', 'safe_page_hydrator.js');

await esbuild.build({
  absWorkingDir: ROOT,
  entryPoints: [join(SRC, 'safe_page_hydrator_entry.ts')],
  bundle: true,
  format: 'iife',
  platform: 'browser',
  target: ['es2022'],
  outfile: HYDRATOR_OUT,
  minify: true,
  logLevel: 'info',
  loader: {
    '.ts': 'ts',
  },
});

console.log('hydrator: esbuild -> internal/ingestion/safe_page_hydrator.js');
