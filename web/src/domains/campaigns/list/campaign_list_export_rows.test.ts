import assert from 'node:assert/strict';
import test from 'node:test';

import type { CampaignListMetrics } from '@/api/campaigns_api.ts';
import type { Campaign, CampaignMargin } from '@/api/types.ts';
import { defaultCampaignListColumnPrefs, visibleCampaignListColumns } from './campaign_list_columns.ts';
import { formatCampaignListExportToast } from './campaign_list_export_toast.ts';
import {
  buildCampaignListExportCsv,
  buildCampaignListExportRows,
  campaignListExportCellValue,
  exportableCampaignListColumns,
} from './campaign_list_export_rows.ts';
import { buildCampaignRowVm } from './campaign_list_row_vm.ts';

const baseCampaign = {
  id: '00000000-0000-4000-8000-000000000001',
  name: 'Test campaign',
  status: 'ACTIVE',
  budget_limit: '100.00',
  current_spend: '25.00',
  current_spend_display: '25.00',
  customer_id: '00000000-0000-4000-8000-000000000010',
  pacing_mode: 'even',
  daily_budget: '0.00',
  timezone: 'UTC',
  freq_limit: 0,
  freq_window: 0,
  target_countries: ['US', 'CA'],
  daypart_hours: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
} satisfies Campaign;

const metrics: CampaignListMetrics = {
  impressions: 4_000,
  clicks: 120,
  conversions: 10,
  blocks: 0,
  leads_raw: 8,
  hold_leads: 1,
  rejected_leads: 0,
  lp_clicks: 40,
  lp_views: 80,
  bots: 0,
  unique_clicks: 100,
  stale: false,
  revenue_micro: 5_000_000,
  cost_micro: 2_500_000,
  profit_micro: 2_500_000,
  ctr_pct: 3,
  cr_pct: 2,
  roi_pct: 100,
};

const margin: CampaignMargin = {
  campaign_id: baseCampaign.id,
  window_start: '2026-01-01T00:00:00Z',
  window_hours: 24,
  advertiser_spend_micro: 900_000,
  rtb_cost_micro: 600_000,
  operator_margin_micro: 100_000,
  publisher_payout_micro: 432_000,
  cost_over_revenue_limit: 0,
  threshold_bps: 500,
  margin_breach: true,
};

test('exportableCampaignListColumns drops selection column', () => {
  const columns = exportableCampaignListColumns(visibleCampaignListColumns(defaultCampaignListColumnPrefs()));
  assert.equal(columns.includes('select' as never), false);
  assert.equal(columns.includes('name'), true);
  assert.equal(columns.includes('status'), true);
});

test('buildCampaignListExportCsv matches visible column labels and row VM values', () => {
  const columns = ['id', 'name', 'status', 'clicks', 'revenue'] as const;
  const vm = buildCampaignRowVm(
    baseCampaign,
    metrics,
    margin,
    { [baseCampaign.customer_id]: 'Acme' },
    {},
    false,
  );
  const csv = buildCampaignListExportCsv(columns, [{ campaign: baseCampaign, vm }]);

  assert.match(csv, /^ID,Name,Status,Clicks,Revenue/);
  assert.match(csv, /"Test campaign"/);
  assert.match(csv, /"Active"/);
  assert.match(csv, /"120"/);
  assert.match(csv, /"5\.00"/);
});

test('campaignListExportCellValue_holdout uses VM not raw campaign counters', () => {
  const vm = buildCampaignRowVm(baseCampaign, metrics, margin, {}, {}, false);
  assert.equal(campaignListExportCellValue('clicks', vm), '120');
  assert.notEqual(campaignListExportCellValue('clicks', vm), String(baseCampaign.current_spend));
});

test('buildCampaignListExportRows maps metrics batch per campaign id', () => {
  const rows = buildCampaignListExportRows(
    [baseCampaign],
    ['clicks', 'roi'],
    { [baseCampaign.id]: metrics },
    { [baseCampaign.id]: margin },
    {},
    {},
  );
  assert.equal(rows.length, 1);
  assert.equal(campaignListExportCellValue('clicks', rows[0]!.vm), '120');
  assert.match(campaignListExportCellValue('roi', rows[0]!.vm), /100/);
});

test('formatCampaignListExportToast reports truncation cap', () => {
  const message = formatCampaignListExportToast(5000, 12_345, true, 'CSV');
  assert.match(message, /5,000 of 12,345/);
  assert.match(message, /max 5,000/);
});
