import { resolve, dirname } from 'node:path';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');

/**
 * @param {{ dev?: boolean }} opts
 */
export function esbuildOptions(opts = {}) {
  const dev = opts.dev ?? false;
  const pkg = JSON.parse(readFileSync(resolve(ROOT, 'package.json'), 'utf8'));
  const buildLabel = String(pkg.version ?? '0.0.0');
  return {
    absWorkingDir: ROOT,
    entryPoints: {
      'assets/main': resolve(ROOT, 'src/main.js'),
      'assets/login': resolve(ROOT, 'src/login.js'),
    },
    outdir: resolve(ROOT, 'dist'),
    bundle: true,
    splitting: true,
    format: 'esm',
    platform: 'browser',
    target: ['es2020'],
    sourcemap: dev,
    minify: !dev,
    metafile: true,
    entryNames: dev ? 'assets/[name]' : 'assets/[name]-[hash]',
    chunkNames: 'assets/[name]-[hash]',
    assetNames: 'assets/[name]-[hash]',
    define: {
      'import.meta.env.DEV': JSON.stringify(dev),
      'import.meta.env.PROD': JSON.stringify(!dev),
      'import.meta.env.BUILD_LABEL': JSON.stringify(buildLabel),
    },
    loader: {
      '.css': 'css',
      '.svg': 'file',
    },
  };
}

export const ROOT_DIR = ROOT;
