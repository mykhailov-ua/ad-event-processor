import assert from 'node:assert/strict';
import test from 'node:test';

import { computeCampaignListSummary } from './campaign_list_summary.ts';

test('computeCampaignListSummary aggregates current page when nothing selected', () => {
  const summary = computeCampaignListSummary(
    [
      { id: 'a', flow_id: 'flow-1' } as never,
      { id: 'b' } as never,
    ],
    new Set(),
    {
      a: { clicks: 10, conversions: 2, stale: false },
      b: { clicks: 5, conversions: 1, stale: true },
    },
    {
      a: { operator_margin_micro: 100, rtb_cost_micro: 200, advertiser_spend_micro: 300 },
      b: { operator_margin_micro: -50, rtb_cost_micro: 100, advertiser_spend_micro: 50 },
    },
  );

  assert.equal(summary.scope, 'page');
  assert.equal(summary.rowCount, 2);
  assert.equal(summary.clicks, 15);
  assert.equal(summary.conversions, 3);
  assert.equal(summary.flows, 1);
  assert.equal(summary.staleCount, 1);
  assert.equal(summary.profitMicro, 50);
});

test('computeCampaignListSummary scopes to selected rows only', () => {
  const summary = computeCampaignListSummary(
    [
      { id: 'a' } as never,
      { id: 'b' } as never,
    ],
    new Set(['b']),
    {
      a: { clicks: 100, conversions: 10, stale: false },
      b: { clicks: 5, conversions: 1, stale: false },
    },
    {
      b: { operator_margin_micro: 20, rtb_cost_micro: 30, advertiser_spend_micro: 50 },
    },
  );

  assert.equal(summary.scope, 'selection');
  assert.equal(summary.rowCount, 1);
  assert.equal(summary.clicks, 5);
  assert.equal(summary.conversions, 1);
});
