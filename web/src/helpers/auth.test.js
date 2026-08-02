import { describe, it, expect, beforeEach } from 'vitest';
import * as auth from './auth.js';
import { getOrCreate } from './idempotency.js';
import * as storage from './storage.js';

describe('auth.js logout', () => {
  beforeEach(() => {
    auth.setCsrfFromLoginResponse('csrf-test');
    auth.setUser({
      id: 'u1',
      email: 'a@test',
      role: 'A',
      customer_id: '',
      permissions: [],
    });
    getOrCreate('logout-test-scope');
    storage.setIdempotencyPending('logout-pending', {
      key: 'k1',
      bodyHash: 'h1',
      ts: Date.now(),
    });
  });

  it('logoutLocal clears user and csrf', () => {
    auth.logoutLocal();
    expect(auth.getUser()).toBeNull();
    expect(auth.getCsrfToken()).toBeNull();
  });

  it('logoutLocal clears idempotency scopes', () => {
    const before = getOrCreate('logout-test-scope');
    auth.logoutLocal();
    const after = getOrCreate('logout-test-scope');
    expect(before).toBeTruthy();
    expect(after).not.toBe(before);
  });

  it('logoutLocal clears idempotency pending in localStorage', () => {
    expect(storage.getIdempotencyPending('logout-pending')).toBeTruthy();
    auth.logoutLocal();
    expect(storage.getIdempotencyPending('logout-pending')).toBeNull();
  });
});
