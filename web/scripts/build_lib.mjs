import * as esbuild from 'esbuild';
import {
  cpSync,
  existsSync,
  mkdirSync,
  readFileSync,
  rmSync,
  statSync,
  writeFileSync,
} from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';
import postcss from 'postcss';
import tailwindcss from 'tailwindcss';
import autoprefixer from 'autoprefixer';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const DIST = join(ROOT, 'dist');
const SRC = join(ROOT, 'src');

export const BUILD_ROOT = ROOT;
export const BUILD_DIST = DIST;
export const BUILD_SRC = SRC;
export const ADMIN_UI_BARE = process.env.ADMIN_UI_BARE === '1';

const require = createRequire(import.meta.url);

function fontPackageSlug(packageName) {
  const trimmed = packageName.replace(/^@fontsource(-variable)?\//, '');
  return trimmed.split('/')[0];
}

function loadFontCss(packageName, fontsOutDir) {
  const cssPath = require.resolve(packageName);
  const cssDir = dirname(cssPath);
  const filesDir = join(cssDir, 'files');
  const slug = fontPackageSlug(packageName);
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

function loadVendorCss(packageEntry) {
  const cssPath = require.resolve(packageEntry);
  return readFileSync(cssPath, 'utf8');
}

export function aliasAtPlugin() {
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
        if (existsSync(base)) {
          try {
            if (statSync(base).isDirectory()) {
              for (const ext of exts) {
                const indexCandidate = join(base, `index${ext}`);
                if (existsSync(indexCandidate)) {
                  return { path: indexCandidate };
                }
              }
            } else {
              return { path: base };
            }
          } catch {
            // Fall through to default extension probe below.
          }
        }
        return { path: base + '.tsx' };
      });
    },
  };
}

export async function buildAppCss() {
  const outDir = join(DIST, 'src', 'styles');
  mkdirSync(outDir, { recursive: true });
  if (ADMIN_UI_BARE) {
    writeFileSync(
      join(outDir, 'app.css'),
      readFileSync(join(SRC, 'styles', 'skeleton.css'), 'utf8'),
      'utf8'
    );
    return;
  }

  const inputPath = join(SRC, 'styles', 'tailwind.css');
  const sonnerCss = loadVendorCss('sonner/dist/styles.css');
  let input = `${sonnerCss}\n${readFileSync(inputPath, 'utf8')}`;
  const FONT_IMPORTS = [
    '@fontsource-variable/inter',
    '@fontsource/inter/400.css',
    '@fontsource/inter/500.css',
    '@fontsource/inter/600.css',
    '@fontsource/inter/700.css',
  ];
  for (const pkg of FONT_IMPORTS) {
    const token = `@import '${pkg}';`;
    if (input.includes(token)) {
      input = input.replace(token, loadFontCss(pkg, outDir));
    }
  }
  const result = await postcss([
    tailwindcss(join(ROOT, 'tailwind.config.ts')),
    autoprefixer,
  ]).process(input, { from: inputPath });
  writeFileSync(join(outDir, 'app.css'), result.css, 'utf8');
}

export function copyCountryFlagSvgs() {
  const pkgJson = require.resolve('country-flag-icons/package.json');
  const flagsSrc = join(dirname(pkgJson), '3x2');
  const flagsDest = join(DIST, 'src', 'flags', '3x2');
  if (!existsSync(flagsSrc)) {
    throw new Error('country-flag-icons 3x2 SVGs missing. Run: cd web && npm install');
  }
  mkdirSync(flagsDest, { recursive: true });
  cpSync(flagsSrc, flagsDest, { recursive: true });
}

