import assert from 'node:assert/strict';
import test from 'node:test';

import {
  campaignListTotalsFromMetricsTotals,
  formatTableMoneyFromMicro,
} from './campaign_list_format.ts';

test('campaignListTotalsFromMetricsTotals maps server metrics-totals response', () => {
  const totals = campaignListTotalsFromMetricsTotals({
    campaign_count: 1,
    flow_count: 2,
    margin_breach_count: 0,
    totals: {
      campaign_id: '00000000-0000-0000-0000-000000000001',
      clicks: 10,
      revenue_micro: 2_000_000,
      cost_micro: 1_000_000,
      profit_micro: 1_000_000,
    },
    from: '2026-01-01T00:00:00.000Z',
    to: '2026-01-02T00:00:00.000Z',
    stale: false,
  });
  assert.equal(totals.flows, 2);
  assert.equal(totals.clicks, 10);
  assert.equal(totals.revenueMicro, 2_000_000);
  assert.equal(totals.costMicro, 1_000_000);
  assert.equal(totals.profitMicro, 1_000_000);
});

test('formatTableMoneyFromMicro formats micro-units as USD', () => {
  assert.deepEqual(formatTableMoneyFromMicro(1_500_000), {
    text: '1.50',
    valUsd: 1.5,
    isZero: false,
  });
});
