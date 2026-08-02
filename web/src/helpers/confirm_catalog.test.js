import { describe, it, expect } from 'vitest';
import { registry } from './confirm_registry.js';
import { REQUIRED_CONFIRM_KEYS } from './confirm_catalog.js';

describe('confirm_catalog vs registry', () => {
  it('has zero missing mutation keys', () => {
    const missing = REQUIRED_CONFIRM_KEYS.filter((key) => !registry.has(key));
    expect(missing).toEqual([]);
  });
});
