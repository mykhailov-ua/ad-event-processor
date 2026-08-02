import { describe, it, expect } from 'vitest';
import {
  validateReportRange,
  validateUuid,
  validateSelfServeBudget,
} from './validators.js';

describe('validators.js', () => {
  it('validateReportRange rejects range over 90 days', () => {
    const from = '2026-01-01T00:00:00Z';
    const to = '2026-05-01T00:00:00Z';
    expect(validateReportRange(from, to)).toBe('range exceeds 90 days');
  });

  it('validateReportRange rejects from >= to', () => {
    expect(validateReportRange('2026-02-01', '2026-01-01')).toBe('from must be before to');
  });

  it('validateUuid accepts lowercase uuid', () => {
    expect(validateUuid('550e8400-e29b-41d4-a716-446655440000')).toBe(true);
    expect(validateUuid('bad')).toBe(false);
  });

  it('validateSelfServeBudget enforces bounds', () => {
    expect(validateSelfServeBudget(100, 50, 200)).toBeNull();
    expect(validateSelfServeBudget(10, 50, 200)).toMatch(/at least/);
    expect(validateSelfServeBudget(500, 50, 200)).toMatch(/exceeds maximum/);
  });
});
