import assert from 'node:assert/strict';
import test from 'node:test';

import {
  parseInAppLocation,
  readExportHubReturnPath,
  rememberExportHubReturnPath,
} from '@/lib/export_hub_return';
import { resolveExportHubReturnHref } from '@/lib/export_hub_paths';

test('parseInAppLocation splits pathname search and hash', () => {
  assert.deepEqual(parseInAppLocation('/campaigns?status=ACTIVE&offset=0'), {
    pathname: '/campaigns',
    search: '?status=ACTIVE&offset=0',
    hash: '',
  });
});

test('rememberExportHubReturnPath stores only campaigns paths', () => {
  const storage = new Map<string, string>();
  const original = globalThis.sessionStorage;
  Object.defineProperty(globalThis, 'sessionStorage', {
    configurable: true,
    value: {
      getItem: (key: string) => storage.get(key) ?? null,
      setItem: (key: string, value: string) => {
        storage.set(key, value);
      },
    },
  });

  rememberExportHubReturnPath('/campaigns?status=PAUSED');
  assert.equal(readExportHubReturnPath(), '/campaigns?status=PAUSED');

  rememberExportHubReturnPath('/exports?job_id=1');
  assert.equal(readExportHubReturnPath(), '/campaigns?status=PAUSED');

  Object.defineProperty(globalThis, 'sessionStorage', {
    configurable: true,
    value: original,
  });
});

test('resolveExportHubReturnHref falls back to session storage', () => {
  const storage = new Map<string, string>();
  const original = globalThis.sessionStorage;
  Object.defineProperty(globalThis, 'sessionStorage', {
    configurable: true,
    value: {
      getItem: (key: string) => storage.get(key) ?? null,
      setItem: (key: string, value: string) => {
        storage.set(key, value);
      },
    },
  });

  rememberExportHubReturnPath('/campaigns?customer_id=abc');
  assert.equal(resolveExportHubReturnHref(new URLSearchParams()), '/campaigns?customer_id=abc');

  Object.defineProperty(globalThis, 'sessionStorage', {
    configurable: true,
    value: original,
  });
});
