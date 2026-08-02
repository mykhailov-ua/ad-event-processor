import { describe, it, expect, beforeEach } from 'vitest';
import {
  isCustomerUuid,
  shortCustomerId,
  touchCustomerContext,
} from './customer_context.js';
import * as storage from './storage.js';

describe('customer_context', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('validates UUIDs', () => {
    expect(isCustomerUuid('550e8400-e29b-41d4-a716-446655440000')).toBe(true);
    expect(isCustomerUuid('not-a-uuid')).toBe(false);
  });

  it('shortens ids for display', () => {
    expect(shortCustomerId('550e8400-e29b-41d4-a716-446655440000')).toBe('550e8400…');
  });

  it('tracks recent customers', () => {
    const id = '550e8400-e29b-41d4-a716-446655440000';
    touchCustomerContext(id);
    expect(storage.getRecentCustomerIds()).toEqual([id]);
    expect(storage.getLastCustomerId()).toBe(id);
  });
});
