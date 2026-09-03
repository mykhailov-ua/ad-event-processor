import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildDevMockCampaignMetrics,
  compareDevMockCampaignMetricSort,
  syncDevMockCampaignLeadsRaw,
} from './campaign_metrics.ts';
import { devMockListCampaigns } from './campaign_list.ts';
import { createDevMockCampaigns } from './fixtures.ts';

test('compareDevMockCampaignMetricSort orders by clicks desc', () => {
  const left = buildDevMockCampaignMetrics('a', 1, '2026-02-01T00:00:00.000Z', '2026-02-08T00:00:00.000Z');
  const right = buildDevMockCampaignMetrics('b', 2, '2026-02-01T00:00:00.000Z', '2026-02-08T00:00:00.000Z');
  assert.ok(compareDevMockCampaignMetricSort('clicks', left, right) < 0);
});

test('syncDevMockCampaignLeadsRaw_holdout matches Go syncLeadsRaw fallback', () => {
  const metrics = buildDevMockCampaignMetrics('a', 1, '2026-02-01T00:00:00.000Z', '2026-02-08T00:00:00.000Z');
  metrics.leads_raw = 0;
  metrics.conversions = 4;
  metrics.hold_leads = 0;
  metrics.rejected_leads = 0;
  syncDevMockCampaignLeadsRaw(metrics);
  assert.equal(metrics.leads_raw, 4);
});

test('devMockListCampaigns rejects metric sort without from/to', () => {
  const result = devMockListCampaigns(
    new URL('http://localhost/api/v1/campaigns?sort=clicks&order=desc'),
    createDevMockCampaigns(),
  );
  assert.equal(result.status, 400);
});

test('devMockListCampaigns sorts by leads when stats window is present', () => {
  const result = devMockListCampaigns(
    new URL(
      'http://localhost/api/v1/campaigns?sort=leads&order=desc&from=2026-02-01T00:00:00.000Z&to=2026-02-08T00:00:00.000Z&limit=50',
    ),
    createDevMockCampaigns(),
  );
  assert.equal(result.status, 200);
  const body = result.body as { items: { id: string }[] };
  assert.ok(body.items.length > 1);
  const ids = body.items.map((item) => item.id);
  const resorted = devMockListCampaigns(
    new URL(
      'http://localhost/api/v1/campaigns?sort=leads&order=asc&from=2026-02-01T00:00:00.000Z&to=2026-02-08T00:00:00.000Z&limit=50',
    ),
    createDevMockCampaigns(),
  );
  const ascIds = (resorted.body as { items: { id: string }[] }).items.map((item) => item.id);
  assert.notDeepEqual(ids, ascIds);
});