export function buildHtmlShells({ cacheBust = Date.now() } = {}) {
  const fontLinks = `    <link rel="stylesheet" href="/src/styles/app.css?v=${cacheBust}" />\n`;

  function buildHtmlShell(sourceName, scriptSrc) {
    const sourcePath = join(ROOT, sourceName);
    const base = existsSync(sourcePath)
      ? readFileSync(sourcePath, 'utf8')
      : `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>AEP Admin (Ad Event Processor - AEP)</title>
  </head>
  <body>
    <div id="root"></div>
  </body>
</html>
`;

    const withFonts = base.replace('</head>', `${fontLinks}  </head>`);
    return withFonts.replace(
      '</body>',
      `    <script type="module" src="${scriptSrc}"></script>\n  </body>`
    );
  }

  writeFileSync(join(DIST, 'index.html'), buildHtmlShell('index.html', '/src/main.js'), 'utf8');
  writeFileSync(join(DIST, 'login.html'), buildHtmlShell('login.html', '/src/login.js'), 'utf8');
}

export function copyThemeBootScripts() {
  const staticDir = join(SRC, 'static');
  const distStatic = join(DIST, 'src', 'static');
  mkdirSync(distStatic, { recursive: true });
  for (const name of ['theme_boot.js', 'theme_boot_login.js']) {
    const src = join(staticDir, name);
    if (!existsSync(src)) {
      throw new Error(`missing web/src/static/${name}`);
    }
    cpSync(src, join(distStatic, name));
  }
}

export async function buildTrackPixel() {
  const staticDir = join(SRC, 'static');
  const trackSrc = join(staticDir, 'track.js');
  if (!existsSync(trackSrc)) {
    throw new Error('missing web/src/static/track.js (required for go:embed admin UI)');
  }
  const { execFileSync } = await import('node:child_process');
  execFileSync('node', ['scripts/build_track_pixel.mjs', `--dist=${DIST}`], {
    cwd: ROOT,
    stdio: 'inherit',
  });
}

function esbuildOptions({ minify, extraPlugins = [] }) {
  return {
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
    minify,
    logLevel: 'info',
    plugins: [aliasAtPlugin(), ...extraPlugins],
    define: {
      'import.meta.env.ADMIN_UI_BARE': ADMIN_UI_BARE ? '"1"' : '""',
    },
    loader: {
      '.ts': 'ts',
      '.tsx': 'tsx',
      '.js': 'js',
      '.css': 'empty',
    },
  };
}

export async function buildJsBundle({ minify = true, extraPlugins = [] } = {}) {
  mkdirSync(join(DIST, 'src'), { recursive: true });
  await esbuild.build(esbuildOptions({ minify, extraPlugins }));
}

export function devRebuildPlugin(onRebuild) {
  return {
    name: 'dev-rebuild',
    setup(build) {
      build.onEnd((result) => {
        if (result.errors.length === 0) {
          onRebuild();
        }
      });
    },
  };
}

export async function createDevEsbuildContext({ onRebuild } = {}) {
  mkdirSync(join(DIST, 'src'), { recursive: true });
  const plugins = onRebuild ? [devRebuildPlugin(onRebuild)] : [];
  const context = await esbuild.context(
    esbuildOptions({ minify: false, extraPlugins: plugins })
  );
  return context;
}

export async function buildProduction() {
  try {
    require.resolve('esbuild');
  } catch {
    console.error('esbuild missing. Run: cd web && npm install');
    process.exit(1);
  }

  rmSync(DIST, { recursive: true, force: true });
  mkdirSync(join(DIST, 'src'), { recursive: true });

  await buildAppCss();
  copyCountryFlagSvgs();
  await buildJsBundle({ minify: true });
  copyThemeBootScripts();
  await buildTrackPixel();
  buildHtmlShells();

  console.log(
    ADMIN_UI_BARE
      ? 'dist: esbuild bundle -> dist/src/{main,login,chunks} + skeleton app.css (ADMIN_UI_BARE=1)'
      : 'dist: esbuild bundle -> dist/src/{main,login,chunks} + tailwind app.css + HTML shells'
  );
}

export async function buildDevBootstrap() {
  mkdirSync(join(DIST, 'src'), { recursive: true });
  await buildAppCss();
  copyCountryFlagSvgs();
  copyThemeBootScripts();
  await buildTrackPixel();
  buildHtmlShells();
}
