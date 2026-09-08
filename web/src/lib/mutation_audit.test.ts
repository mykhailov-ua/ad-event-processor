import assert from 'node:assert/strict';
import test from 'node:test';

import { mutationError } from './mutation_audit.ts';

test('mutationError wraps unknown values', () => {
  assert.equal(mutationError(new Error('boom')).message, 'boom');
  assert.equal(mutationError('oops').message, 'oops');
});
