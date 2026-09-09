import assert from 'node:assert/strict';
import test from 'node:test';

import { ApiError } from '@/api/client';
import {
  buildCloneRequestBody,
  cloneMutationErrorMessage,
  cloneRequestError,
  DEFAULT_CLONE_OPTIONS,
} from '@/domains/campaigns/editor/campaign_clone_request';

test('buildCloneRequestBody omits empty name_suffix', () => {
  const body = buildCloneRequestBody('   ', DEFAULT_CLONE_OPTIONS);
  assert.deepEqual(body, { options: DEFAULT_CLONE_OPTIONS });
});

test('buildCloneRequestBody_holdoutIncludesSuffixAndOptions', () => {
  const body = buildCloneRequestBody(' (copy)', {
    ...DEFAULT_CLONE_OPTIONS,
    include_flow: false,
  });
  assert.equal(body.name_suffix, '(copy)');
  assert.equal(body.options?.include_flow, false);
});

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

test('cloneRequestError_holdout preserves ApiError from parseApiError', () => {
  const err = new ApiError(400, 'BAD_REQUEST', 'Idempotency-Key header is required');
  const normalized = cloneRequestError(err);
  assert.equal(normalized.message, 'Idempotency-Key header is required');
  assert.equal((normalized as ApiError).code, 'BAD_REQUEST');
});
