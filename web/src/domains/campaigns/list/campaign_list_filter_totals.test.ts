import test from 'node:test';
import assert from 'node:assert/strict';

import { campaignListFilterTotalsFromApi } from './campaign_list_filter_totals.ts';

test('campaignListFilterTotalsFromApi maps server totals row', () => {
  const view = campaignListFilterTotalsFromApi({
    campaign_count: 12,
    flow_count: 4,
    margin_breach_count: 2,
    from: '2026-01-01T00:00:00Z',
    to: '2026-01-08T00:00:00Z',
    stale: false,
    totals: {
      campaign_id: '',
      impressions: 1000,
      clicks: 200,
      conversions: 20,
      blocks: 5,
      leads_raw: 25,
      hold_leads: 3,
      rejected_leads: 2,
      revenue_micro: 5_000_000,
      cost_micro: 3_000_000,
      profit_micro: 2_000_000,
    },
  });

  assert.ok(view);
  assert.equal(view.totals.flows, 4);
  assert.equal(view.totals.clicks, 200);
  assert.equal(view.funnelTotals.rawLeads, 25);
  assert.equal(view.campaignCount, 12);
  assert.equal(view.marginBreachCount, 2);
});

test('campaignListFilterTotalsFromApi returns undefined without response', () => {
  assert.equal(campaignListFilterTotalsFromApi(undefined), undefined);
});
