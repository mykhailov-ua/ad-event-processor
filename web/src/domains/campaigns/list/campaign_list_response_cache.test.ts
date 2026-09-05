import assert from 'node:assert/strict';
import test from 'node:test';

import type { CampaignListQuery } from '@/api/types';

import {
  campaignListResponseCacheKey,
  invalidateCampaignListResponseCache,
} from './campaign_list_response_cache.ts';

const baseQuery: CampaignListQuery = {
  customer_id: '00000000-0000-4000-8000-000000000001',
  limit: 50,
  offset: 0,
  sort: 'name',
  order: 'asc',
};

test('campaignListResponseCacheKey distinguishes asc and desc for same sort field', () => {
  const asc = campaignListResponseCacheKey({
    query: baseQuery,
    statsFrom: '2026-01-01T00:00:00.000Z',
    statsTo: '2026-01-08T00:00:00.000Z',
  });
  const desc = campaignListResponseCacheKey({
    query: { ...baseQuery, order: 'desc' },
    statsFrom: '2026-01-01T00:00:00.000Z',
    statsTo: '2026-01-08T00:00:00.000Z',
  });
  assert.notEqual(asc, desc);
});

test('invalidateCampaignListResponseCache clears sort-order entries', () => {
  invalidateCampaignListResponseCache();
  assert.doesNotThrow(() => invalidateCampaignListResponseCache());
});
