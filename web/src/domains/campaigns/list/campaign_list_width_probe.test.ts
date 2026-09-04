import test from 'node:test';
import assert from 'node:assert/strict';

import type { CampaignListQuery } from '@/api/types';

import {
  buildCampaignListWidthProbeQuery,
  listResponseCoversWidthProbeDataset,
  mergeCampaignIdsForMetricsBatch,
} from './campaign_list_width_probe.ts';

test('buildCampaignListWidthProbeQuery mirrors list filters with probe sort', () => {
  const query: CampaignListQuery = {
    customer_id: 'c1',
    status: 'ACTIVE',
    q: 'foo',
    pacing_mode: 'EVEN',
    budget_min_micro: 100,
    owner_user_id: 'u1',
    country: 'US',
    limit: 50,
    offset: 50,
    sort: 'spend',
    order: 'desc',
    from: '2026-01-01T00:00:00.000Z',
    to: '2026-01-31T00:00:00.000Z',
  };

  const probe = buildCampaignListWidthProbeQuery(query);

  assert.equal(probe.customer_id, 'c1');
  assert.equal(probe.status, 'ACTIVE');
  assert.equal(probe.q, 'foo');
  assert.equal(probe.pacing_mode, 'EVEN');
  assert.equal(probe.budget_min_micro, 100);
  assert.equal(probe.owner_user_id, 'u1');
  assert.equal(probe.country, 'US');
  assert.equal(probe.limit, 100);
  assert.equal(probe.offset, 0);
  assert.equal(probe.sort, 'name');
  assert.equal(probe.order, 'asc');
  assert.equal(probe.from, undefined);
});

test('listResponseCoversWidthProbeDataset when all filtered rows are on the list response', () => {
  assert.equal(
    listResponseCoversWidthProbeDataset({
      total: 12,
      items: Array.from({ length: 12 }, (_, index) => ({ id: `c${index}` })),
    }),
    true,
  );
});

test('listResponseCoversWidthProbeDataset_holdout rejects duplicate row ids', () => {
  assert.equal(
    listResponseCoversWidthProbeDataset({
      total: 2,
      items: [{ id: 'c1' }, { id: 'c1' }],
    }),
    false,
  );
});

test('listResponseCoversWidthProbeDataset_holdout rejects empty row ids', () => {
  assert.equal(
    listResponseCoversWidthProbeDataset({
      total: 2,
      items: [{ id: 'c1' }, { id: '' }],
    }),
    false,
  );
});

test('listResponseCoversWidthProbeDataset false when paginated or over probe cap', () => {
  assert.equal(
    listResponseCoversWidthProbeDataset({
      total: 80,
      items: Array.from({ length: 50 }, (_, index) => ({ id: `c${index}` })),
    }),
    false,
  );
  assert.equal(
    listResponseCoversWidthProbeDataset({
      total: 150,
      items: Array.from({ length: 100 }, (_, index) => ({ id: `c${index}` })),
    }),
    false,
  );
  assert.equal(listResponseCoversWidthProbeDataset(undefined), false);
});

test('mergeCampaignIdsForMetricsBatch dedupes page and probe ids', () => {
  const a = '6ba7b810-9dad-11d1-80b4-00c04fd430c8';
  const b = '00000000-0000-0000-0000-000000000041';
  const c = '00000000-0000-0000-0000-000000000042';
  assert.deepEqual(mergeCampaignIdsForMetricsBatch([a, b], [b, c]), [a, b, c]);
});

test('mergeCampaignIdsForMetricsBatch_holdout skips non-uuid ids', () => {
  assert.deepEqual(
    mergeCampaignIdsForMetricsBatch(
      ['6ba7b810-9dad-11d1-80b4-00c04fd430c8', 'not-a-uuid'],
      ['00000000-0000-0000-0000-000000000041', ''],
    ),
    [
      '6ba7b810-9dad-11d1-80b4-00c04fd430c8',
      '00000000-0000-0000-0000-000000000041',
    ],
  );
});
