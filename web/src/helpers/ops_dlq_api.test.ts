import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  buildDlqListUrl,
  isOpsDlqEntryRetryable,
} from './ops_dlq_api.js';
import type { DLQEntryDTO } from '../types/api/ops_extra.js';

const SAMPLE_ENTRY: DLQEntryDTO = {
  id: 'shard-1-1700000000000-0',
  shard_id: 1,
  stream_id: 'events:ch',
  entry_id: '1700000000000-0',
  failed_at: '2026-08-12T10:00:00Z',
  retry_count: 1,
};

describe('buildDlqListUrl', () => {
  it('defaults limit to 50 without cursor', () => {
    assert.equal(buildDlqListUrl(), '/api/v1/ops/dlq?limit=50');
  });

  it('includes cursor when provided', () => {
    const url = buildDlqListUrl('shard-1-abc', 25);
    assert.equal(url, '/api/v1/ops/dlq?limit=25&cursor=shard-1-abc');
  });
});

describe('isOpsDlqEntryRetryable', () => {
  it('allows retry for pending entries', () => {
    assert.equal(isOpsDlqEntryRetryable(SAMPLE_ENTRY), true);
  });

  it('blocks retry after RETRIED status', () => {
    assert.equal(
      isOpsDlqEntryRetryable({ ...SAMPLE_ENTRY, status: 'RETRIED' }),
      false,
    );
  });
});
