import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import * as auth from './auth.js';
import { isAuthMutationPath, tryRefreshSession } from './session.js';

describe('session.js', () => {
  it('detects auth mutation paths', () => {
    expect(isAuthMutationPath('/api/v1/auth/login')).toBe(true);
    expect(isAuthMutationPath('/api/v1/campaigns')).toBe(false);
  });

  describe('tryRefreshSession', () => {
    /** @type {import('vitest').MockInstance} */
    let fetchMock;

    beforeEach(() => {
      fetchMock = vi.fn();
      vi.stubGlobal('fetch', fetchMock);
    });

    afterEach(() => {
      vi.unstubAllGlobals();
    });

    it('returns false when refresh fails', async () => {
      fetchMock.mockResolvedValueOnce({ ok: false });
      expect(await tryRefreshSession()).toBe(false);
    });

    it('hydrates user and csrf after refresh + me', async () => {
      fetchMock
        .mockResolvedValueOnce({ ok: true })
        .mockResolvedValueOnce({
          ok: true,
          headers: new Headers({ 'X-CSRF-Token': 'csrf-refreshed' }),
          text: async () => JSON.stringify({
            id: 'u2',
            email: 'ref@test',
            role: 'A',
            customer_id: '',
            permissions: ['settings:read'],
          }),
        });

      expect(await tryRefreshSession()).toBe(true);
      expect(auth.getCsrfToken()).toBe('csrf-refreshed');
      expect(auth.getUser()?.email).toBe('ref@test');
    });
  });
});
