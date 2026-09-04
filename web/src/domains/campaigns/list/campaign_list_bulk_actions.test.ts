import assert from 'node:assert/strict';
import test from 'node:test';

import { CAMPAIGN_LIST_BULK_CHUNK_SIZE } from '@/domains/campaigns/list/campaign_list_limits.ts';

test('bulk archive uses the same server chunk size as pause and resume', () => {
  assert.equal(CAMPAIGN_LIST_BULK_CHUNK_SIZE, 50);
});
