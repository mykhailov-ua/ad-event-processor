import { describe, it, expect } from 'vitest';
import { getConfirmLevel } from './confirm_registry.js';

describe('confirm_registry.js', () => {
  it('returns correct confirm level for registered endpoints', () => {
    expect(getConfirmLevel('POST', '/auth/login').level).toBe('none');
    expect(getConfirmLevel('POST', '/billing/invoices/123/void').level).toBe('strong');
    expect(getConfirmLevel('POST', '/selfserve/campaigns').level).toBe('financial');
    expect(getConfirmLevel('POST', '/ops/blacklist').level).toBe('destructive');
  });

  it('handles wildcard pattern matching for {id}', () => {
    expect(getConfirmLevel('POST', '/selfserve/campaigns/abc-123/pause').level).toBe('destructive');
  });
});
