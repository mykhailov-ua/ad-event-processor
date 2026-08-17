import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  buildDlqInboxListUrl,
  isDlqInboxEntryRetryable,
} from './ops_dlq_inbox_api.js';

describe('buildDlqInboxListUrl', () => {
  it('defaults limit to 50 without filters', () => {
    assert.equal(buildDlqInboxListUrl(), '/api/v1/ops/dlq/inbox?limit=50');
  });

  it('includes source and cursor when provided', () => {
    const url = buildDlqInboxListUrl('capi', 'cursor-1', 25);
    assert.equal(url, '/api/v1/ops/dlq/inbox?limit=25&source=capi&cursor=cursor-1');
  });
});

describe('isDlqInboxEntryRetryable', () => {
  it('allows retry for failed entries', () => {
    assert.equal(
      isDlqInboxEntryRetryable({ id: '1', source: 'postback', status: 'FAILED' }),
      true,
    );
  });

  it('blocks retry after RETRIED status', () => {
    assert.equal(
      isDlqInboxEntryRetryable({ id: '1', source: 'capi', status: 'RETRIED' }),
      false,
    );
  });
});
