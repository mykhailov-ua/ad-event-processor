#!/usr/bin/env node

import { createServer } from 'node:http';
import { existsSync, readFileSync } from 'node:fs';
import { extname, resolve, dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const DIST = join(ROOT, 'dist');
const PORT = Number(process.env.ADMIN_PREVIEW_PORT ?? 4173);

const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.mjs': 'text/javascript; charset=utf-8',
  '.map': 'application/json',
};

function serveFile(res, filePath) {
  if (!existsSync(filePath)) {
    res.writeHead(404);
    res.end('Not found');
    return;
  }
  const ext = extname(filePath);
  res.writeHead(200, {
    'Content-Type': MIME[ext] ?? 'application/octet-stream',
    'Cache-Control': ext === '.html' ? 'no-cache' : 'public, max-age=31536000, immutable',
  });
  res.end(readFileSync(filePath));
}

function isSpaPath(path) {
  if (path.startsWith('/api/')) return false;
  if (path.startsWith('/src/')) return false;
  if (path.includes('.')) return false;
  return true;
}

createServer((req, res) => {
  const path = req.url?.split('?')[0] ?? '/';

  if (path === '/login' || path === '/login.html') {
    serveFile(res, join(DIST, 'login.html'));
    return;
  }

  if (path === '/' || path === '/index.html' || isSpaPath(path)) {
    serveFile(res, join(DIST, 'index.html'));
    return;
  }

  const rel = path.replace(/^\
  serveFile(res, resolve(DIST, rel));
}).listen(PORT, () => {
  console.log(`Admin preview: http://127.0.0.1:${PORT}`);
});
