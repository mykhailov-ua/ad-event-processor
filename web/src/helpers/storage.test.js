import { describe, test, expect, beforeEach, afterEach } from 'vitest';
import { getThemePalette, setThemePalette } from './storage.js';

describe('storage.js', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme-palette');
  });

  afterEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme-palette');
  });

  test('theme-palette Whitelisted round-trip', () => {
    expect(getThemePalette()).toBe('neutral');

    setThemePalette('neutral');
    expect(getThemePalette()).toBe('neutral');
    expect(document.documentElement.getAttribute('data-theme-palette')).toBe('neutral');

    setThemePalette('default');
    expect(getThemePalette()).toBe('default');
    expect(document.documentElement.getAttribute('data-theme-palette')).toBe('default');
  });

  test('theme-palette rejects invalid values', () => {
    setThemePalette('invalid');
    expect(getThemePalette()).toBe('neutral');
  });
});
