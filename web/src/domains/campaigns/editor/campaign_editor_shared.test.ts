import assert from 'node:assert/strict';
import test from 'node:test';

import { ApiError } from '@/api/client';
import { cloneMutationErrorMessage } from '@/domains/campaigns/editor/campaign_editor_shared';

test('cloneMutationErrorMessage maps insufficient balance to operator guidance', () => {
  const err = new ApiError(400, 'BAD_REQUEST', 'insufficient balance');
  const message = cloneMutationErrorMessage(err);
  assert.match(message, /balance is too low/i);
  assert.match(message, /budget_limit/i);
});

test('cloneMutationErrorMessage_holdout passes through unrelated errors', () => {
  const err = new ApiError(400, 'BAD_REQUEST', 'invalid campaign id');
  assert.equal(cloneMutationErrorMessage(err), 'invalid campaign id');
});
