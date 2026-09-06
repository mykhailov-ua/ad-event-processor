import assert from 'node:assert/strict';
import test from 'node:test';

import { ApiError } from '../api/api_error.ts';
import {
  adminErrorUserMessage,
  formatAdminErrorDetails,
  isPaymentUnavailableError,
  userErrorMessage,
} from './admin_error.ts';

test('userErrorMessage hides generic Error details outside dev mode', () => {
  const message = userErrorMessage(new Error('sql: connection refused on 10.0.0.5'));
  assert.equal(message, 'Something went wrong. Try again or return to the home page.');
});

test('userErrorMessage maps ApiError status to operator copy', () => {
  assert.equal(
    userErrorMessage(new ApiError(404, 'NOT_FOUND', 'campaign missing')),
    'The requested resource was not found.'
  );
  assert.equal(
    userErrorMessage(new ApiError(403, 'FORBIDDEN', 'role denied')),
    'You do not have permission to view this resource.'
  );
  assert.equal(
    userErrorMessage(new ApiError(0, 'TIMEOUT', 'timed out')),
    'The request timed out. Check your connection and try again.'
  );
});

test('formatAdminErrorDetails includes ApiError fields', () => {
  const details = formatAdminErrorDetails(new ApiError(502, 'BAD_GATEWAY', 'upstream down'));
  assert.match(details, /status: 502/);
  assert.match(details, /code: BAD_GATEWAY/);
  assert.match(details, /upstream down/);
});

test('userErrorMessage maps payment unavailable', () => {
  assert.equal(
    userErrorMessage(new ApiError(503, 'PAYMENT_UNAVAILABLE', 'payment service not configured')),
    'Payment history is not available in this deployment. Enable the payment module or use the ledger tab for balance activity.'
  );
});

test('isPaymentUnavailableError matches PAYMENT_UNAVAILABLE code', () => {
  assert.equal(
    isPaymentUnavailableError(
      new ApiError(503, 'PAYMENT_UNAVAILABLE', 'payment service not configured')
    ),
    true
  );
  assert.equal(
    isPaymentUnavailableError(new ApiError(500, 'INTERNAL_ERROR', 'internal error')),
    false
  );
});
