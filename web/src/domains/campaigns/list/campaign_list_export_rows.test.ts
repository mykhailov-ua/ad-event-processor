import assert from 'node:assert/strict';
import test from 'node:test';

import type { CampaignListMetrics } from '@/api/campaigns_api.ts';
import type { Campaign, CampaignMargin } from '@/api/types.ts';
import { seedDeterministicUuid } from '@/lib/uuid.ts';
import { defaultCampaignListExportDataColumns } from './campaign_list_columns.ts';
import { formatCampaignListExportToast } from './campaign_list_export_toast.ts';
import {
  buildCampaignListExportCsv,
  buildCampaignListExportRows,
  campaignListExportCellValue,
} from './campaign_list_export_rows.ts';
import { buildCampaignRowVm } from './campaign_list_row_vm.ts';

const baseCampaign = {
  id: seedDeterministicUuid('campaign', 1),
  name: 'Test campaign',
  status: 'ACTIVE',
  budget_limit: '100.00',
  current_spend: '25.00',
  current_spend_display: '25.00',
  customer_id: seedDeterministicUuid('customer', 1),
  pacing_mode: 'even',
  daily_budget: '0.00',
  timezone: 'UTC',
} as Campaign;

test('buildCampaignListExportCsv matches visible column labels and row VM values', () => {
  const columns = defaultCampaignListExportDataColumns();
  const metrics: CampaignListMetrics = { clicks: 42, revenue_micro: 1_000_000 };
  const margin: CampaignMargin = {
    campaign_id: baseCampaign.id,
    window_start: new Date(0).toISOString(),
    window_hours: 24,
    advertiser_spend_micro: 0,
    rtb_cost_micro: 0,
    operator_margin_micro: 0,
    publisher_payout_micro: 0,
    cost_over_revenue_limit: 0,
    threshold_bps: 0,
    margin_breach: false,
  };
  const vm = buildCampaignRowVm(baseCampaign, metrics, margin, {}, {});
  const csv = buildCampaignListExportCsv(columns, [{ campaign: baseCampaign, vm }]);
  assert.match(csv, /^ID,Name,/);
  assert.match(csv, /Test campaign/);
});

test('campaignListExportCellValue maps id and name columns', () => {
  const metrics: CampaignListMetrics = { clicks: 1 };
  const vm = buildCampaignRowVm(baseCampaign, metrics, undefined, {}, {});
  const columns = defaultCampaignListExportDataColumns();
  const idColumn = columns.find((column) => column === 'id');
  const nameColumn = columns.find((column) => column === 'name');
  assert.ok(idColumn);
  assert.ok(nameColumn);
  assert.ok(campaignListExportCellValue(idColumn!, vm).length > 0);
  assert.equal(campaignListExportCellValue(nameColumn!, vm), baseCampaign.name);
});

test('buildCampaignListExportRows maps metrics batch per campaign id', () => {
  const rows = buildCampaignListExportRows(
    [baseCampaign],
    defaultCampaignListExportDataColumns(),
    { [baseCampaign.id]: { clicks: 9 } },
    {},
    {},
    {}
  );
  assert.equal(rows.length, 1);
  assert.equal(rows[0]?.vm.clicks.text, '9');
});

test('formatCampaignListExportToast mentions partial export cap when truncated', () => {
  const message = formatCampaignListExportToast(10, 100, true, 'CSV');
  assert.match(message, /Exported 10 of 100 campaign\(s\)/);
  assert.match(message, /max 5,000/);
  assert.doesNotMatch(formatCampaignListExportToast(10, 10, false, 'CSV'), / of /);
});
