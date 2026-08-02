import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import * as auth from './auth.js';
import { getOrCreate, clearScope } from './idempotency.js';

describe('api_client CSRF', () => {
  beforeEach(() => {
    auth.logoutLocal();
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('sends X-CSRF-Token on mutations when token is set', async () => {
    auth.setCsrfFromLoginResponse('csrf-test-token');
    const { api } = await import('./api_client.js');

    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ 'content-type': 'application/json' }),
      text: async () => '{}',
    });

    await api('/api/v1/settings/platform', { method: 'PATCH', body: '{}' });

    const [, init] = vi.mocked(fetch).mock.calls[0];
    const headers = init.headers;
    expect(headers.get('X-CSRF-Token')).toBe('csrf-test-token');
  });

  it('does not send CSRF header on GET', async () => {
    auth.setCsrfFromLoginResponse('csrf-test-token');
    const { api } = await import('./api_client.js');

    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ 'content-type': 'application/json' }),
      text: async () => '{}',
    });

    await api('/api/v1/meta');

    const [, init] = vi.mocked(fetch).mock.calls[0];
    const headers = init.headers;
    expect(headers.get('X-CSRF-Token')).toBeNull();
  });
});

describe('api_client Idempotency-Key', () => {
  beforeEach(() => {
    auth.logoutLocal();
    clearScope('test-scope');
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('sends stable Idempotency-Key for idempotencyScope', async () => {
    const { api } = await import('./api_client.js');

    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ 'content-type': 'application/json' }),
      text: async () => '{}',
    });

    const keyBefore = getOrCreate('test-scope');
    await api('/api/v1/selfserve/campaigns', {
      method: 'POST',
      body: '{}',
      idempotencyScope: 'test-scope',
    });

    const [, init] = vi.mocked(fetch).mock.calls[0];
    expect(init.headers.get('Idempotency-Key')).toBe(keyBefore);
  });
});
