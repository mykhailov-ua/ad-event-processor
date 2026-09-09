import assert from 'node:assert/strict';
import test from 'node:test';

import { ApiError } from '../api/api_error.ts';
import {
  adminErrorUserMessage,
  formatAdminErrorDetails,
  isPaymentUnavailableError,
  normalizeSubmitError,
  userErrorMessage,
} from './admin_error.ts';
import { validationError } from './admin_validation_error.ts';

test('userErrorMessage hides generic Error details outside dev mode', () => {
  const message = userErrorMessage(new Error('sql: connection refused on 10.0.0.5'));
  assert.equal(message, 'Something went wrong. Try again or return to the home page.');
});

test('userErrorMessage surfaces AdminValidationError copy', () => {
  assert.equal(
    userErrorMessage(validationError('Customer name is required.')),
    'Customer name is required.'
  );
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
    userErrorMessage(new ApiError(501, 'NOT_IMPLEMENTED', 'Wizard not wired')),
    'Wizard not wired'
  );
  assert.equal(
    userErrorMessage(new ApiError(501, 'NOT_IMPLEMENTED', '')),
    'Something went wrong. Try again or return to the home page.'
  );
  assert.equal(
    userErrorMessage(new ApiError(500, 'INTERNAL_ERROR', 'sql: connection refused')),
    'The server encountered an error. Try again later.'
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

test('normalizeSubmitError preserves ApiError and wraps unknown values', () => {
  const apiError = new ApiError(400, 'BAD_REQUEST', 'invalid credentials');
  assert.equal(normalizeSubmitError(apiError, 'Sign in failed'), apiError);

  const generic = normalizeSubmitError('network down', 'Sign in failed');
  assert.equal(generic.message, 'Sign in failed');
  assert.notEqual(generic, apiError);
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
