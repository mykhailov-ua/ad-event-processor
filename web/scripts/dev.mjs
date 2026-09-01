#!/usr/bin/env node

import { createServer, request as httpRequest } from 'node:http';
import { existsSync, readFileSync, watch } from 'node:fs';
import { extname, resolve, dirname, join } from 'node:path';
import { spawn } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const DIST = join(ROOT, 'dist');
const SRC = join(ROOT, 'src');
const PORT = Number(process.env.ADMIN_DEV_PORT ?? 5173);
const API_TARGET = process.env.ADMIN_API_PROXY ?? 'http://127.0.0.1:8188';
const WATCH = process.env.ADMIN_DEV_WATCH !== '0';
const BUILD_SCRIPT = join(ROOT, 'scripts', 'build.mjs');

function runBuild() {
  return new Promise((resolvePromise, reject) => {
    const child = spawn(process.execPath, [BUILD_SCRIPT], { stdio: 'inherit', cwd: ROOT });
    child.on('exit', (code) => {
      if (code === 0) resolvePromise();
      else reject(new Error(`build exited ${code}`));
    });
  });
}

const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.mjs': 'text/javascript; charset=utf-8',
  '.map': 'application/json',
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
    }
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

function startServer() {
  createServer((req, res) => {
    const path = req.url?.split('?')[0] ?? '/';

    if (path === '/favicon.ico') {
      res.writeHead(204);
      res.end();
      return;
    }

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

    const rel = path.replace(/^\//, '');
    const filePath = resolve(DIST, rel);
    if (filePath.startsWith(DIST)) {
      serveFile(res, filePath);
      return;
    }

    res.writeHead(404);
    res.end('Not found');
  }).listen(PORT, () => {
    console.log(`Admin dev: http://127.0.0.1:${PORT} (API -> ${API_TARGET})`);
    if (WATCH) console.log('Watching web/src for rebuilds');
  });
}

await runBuild();
startServer();

if (WATCH) {
  let timer = null;
  let building = false;
  let pending = false;
  const kick = () => {
    if (building) {
      pending = true;
      return;
    }
    building = true;
    runBuild()
      .catch((err) => console.error(err))
      .finally(() => {
        building = false;
        if (pending) {
          pending = false;
          kick();
        }
      });
  };
  watch(SRC, { recursive: true }, () => {
    clearTimeout(timer);
    timer = setTimeout(kick, 120);
  });
  watch(join(ROOT, 'tailwind.config.ts'), () => {
    clearTimeout(timer);
    timer = setTimeout(kick, 120);
  });
}
