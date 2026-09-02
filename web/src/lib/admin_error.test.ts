import assert from 'node:assert/strict';
import test from 'node:test';

import { ApiError } from '../api/api_error.ts';
import {
  adminErrorUserMessage,
  formatAdminErrorDetails,
  userErrorMessage,
} from './admin_error.ts';

test('userErrorMessage hides generic Error details outside dev mode', () => {
  const message = userErrorMessage(new Error('sql: connection refused on 10.0.0.5'));
  assert.equal(message, 'Something went wrong. Try again or return to the home page.');
});

test('userErrorMessage maps ApiError status to operator copy', () => {
  assert.equal(
    userErrorMessage(new ApiError(404, 'NOT_FOUND', 'campaign missing')),
    'The requested resource was not found.',
  );
  assert.equal(
    userErrorMessage(new ApiError(403, 'FORBIDDEN', 'role denied')),
    'You do not have permission to view this resource.',
  );
  assert.equal(
    userErrorMessage(new ApiError(0, 'TIMEOUT', 'timed out')),
    'The request timed out. Check your connection and try again.',
  );
});

test('formatAdminErrorDetails includes ApiError fields', () => {
  const details = formatAdminErrorDetails(new ApiError(502, 'BAD_GATEWAY', 'upstream down'));
  assert.match(details, /status: 502/);
  assert.match(details, /code: BAD_GATEWAY/);
  assert.match(details, /upstream down/);
});

test('adminErrorUserMessage has copy for not-found', () => {
  assert.match(adminErrorUserMessage('not-found'), /does not exist/);
});
