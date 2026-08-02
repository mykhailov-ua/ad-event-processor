import { describe, it, expect, beforeEach } from 'vitest';
import { getOrCreate, clearScope, clearAll } from './idempotency.js';

describe('idempotency.js', () => {
  beforeEach(() => {
    clearAll();
  });

  it('returns stable key for same scope', () => {
    const k1 = getOrCreate('create-campaign');
    const k2 = getOrCreate('create-campaign');
    expect(k1).toBe(k2);
  });

  it('returns new key after clearScope', () => {
    const k1 = getOrCreate('create-campaign');
    clearScope('create-campaign');
    const k2 = getOrCreate('create-campaign');
    expect(k1).not.toBe(k2);
  });
});
