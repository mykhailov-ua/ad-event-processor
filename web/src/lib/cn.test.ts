import test from 'node:test';
import assert from 'node:assert/strict';
import { cn } from './cn.ts';

test('cn joins class names', () => {
  assert.equal(cn('a', false, 'b'), 'a b');
});
