import test from 'node:test';
import assert from 'node:assert/strict';

import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin } from '@/api/types';

import { computeCampaignListColumnWidths, campaignListMiddleCellText, defaultCampaignListColumnWidths } from './campaign_list_column_widths.ts';

test('defaultCampaignListColumnWidths fits header labels with tools gutter', () => {
  const widths = defaultCampaignListColumnWidths(['select', 'unique_clicks', 'lp_views']);

  assert.equal(widths.select, 48);
  assert.ok(widths.unique_clicks >= 150, `unique_clicks too narrow: ${widths.unique_clicks}`);
  assert.ok(widths.lp_views >= 120, `lp_views too narrow: ${widths.lp_views}`);
});

test('computeCampaignListColumnWidths uses dataset max not current page only', () => {
  const items: Campaign[] = [
    { id: 'a', name: 'Short', customer_id: 'c1', status: 'ACTIVE' } as Campaign,
    { id: 'b', name: 'Short', customer_id: 'c1', status: 'ACTIVE' } as Campaign,
  ];

  const metricsById: Record<string, CampaignListMetrics> = {
    a: { clicks: 12, conversions: 1 },
    b: { clicks: 1_234_567, conversions: 9 },
  };

  const widths = computeCampaignListColumnWidths({
    columns: ['select', 'id', 'name', 'clicks'],
    items,
    metricsById,
    marginsById: {},
    customerNameById: { c1: 'Buyer' },
  });

  assert.ok(widths.clicks >= 75);
});

test('computeCampaignListColumnWidths keeps widths stable when sort order changes', () => {
  const items: Campaign[] = [
    { id: 'a', name: 'Alpha', customer_id: 'c1', status: 'ACTIVE' } as Campaign,
    { id: 'b', name: 'Beta', customer_id: 'c1', status: 'ACTIVE' } as Campaign,
  ];
  const metricsById: Record<string, CampaignListMetrics> = {
    a: { clicks: 999_999, conversions: 1 },
    b: { clicks: 1, conversions: 0 },
  };

  const forward = computeCampaignListColumnWidths({
    columns: ['clicks'],
    items,
    metricsById,
    marginsById: {},
    customerNameById: {},
  });
  const reversed = computeCampaignListColumnWidths({
    columns: ['clicks'],
    items: [...items].reverse(),
    metricsById,
    marginsById: {},
    customerNameById: {},
  });

  assert.equal(forward.clicks, reversed.clicks);
});

test('campaignListMiddleCellText uses server derived epc micro not client math', () => {
  const campaign = { id: 'a', name: 'A', customer_id: 'c1', status: 'ACTIVE' } as Campaign;
  const metrics: CampaignListMetrics = {
    clicks: 100,
    revenue_micro: 500_000_000,
    epc_micro: 5_000_000,
  };

  const text = campaignListMiddleCellText('epc', campaign, metrics, undefined, { c1: 'Buyer' });

  assert.equal(text, '5.00');
});

test('computeCampaignListColumnWidths uses filter totals label for footer probe', () => {
  const items: Campaign[] = [
    { id: 'a', name: 'Short', customer_id: 'c1', status: 'ACTIVE' } as Campaign,
  ];

  const widths = computeCampaignListColumnWidths({
    columns: ['name'],
    items,
    metricsById: { a: { clicks: 1 } },
    marginsById: {},
    customerNameById: {},
    filterTotals: {
      totals: {
        flows: 0,
        clicks: 0,
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
      campaignCount: 10,
      stale: false,
    },
  });

  assert.ok(widths.name >= 90);
});
