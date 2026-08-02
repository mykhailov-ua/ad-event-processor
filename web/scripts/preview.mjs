import { createServer } from 'node:http';
import { readFileSync, existsSync } from 'node:fs';
import { resolve, extname } from 'node:path';
import { ROOT_DIR } from './esbuild-shared.mjs';

const PORT = Number(process.env.ADMIN_PREVIEW_PORT ?? 4173);
const distDir = resolve(ROOT_DIR, 'dist');

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
  res.writeHead(200, {
    'Content-Type': MIME[ext] ?? 'application/octet-stream',
    'Cache-Control': ext === '.html' ? 'no-cache' : 'public, max-age=31536000, immutable',
  });
  res.end(readFileSync(filePath));
}

function isSpaPath(path) {
  if (path.startsWith('/api/')) return false;
  if (path.startsWith('/assets/')) return false;
  if (path.includes('.')) return false;
  return true;
}

createServer((req, res) => {
  const path = req.url?.split('?')[0] ?? '/';

  if (path === '/login' || path === '/login.html') {
    serveFile(res, resolve(distDir, 'login.html'));
    return;
  }

  if (path === '/' || path === '/index.html' || isSpaPath(path)) {
    serveFile(res, resolve(distDir, 'index.html'));
    return;
  }

  const rel = path.replace(/^\//, '');
  serveFile(res, resolve(distDir, rel));
}).listen(PORT, () => {
  console.log(`Admin preview: http://127.0.0.1:${PORT}`);
});
