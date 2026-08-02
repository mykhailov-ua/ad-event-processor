import { writeFileSync, rmSync } from 'node:fs';
import { resolve } from 'node:path';
import esbuild from 'esbuild';
import { esbuildOptions, ROOT_DIR } from './esbuild-shared.mjs';
import {
  INDEX_HTML,
  LOGIN_HTML,
  renderHtml,
  assetsForEntry,
} from './html-templates.mjs';

const distDir = resolve(ROOT_DIR, 'dist');

rmSync(distDir, { recursive: true, force: true });

const result = await esbuild.build(esbuildOptions({ dev: false }));

if (!result.metafile) {
  throw new Error('esbuild metafile missing');
}

const mainAssets = assetsForEntry(result.metafile, 'src/main.js');
const loginAssets = assetsForEntry(result.metafile, 'src/login.js');

writeFileSync(
  resolve(distDir, 'index.html'),
  renderHtml(INDEX_HTML, mainAssets.scripts, mainAssets.styles),
  'utf8',
);
writeFileSync(
  resolve(distDir, 'login.html'),
  renderHtml(LOGIN_HTML, loginAssets.scripts, loginAssets.styles),
  'utf8',
);

console.log('esbuild: dist written');
