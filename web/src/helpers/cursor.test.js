import { describe, it, expect } from 'vitest';
import { encodeCursor, decodeCursor } from './cursor.js';

describe('cursor.js', () => {
  it('encodes offset as base64 int string', () => {
    expect(encodeCursor(25)).toBe(btoa('25'));
    expect(decodeCursor(encodeCursor(25))).toBe(25);
  });

  it('decode empty cursor returns 0', () => {
    expect(decodeCursor('')).toBe(0);
  });

  it('rejects invalid cursor', () => {
    expect(() => decodeCursor(btoa('not-a-number'))).toThrow('invalid cursor');
  });
});
