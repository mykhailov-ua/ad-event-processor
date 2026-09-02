import test from 'node:test';
import assert from 'node:assert/strict';

import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin } from '@/api/types';

import { computeCampaignListColumnWidths } from './campaign_list_column_widths.ts';

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

  assert.ok(widths.clicks > 80);
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
