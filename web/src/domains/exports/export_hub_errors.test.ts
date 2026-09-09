import assert from 'node:assert/strict';
import test from 'node:test';

import { ApiError } from '@/api/api_error';
import { exportHubErrorMessage, exportHubJobErrorMessage } from '@/domains/exports/export_hub_errors';

test('exportHubErrorMessage maps client timeout', () => {
  const message = exportHubErrorMessage(new ApiError(0, 'TIMEOUT', 'timeout'));
  assert.match(message, /timed out/i);
});

test('exportHubErrorMessage hides clickhouse details from job status', () => {
  const message = exportHubJobErrorMessage('clickhouse: Code: 241. DB::Exception: Memory limit exceeded');
  assert.ok(message);
  assert.doesNotMatch(message!, /clickhouse/i);
  assert.match(message!, /data source failed/i);
});

test('exportHubErrorMessage keeps validation copy', () => {
  const message = exportHubErrorMessage(new ApiError(400, 'BAD_REQUEST', 'invalid customer_id'));
  assert.equal(message, 'invalid customer_id');
});
