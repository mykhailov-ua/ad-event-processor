import { describe, it, expect } from 'vitest';
import { appendPage } from './immutable.js';

describe('immutable.js', () => {
  it('appendPage does not mutate prior array', () => {
    const pages = [['a']];
    const next = appendPage(pages, ['b']);
    expect(next).toEqual([['a'], ['b']]);
    expect(pages).toEqual([['a']]);
  });
});
