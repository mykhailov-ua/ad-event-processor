import assert from 'node:assert/strict';
import test from 'node:test';

import { computeCampaignListSummary, resolveCampaignListSummary } from './campaign_list_summary.ts';

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

test('computeCampaignListSummary_holdout prefers metrics batch micros over margin', () => {
  const summary = computeCampaignListSummary(
    [{ id: 'a' } as never],
    new Set(),
    {
      a: {
        clicks: 1,
        conversions: 0,
        revenue_micro: 2_000_000,
        cost_micro: 1_000_000,
        profit_micro: 1_000_000,
      },
    },
    {
      a: {
        operator_margin_micro: 99,
        rtb_cost_micro: 99,
        advertiser_spend_micro: 99,
      },
    },
  );

  assert.equal(summary.revenueMicro, 2_000_000);
  assert.equal(summary.costMicro, 1_000_000);
  assert.equal(summary.profitMicro, 1_000_000);
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

test('resolveCampaignListSummary uses filter totals when nothing selected', () => {
  const summary = resolveCampaignListSummary(
    [{ id: 'a' } as never],
    new Set(),
    { a: { clicks: 1, conversions: 0 } },
    {},
    {
      totals: {
        flows: 2,
        clicks: 500,
        impressions: 0,
        blocks: 0,
        conversions: 40,
        revenueMicro: 10_000_000,
        costMicro: 6_000_000,
        profitMicro: 4_000_000,
      },
      funnelTotals: {
        rawLeads: 50,
        approved: 40,
        hold: 5,
        rejected: 5,
        lpClicks: 0,
        lpViews: 0,
        bots: 0,
      },
      campaignCount: 12,
      marginBreachCount: 3,
      stale: false,
    },
  );

  assert.equal(summary.scope, 'filter');
  assert.equal(summary.rowCount, 12);
  assert.equal(summary.clicks, 500);
  assert.equal(summary.profitMicro, 4_000_000);
  assert.equal(summary.marginBreachCount, 3);
});

test('resolveCampaignListSummary_holdout prefers selection over filter totals', () => {
  const summary = resolveCampaignListSummary(
    [{ id: 'a' } as never],
    new Set(['a']),
    { a: { clicks: 3, conversions: 1 } },
    { a: { operator_margin_micro: 10, rtb_cost_micro: 20, advertiser_spend_micro: 30 } },
    {
      totals: {
        flows: 0,
        clicks: 999,
        impressions: 0,
        blocks: 0,
        conversions: 0,
        revenueMicro: 0,
        costMicro: 0,
        profitMicro: 0,
      },
      funnelTotals: {
        rawLeads: 0,
        approved: 0,
        hold: 0,
        rejected: 0,
        lpClicks: 0,
        lpViews: 0,
        bots: 0,
      },
      campaignCount: 99,
      marginBreachCount: 0,
      stale: false,
    },
  );

  assert.equal(summary.scope, 'selection');
  assert.equal(summary.clicks, 3);
});
