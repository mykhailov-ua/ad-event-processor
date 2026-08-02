import { describe, it, expect } from 'vitest';
import { ParseDecimal, formatMicro, formatMicroFull } from './money.js';

describe('money.js', () => {
  it('parses valid decimal strings to micro-units', () => {
    expect(ParseDecimal('10.50')).toBe(10500000);
    expect(ParseDecimal('0')).toBe(0);
    expect(ParseDecimal('0.000001')).toBe(1);
    expect(ParseDecimal('100.123456')).toBe(100123456);
    expect(ParseDecimal('-5.25')).toBe(-5250000);
  });

  it('rejects invalid decimal strings', () => {
    expect(() => ParseDecimal('')).toThrow();
    expect(() => ParseDecimal('abc')).toThrow();
    expect(() => ParseDecimal('10.1234567')).toThrow();
  });

  it('formats micro-units to display string', () => {
    expect(formatMicro(10500000)).toBe('10.50');
    expect(formatMicro(0)).toBe('0.00');
    expect(formatMicro(-5250000)).toBe('-5.25');
  });

  it('formats micro-units to 6 decimal places', () => {
    expect(formatMicroFull(100123456)).toBe('100.123456');
  });
});
