import assert from 'node:assert/strict';
import test from 'node:test';

import { ApiError } from '../api/api_error.ts';
import {
  actionGuardError,
  isValidationError,
  requireDateRange,
  requireHexLength,
  requireInteger,
  requireJsonObject,
  requireNonEmpty,
  requireNonNegativeInteger,
  requirePositiveInteger,
  requireZeroOrOne,
  toValidationError,
  validationError,
  validationErrorMessage,
} from './admin_validation_error.ts';

test('validationErrorMessage surfaces AdminValidationError copy', () => {
  assert.equal(
    validationErrorMessage(validationError('Customer is required.')),
    'Customer is required.'
  );
});

test('validationErrorMessage surfaces ApiError 400 message', () => {
  assert.equal(
    validationErrorMessage(new ApiError(400, 'BAD_REQUEST', 'budget_limit_micro must be positive')),
    'budget_limit_micro must be positive'
  );
});

test('userErrorMessage path hides generic Error but validationErrorMessage shows it', () => {
  const generic = new Error('sql: connection refused');
  assert.equal(validationErrorMessage(generic), 'sql: connection refused');
});

test('actionGuardError is validation error with action kind', () => {
  const err = actionGuardError('Select at least one campaign');
  assert.equal(isValidationError(err), true);
  assert.equal(err.kind, 'action');
  assert.equal(err.code, 'ACTION_GUARD');
});

test('requireNonEmpty rejects blank input', () => {
  const result = requireNonEmpty('  ', 'Customer ID', 'customer_id');
  assert.equal(result.ok, false);
  if (!result.ok) {
    assert.equal(result.error.message, 'Customer ID is required.');
    assert.equal(result.error.field, 'customer_id');
  }
});

test('requirePositiveInteger rejects zero and non-integers', () => {
  assert.equal(requirePositiveInteger('0', 'Budget').ok, false);
  assert.equal(requirePositiveInteger('12.5', 'Budget').ok, false);
  const ok = requirePositiveInteger('100', 'Budget');
  assert.equal(ok.ok, true);
  if (ok.ok) {
    assert.equal(ok.value, 100);
  }
});

test('requireNonNegativeInteger accepts zero', () => {
  const ok = requireNonNegativeInteger('0', 'spend_cap_micro');
  assert.equal(ok.ok, true);
});

test('requireInteger enforces bounds', () => {
  const fail = requireInteger('256', 'Threshold', { min: 0, max: 255 });
  assert.equal(fail.ok, false);
});

test('requireHexLength enforces length and charset', () => {
  assert.equal(requireHexLength('abc', 32, 'IP hash').ok, false);
  const hex = 'a'.repeat(32);
  assert.equal(requireHexLength(hex, 32, 'IP hash').ok, true);
});

test('requireZeroOrOne accepts 0 and 1 only', () => {
  assert.equal(requireZeroOrOne('2', 'Label').ok, false);
  assert.equal(requireZeroOrOne('1', 'Label').ok, true);
});

test('requireJsonObject rejects arrays', () => {
  assert.equal(requireJsonObject('[]', 'Setup configuration').ok, false);
  assert.equal(requireJsonObject('{"a":1}', 'Setup configuration').ok, true);
});

test('requireDateRange rejects inverted range', () => {
  const result = requireDateRange('2026-01-02T00:00', '2026-01-01T00:00');
  assert.equal(result.ok, false);
  if (!result.ok) {
    assert.match(result.error.message, /before/);
  }
});

test('toValidationError wraps unknown values', () => {
  const wrapped = toValidationError('bad', 'Fallback message');
  assert.equal(wrapped.message, 'Fallback message');
  assert.equal(isValidationError(wrapped), true);
});
