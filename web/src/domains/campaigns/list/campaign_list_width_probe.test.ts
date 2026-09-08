import test from 'node:test';
import assert from 'node:assert/strict';

import type { CampaignListQuery } from '@/api/types';

import { seedDeterministicUuid } from '@/lib/uuid.ts';

import {
  buildCampaignListWidthProbeQuery,
  listResponseCoversWidthProbeDataset,
  mergeCampaignIdsForMetricsBatch,
  shouldFetchCampaignListWidthProbe,
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

test('listResponseCoversWidthProbeDataset true for empty filter result', () => {
  assert.equal(
    listResponseCoversWidthProbeDataset({
      total: 0,
      items: [],
    }),
    true
  );
});

test('listResponseCoversWidthProbeDataset when all filtered rows are on the list response', () => {
  assert.equal(
    listResponseCoversWidthProbeDataset({
      total: 12,
      items: Array.from({ length: 12 }, (_, index) => ({ id: `c${index}` })),
    }),
    true
  );
});

test('listResponseCoversWidthProbeDataset_holdout rejects duplicate row ids', () => {
  assert.equal(
    listResponseCoversWidthProbeDataset({
      total: 2,
      items: [{ id: 'c1' }, { id: 'c1' }],
    }),
    false
  );
});

test('listResponseCoversWidthProbeDataset_holdout rejects empty row ids', () => {
  assert.equal(
    listResponseCoversWidthProbeDataset({
      total: 2,
      items: [{ id: 'c1' }, { id: '' }],
    }),
    false
  );
});

test('listResponseCoversWidthProbeDataset false when paginated or over probe cap', () => {
  assert.equal(
    listResponseCoversWidthProbeDataset({
      total: 80,
      items: Array.from({ length: 50 }, (_, index) => ({ id: `c${index}` })),
    }),
    false
  );
  assert.equal(
    listResponseCoversWidthProbeDataset({
      total: 150,
      items: Array.from({ length: 100 }, (_, index) => ({ id: `c${index}` })),
    }),
    false
  );
  assert.equal(listResponseCoversWidthProbeDataset(undefined), false);
});

test('shouldFetchCampaignListWidthProbe waits for main list snapshot', () => {
  assert.equal(shouldFetchCampaignListWidthProbe(undefined), false);
});

test('shouldFetchCampaignListWidthProbe_holdout skips covered and empty datasets', () => {
  assert.equal(
    shouldFetchCampaignListWidthProbe({
      total: 0,
      items: [],
    }),
    false
  );
  assert.equal(
    shouldFetchCampaignListWidthProbe({
      total: 3,
      items: [{ id: 'c1' }, { id: 'c2' }, { id: 'c3' }],
    }),
    false
  );
});

test('shouldFetchCampaignListWidthProbe true when paginated', () => {
  assert.equal(
    shouldFetchCampaignListWidthProbe({
      total: 80,
      items: Array.from({ length: 50 }, (_, index) => ({ id: `c${index}` })),
    }),
    true
  );
});

test('mergeCampaignIdsForMetricsBatch dedupes page and probe ids', () => {
  const a = '6ba7b810-9dad-11d1-80b4-00c04fd430c8';
  const b = seedDeterministicUuid('campaign', 41);
  const c = seedDeterministicUuid('campaign', 42);
  assert.deepEqual(mergeCampaignIdsForMetricsBatch([a, b], [b, c]), [a, b, c]);
});

test('mergeCampaignIdsForMetricsBatch_holdout skips non-uuid ids', () => {
  const probeId = seedDeterministicUuid('campaign', 41);
  assert.deepEqual(
    mergeCampaignIdsForMetricsBatch(
      ['6ba7b810-9dad-11d1-80b4-00c04fd430c8', 'not-a-uuid'],
      [probeId, '']
    ),
    ['6ba7b810-9dad-11d1-80b4-00c04fd430c8', probeId]
  );
});
