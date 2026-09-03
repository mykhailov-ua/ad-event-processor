#!/usr/bin/env node

import { register } from 'node:module';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { dirname, join } from 'node:path';

register('./test_resolve.mjs', import.meta.url);

const ROOT = dirname(fileURLToPath(import.meta.url));

let resolveDevMockRequest;

async function loadMockHandler() {
  if (!resolveDevMockRequest) {
    ({ resolveDevMockRequest } = await import(
      pathToFileURL(join(ROOT, '../src/api/dev_mock/handler.ts')).href
    ));
  }
}

export async function respondDevMockApi(req, res, bodyText) {
  await loadMockHandler();

  const url = new URL(req.url ?? '/', 'http://dev.local');
  const path = url.pathname + url.search;
  const result = resolveDevMockRequest(path, {
    method: req.method ?? 'GET',
    body: bodyText || undefined,
  });

  const mockHeaders = {
    'X-Admin-Dev-Mock': '1',
    'Cache-Control': 'no-store',
  };

  if (!result) {
    res.writeHead(404, {
      ...mockHeaders,
      'Content-Type': 'application/json; charset=utf-8',
    });
    res.end(
      JSON.stringify({
        error: { code: 'MOCK_NOT_FOUND', message: `No dev mock handler for ${url.pathname}` },
      }),
    );
    return;
  }

  if (result.status === 204 || result.status === 202) {
    res.writeHead(result.status, mockHeaders);
    res.end();
    return;
  }

  res.writeHead(result.status, {
    ...mockHeaders,
    'Content-Type': result.contentType ?? 'application/json; charset=utf-8',
    ...(result.status === 501 ? { 'X-API-Stub': 'true' } : {}),
  });
  res.end(JSON.stringify(result.body ?? {}));
}
