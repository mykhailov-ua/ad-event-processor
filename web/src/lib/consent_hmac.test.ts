import assert from 'node:assert/strict';
import { createHmac } from 'node:crypto';
import test from 'node:test';

import { buildConsentRecordJson } from './consent_hmac.ts';

test('buildConsentRecordJson omits empty timestamp', () => {
  assert.equal(
    buildConsentRecordJson({
      user_id: 'user-1',
      purposes: 1,
      source: 'cmp',
    }),
    '{"user_id":"user-1","purposes":1,"source":"cmp"}'
  );
});

test('buildConsentRecordJson includes timestamp when set', () => {
  const body = buildConsentRecordJson({
    user_id: 'user-1',
    purposes: 2,
    source: 'admin',
    timestamp: '2026-09-07T12:00:00Z',
  });
  assert.equal(
    body,
    '{"user_id":"user-1","purposes":2,"source":"admin","timestamp":"2026-09-07T12:00:00Z"}'
  );
  const expected = createHmac('sha256', 'test-secret').update(body).digest('hex');
  assert.match(expected, /^[0-9a-f]{64}$/);
});
