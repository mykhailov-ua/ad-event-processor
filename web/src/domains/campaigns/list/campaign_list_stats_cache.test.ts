import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildCampaignStatsCacheKey,
  campaignStatsFromListMetrics,
  clearCampaignStatsCache,
  readCachedCampaignStats,
  readCachedCampaignStatsForCampaign,
  writeCachedCampaignStats,
} from './campaign_list_stats_cache.ts';

test('campaignStatsFromListMetrics maps list batch counts', () => {
  const stats = campaignStatsFromListMetrics('cmp-1', {
    impressions: 10,
    clicks: 4,
    conversions: 1,
    stale: true,
  });
  assert.equal(stats.campaign_id, 'cmp-1');
  assert.equal(stats.metrics?.impressions, 10);
  assert.equal(stats.metrics?.clicks, 4);
  assert.equal(stats.metrics?.conversions, 1);
  assert.equal(stats.source, 'list_batch');
  assert.equal(stats.hourly?.length, 0);
});

test('campaignStatsCache_holdoutReusesSessionSnapshot', () => {
  clearCampaignStatsCache();
  const key = buildCampaignStatsCacheKey('cmp-1', { from: 'a', to: 'b' }, 'scope-1');
  const seeded = campaignStatsFromListMetrics('cmp-1', { impressions: 1, clicks: 0, conversions: 0 });
  writeCachedCampaignStats(key, {
    ...seeded,
    hourly: [{ hour: '2026-01-01T00:00:00Z', impressions: 1, clicks: 0, conversions: 0 }],
    source: 'stats',
  });

  const cached = readCachedCampaignStats(key);
  assert.equal(cached?.source, 'stats');
  assert.equal(cached?.hourly?.length, 1);
  assert.equal(readCachedCampaignStats(buildCampaignStatsCacheKey('cmp-2', {}, 'scope-1')), undefined);
});

test('readCachedCampaignStatsForCampaign_holdoutFindsListScopeRevision', () => {
  clearCampaignStatsCache();
  const listKey = buildCampaignStatsCacheKey('cmp-9', { from: 'a', to: 'b' }, 'list-scope');
  writeCachedCampaignStats(listKey, {
    ...campaignStatsFromListMetrics('cmp-9', { impressions: 3, clicks: 1, conversions: 0 }),
    source: 'stats',
  });

  const found = readCachedCampaignStatsForCampaign('cmp-9', { from: 'a', to: 'b' });
  assert.equal(found?.metrics?.impressions, 3);
  assert.equal(readCachedCampaignStatsForCampaign('cmp-9', {}), undefined);
});

test('readCachedCampaignStats_holdoutExpiresAfterTtl', () => {
  clearCampaignStatsCache();
  const key = buildCampaignStatsCacheKey('cmp-ttl', { from: 'a', to: 'b' }, 'scope-1');
  const originalNow = Date.now;
  let now = 1_000;
  Date.now = () => now;

  try {
    writeCachedCampaignStats(key, {
      ...campaignStatsFromListMetrics('cmp-ttl', { impressions: 9, clicks: 0, conversions: 0 }),
      source: 'stats',
    });
    assert.equal(readCachedCampaignStats(key)?.metrics?.impressions, 9);

    now += 61_000;
    assert.equal(readCachedCampaignStats(key), undefined);
  } finally {
    Date.now = originalNow;
    clearCampaignStatsCache();
  }
});
