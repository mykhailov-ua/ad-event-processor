#!/usr/bin/env node
/**
 * Serve dist/ HTML + live web/src/ assets with API proxy for local admin dev.
 */
import { createServer, request as httpRequest } from 'node:http';
import { existsSync, readFileSync } from 'node:fs';
import { extname, resolve } from 'node:path';
import { spawn } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const DIST = join(ROOT, 'dist');
const SRC = join(ROOT, 'src');
const PORT = Number(process.env.ADMIN_DEV_PORT ?? 5173);
const API_TARGET = process.env.ADMIN_API_PROXY ?? 'http://127.0.0.1:8188';

const build = spawn(process.execPath, [join(ROOT, 'scripts', 'build.mjs')], {
  stdio: 'inherit',
  cwd: ROOT,
});
build.on('exit', (code) => {
  if (code !== 0) process.exit(code ?? 1);
  startServer();
});

function startServer() {
  const MIME = {
    '.html': 'text/html; charset=utf-8',
    '.css': 'text/css; charset=utf-8',
    '.js': 'text/javascript; charset=utf-8',
    '.mjs': 'text/javascript; charset=utf-8',
    '.json': 'application/json',
    '.svg': 'image/svg+xml',
  };

  function serveFile(res, filePath) {
    if (!existsSync(filePath)) {
      res.writeHead(404, { 'Cache-Control': 'no-store' });
      res.end('Not found');
      return;
    }
    const ext = extname(filePath);
    res.writeHead(200, {
      'Content-Type': MIME[ext] ?? 'application/octet-stream',
      'Cache-Control': 'no-store',
    });
    res.end(readFileSync(filePath));
  }

  function proxyApi(req, res) {
    const url = new URL(req.url ?? '/', API_TARGET);
    const headers = { ...req.headers, host: url.host };
    delete headers.connection;
    const upstream = httpRequest(
      {
        method: req.method,
        hostname: url.hostname,
        port: url.port,
        path: url.pathname + url.search,
        headers,
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
    if (path.startsWith('/src/')) return false;
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
      serveFile(res, join(DIST, 'login.html'));
      return;
    }

    if (path === '/' || path === '/index.html' || isSpaPath(path)) {
      serveFile(res, join(DIST, 'index.html'));
      return;
    }

    // Live assets from web/src/ — no rebuild needed for CSS/JS edits.
    if (path.startsWith('/src/')) {
      const srcPath = resolve(SRC, path.slice('/src/'.length));
      if (srcPath.startsWith(SRC)) {
        console.log(`[LIVE] ${path} -> ${srcPath}`);
        serveFile(res, srcPath);
        return;
      }
    }

    const rel = path.replace(/^\//, '');
    const filePath = resolve(DIST, rel);
    if (filePath.startsWith(DIST)) {
      serveFile(res, filePath);
      return;
    }

    res.writeHead(404);
    res.end('Not found');
  }).listen(PORT, () => {
    console.log(`Admin dev: http://127.0.0.1:${PORT} (API → ${API_TARGET})`);
    console.log('Live: /src/* served from web/src/ (hard refresh if styles look stale)');
  });
}
