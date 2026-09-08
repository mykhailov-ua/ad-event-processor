import test from 'node:test';
import assert from 'node:assert/strict';

import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin } from '@/api/types';

import {
  computeCampaignListColumnWidths,
  campaignListMiddleCellText,
  defaultCampaignListColumnWidths,
} from './campaign_list_column_widths.ts';
import { CAMPAIGN_LIST_STATUS_COLUMN_WIDTH_PX } from './campaign_list_columns.ts';

test('defaultCampaignListColumnWidths fits header labels with drag grip', () => {
  const widths = defaultCampaignListColumnWidths(['select', 'unique_clicks', 'lp_views']);

  assert.equal(widths.select, 48);
  assert.ok(widths.unique_clicks >= 140, `unique_clicks too narrow: ${widths.unique_clicks}`);
  assert.ok(widths.lp_views >= 110, `lp_views too narrow: ${widths.lp_views}`);
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

test('computeCampaignListColumnWidths fits eight-digit id with copy control', () => {
  const items: Campaign[] = [
    {
      id: '00000000-0000-7000-8000-000000000001',
      display_id: '47501610',
      name: 'Campaign',
      customer_id: 'c1',
      status: 'ACTIVE',
    } as Campaign,
  ];

  const widths = computeCampaignListColumnWidths({
    columns: ['id'],
    items,
    metricsById: {},
    marginsById: {},
    customerNameById: {},
  });

  assert.ok(widths.id >= 100, `id column too narrow for display id: ${widths.id}`);
  assert.ok(widths.id > 96, `id column still capped at legacy 96px max: ${widths.id}`);
});

test('computeCampaignListColumnWidths sizes countries column for three flags and overflow menu', () => {
  const items: Campaign[] = [
    {
      id: 'a',
      name: 'Multi GEO',
      customer_id: 'c1',
      status: 'ACTIVE',
      target_countries: ['US', 'CA', 'DE', 'FR', 'GB'],
    } as Campaign,
  ];

  const widths = computeCampaignListColumnWidths({
    columns: ['countries'],
    items,
    metricsById: {},
    marginsById: {},
    customerNameById: {},
  });

  assert.ok(widths.countries >= 116, `countries column too narrow: ${widths.countries}`);
});

test('computeCampaignListColumnWidths keeps status column fixed for Exhausted badge', () => {
  const items: Campaign[] = [
    { id: 'a', name: 'Campaign', customer_id: 'c1', status: 'EXHAUSTED' } as Campaign,
  ];

  const widths = computeCampaignListColumnWidths({
    columns: ['status'],
    items,
    metricsById: {},
    marginsById: {},
    customerNameById: {},
  });

  assert.equal(widths.status, CAMPAIGN_LIST_STATUS_COLUMN_WIDTH_PX);
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
