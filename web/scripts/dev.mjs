#!/usr/bin/env node

import { createServer, request as httpRequest } from 'node:http';
import { existsSync, readFileSync, watch } from 'node:fs';
import { extname, join, resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

import { adminApiTarget, prepareDevBuildEnv } from './admin_dev_env.mjs';
import {
  BUILD_DIST as DIST,
  BUILD_ROOT as ROOT,
  BUILD_SRC as SRC,
  buildAppCss,
  buildDevBootstrap,
  createDevEsbuildContext,
} from './build_lib.mjs';

const PORT = Number(process.env.ADMIN_DEV_PORT ?? 5173);
const WATCH = process.env.ADMIN_DEV_WATCH !== '0';
const LIVERELOAD_PATH = '/__livereload';
const LIVERELOAD_SNIPPET =
  '<script>(function(){var es=new EventSource("/__livereload");es.onmessage=function(){location.reload()};})();</script>';

const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.mjs': 'text/javascript; charset=utf-8',
  '.map': 'application/json',
  '.json': 'application/json',
  '.svg': 'image/svg+xml',
};

const reloadClients = new Set();
let reloadTimer = null;
let cssBuilding = false;
let cssPending = false;

function scheduleReload() {
  clearTimeout(reloadTimer);
  reloadTimer = setTimeout(() => {
    for (const client of reloadClients) {
      client.write('data: reload\n\n');
    }
  }, 120);
}

function serveBytes(res, body, contentType) {
  res.writeHead(200, {
    'Content-Type': contentType,
    'Cache-Control': 'no-store',
  });
  res.end(body);
}

function serveFile(res, filePath) {
  if (!existsSync(filePath)) {
    res.writeHead(404, { 'Cache-Control': 'no-store' });
    res.end('Not found');
    return;
  }
  const ext = extname(filePath);
  let body = readFileSync(filePath);
  if (ext === '.html') {
    const html = body.toString('utf8');
    if (!html.includes(LIVERELOAD_PATH)) {
      body = Buffer.from(html.replace('</body>', `    ${LIVERELOAD_SNIPPET}\n  </body>`), 'utf8');
    }
  }
  serveBytes(res, body, MIME[ext] ?? 'application/octet-stream');
}

function readRequestBody(req) {
  return new Promise((resolvePromise, reject) => {
    const chunks = [];
    req.on('data', (chunk) => chunks.push(chunk));
    req.on('end', () => resolvePromise(Buffer.concat(chunks).toString('utf8')));
    req.on('error', reject);
  });
}

function proxyApi(req, res, bodyText) {
  const apiTarget = adminApiTarget();
  const url = new URL(req.url ?? '/', apiTarget);
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
  if (bodyText) {
    upstream.end(bodyText);
    return;
  }
  req.pipe(upstream);
}

async function handleApi(req, res) {
  const bodyText = await readRequestBody(req);
  proxyApi(req, res, bodyText);
}

function shouldProxyToControl(path) {
  if (path.startsWith('/api/')) {
    return true;
  }
  return path === '/health' || path === '/healthz' || path === '/readyz' || path === '/metrics';
}

function isSpaPath(path) {
  if (shouldProxyToControl(path)) {
    return false;
  }
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

    if (path === LIVERELOAD_PATH) {
      res.writeHead(200, {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-store',
        Connection: 'keep-alive',
      });
      res.write(':\n\n');
      reloadClients.add(res);
      req.on('close', () => reloadClients.delete(res));
      return;
    }

    if (shouldProxyToControl(path)) {
      void handleApi(req, res);
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
    console.log(`Admin dev: http://127.0.0.1:${PORT} (API -> ${adminApiTarget()})`);
    if (WATCH) {
      console.log('Live reload: save a file -> incremental rebuild -> browser refresh');
    }
  });
}

async function runCssBuild() {
  if (cssBuilding) {
    cssPending = true;
    return;
  }
  cssBuilding = true;
  try {
    await buildAppCss();
    scheduleReload();
  } catch (err) {
    console.error(err);
  } finally {
    cssBuilding = false;
    if (cssPending) {
      cssPending = false;
      void runCssBuild();
    }
  }
}

function watchCssSources() {
  let timer = null;
  const kick = () => {
    clearTimeout(timer);
    timer = setTimeout(() => {
      void runCssBuild();
    }, 300);
  };
  watch(join(SRC, 'styles'), { recursive: true }, kick);
  watch(join(ROOT, 'tailwind.config.ts'), kick);
  watch(SRC, { recursive: true }, (_event, filename) => {
    if (!filename) {
      kick();
      return;
    }
    if (/\.(tsx?|jsx?)$/.test(filename)) {
      kick();
    }
  });
}

await prepareDevBuildEnv();
await buildDevBootstrap();

let esbuildContext = null;
if (WATCH) {
  esbuildContext = await createDevEsbuildContext({
    onRebuild: () => scheduleReload(),
  });
  await esbuildContext.watch();
  watchCssSources();
} else {
  const { buildJsBundle } = await import('./build_lib.mjs');
  await buildJsBundle({ minify: false });
}

startServer();

process.on('SIGINT', async () => {
  if (esbuildContext) {
    await esbuildContext.dispose();
  }
  process.exit(0);
});
