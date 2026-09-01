import test from 'node:test';
import assert from 'node:assert/strict';
import { cn } from './utils.ts';

test('cn merges tailwind classes', () => {
  assert.equal(cn('px-2 py-1', 'px-4'), 'py-1 px-4');
});

test('cn joins class names', () => {
  assert.equal(cn('a', false, 'b'), 'a b');
});
