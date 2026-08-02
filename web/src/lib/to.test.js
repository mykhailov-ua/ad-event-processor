import { describe, expect, it } from 'vitest';
import { to } from './to.js';

describe('to', () => {
  it('returns value on success', async () => {
    const [val, err] = await to(Promise.resolve(42));
    expect(err).toBeNull();
    expect(val).toBe(42);
  });

  it('returns Error on failure', async () => {
    const [val, err] = await to(Promise.reject(new Error('fail')));
    expect(val).toBeNull();
    expect(err?.message).toBe('fail');
  });
});
