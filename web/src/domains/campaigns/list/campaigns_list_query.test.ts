import test from 'node:test';
import assert from 'node:assert/strict';

import {
  applyCampaignListQueryPatch,
  buildCampaignListQuery,
  parseCampaignListSort,
} from './campaigns_list_query.ts';

test('parseCampaignListSort maps legacy id to updated_at', () => {
  assert.equal(parseCampaignListSort('id'), 'updated_at');
});

test('buildCampaignListQuery adds stats window for metric sort', () => {
  const params = new URLSearchParams({
    sort: 'clicks',
    order: 'desc',
    stats_from: '2026-01-01T00:00:00.000Z',
    stats_to: '2026-01-31T00:00:00.000Z',
  });

  const query = buildCampaignListQuery(params, undefined);

  assert.equal(query.sort, 'clicks');
  assert.equal(query.from, '2026-01-01T00:00:00.000Z');
  assert.equal(query.to, '2026-01-31T00:00:00.000Z');
});

test('applyCampaignListQueryPatch clears removed filters', () => {
  const current = new URLSearchParams({
    customer_id: 'c1',
    status: 'ACTIVE',
    q: 'foo',
    limit: '50',
    offset: '0',
    sort: 'name',
    order: 'asc',
  });
  const query = buildCampaignListQuery(current, undefined);
  const next = applyCampaignListQueryPatch(current, query, {
    status: undefined,
    q: undefined,
    offset: 0,
  });

  assert.equal(next.get('customer_id'), 'c1');
  assert.equal(next.get('status'), null);
  assert.equal(next.get('q'), null);
});
