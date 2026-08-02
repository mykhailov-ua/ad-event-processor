import { createServer, request as httpRequest } from 'node:http';
import { readFileSync, existsSync, writeFileSync, mkdirSync } from 'node:fs';
import { resolve, extname } from 'node:path';
import esbuild from 'esbuild';
import { esbuildOptions, ROOT_DIR } from './esbuild-shared.mjs';
import { INDEX_HTML, LOGIN_HTML, renderHtml, assetsForEntry } from './html-templates.mjs';

const PORT = Number(process.env.ADMIN_DEV_PORT ?? 5173);
const API_TARGET = process.env.ADMIN_API_PROXY ?? 'http://127.0.0.1:8188';
const distDir = resolve(ROOT_DIR, 'dist');

mkdirSync(distDir, { recursive: true });

function writeHtmlFromMeta(meta) {
  const mainAssets = assetsForEntry(meta, 'src/main.js');
  const loginAssets = assetsForEntry(meta, 'src/login.js');
  writeFileSync(
    resolve(distDir, 'index.html'),
    renderHtml(
      INDEX_HTML,
      mainAssets.scripts.length ? mainAssets.scripts : ['/assets/main.js'],
      mainAssets.styles.length ? mainAssets.styles : ['/assets/main.css'],
    ),
    'utf8',
  );
  writeFileSync(
    resolve(distDir, 'login.html'),
    renderHtml(
      LOGIN_HTML,
      loginAssets.scripts.length ? loginAssets.scripts : ['/assets/login.js'],
      loginAssets.styles.length ? loginAssets.styles : ['/assets/login.css'],
    ),
    'utf8',
  );
}

const devOpts = esbuildOptions({ dev: true });
devOpts.plugins = [{
  name: 'write-dev-html',
  setup(build) {
    build.onEnd((result) => {
      if (result.metafile) writeHtmlFromMeta(result.metafile);
    });
  },
}];

const ctx = await esbuild.context(devOpts);
await ctx.watch();
await ctx.rebuild();

const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.svg': 'image/svg+xml',
};

function serveFile(res, filePath) {
  if (!existsSync(filePath)) {
    res.writeHead(404);
    res.end('Not found');
    return;
  }
  const ext = extname(filePath);
  res.writeHead(200, { 'Content-Type': MIME[ext] ?? 'application/octet-stream' });
  res.end(readFileSync(filePath));
}

function proxyApi(req, res) {
  const url = new URL(req.url ?? '/', API_TARGET);
  const upstream = httpRequest(
    {
      method: req.method,
      hostname: url.hostname,
      port: url.port,
      path: url.pathname + url.search,
      headers: req.headers,
    },
    (proxyRes) => {
      res.writeHead(proxyRes.statusCode ?? 502, proxyRes.headers);
      proxyRes.pipe(res);
    },
  );
  upstream.on('error', () => {
    res.writeHead(502);
    res.end('API proxy error');
  });
  req.pipe(upstream);
}

function isSpaPath(path) {
  if (path.startsWith('/api/')) return false;
  if (path.startsWith('/assets/')) return false;
  if (path.includes('.')) return false;
  return true;
}

createServer((req, res) => {
  const path = req.url?.split('?')[0] ?? '/';

  if (path.startsWith('/api/')) {
    proxyApi(req, res);
    return;
  }

  if (path === '/login' || path === '/login.html') {
    serveFile(res, resolve(distDir, 'login.html'));
    return;
  }

  if (path === '/' || path === '/index.html' || isSpaPath(path)) {
    serveFile(res, resolve(distDir, 'index.html'));
    return;
  }

  const rel = path.replace(/^\//, '');
  const filePath = resolve(distDir, rel);
  if (filePath.startsWith(distDir)) {
    serveFile(res, filePath);
    return;
  }

  res.writeHead(404);
  res.end('Not found');
}).listen(PORT, () => {
  console.log(`Admin dev: http://127.0.0.1:${PORT} (API → ${API_TARGET})`);
});
