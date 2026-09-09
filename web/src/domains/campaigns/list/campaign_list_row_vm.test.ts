import assert from 'node:assert/strict';
import test from 'node:test';

import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin } from '@/api/types';
import { adminMetricNegativeClass } from '@/lib/admin_metric_tone';
import { seedDeterministicUuid } from '@/lib/uuid.ts';
import { buildCampaignRowVm } from '@/domains/campaigns/list/campaign_list_row_vm.ts';

const baseCampaign = {
  id: seedDeterministicUuid('campaign', 1),
  name: 'Alpha',
  status: 'ACTIVE',
  budget_limit: '100.00',
  current_spend: '25.00',
  current_spend_display: '25.00',
  customer_id: seedDeterministicUuid('customer', 1),
  pacing_mode: 'even',
  daily_budget: '0.00',
  timezone: 'UTC',
  freq_limit: 0,
  freq_window: 0,
  target_countries: ['US'],
  daypart_hours: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
} satisfies Campaign;

const baseMargin: CampaignMargin = {
  campaign_id: baseCampaign.id,
  window_start: '2026-01-01T00:00:00Z',
  window_hours: 24,
  advertiser_spend_micro: 900_000,
  rtb_cost_micro: 600_000,
  operator_margin_micro: 100_000,
  publisher_payout_micro: 432_000,
  cost_over_revenue_limit: 0,
  threshold_bps: 500,
  margin_breach: false,
};

test('buildCampaignRowVm_holdout does not derive KPI rates from raw counts', () => {
  const metrics: CampaignListMetrics = {
    impressions: 10_000,
    clicks: 500,
    conversions: 40,
    blocks: 25,
    bots: 10,
  };

  const vm = buildCampaignRowVm(baseCampaign, metrics, baseMargin, {}, {});

  assert.equal(vm.ctr, null);
  assert.equal(vm.roi.text, '-');
  assert.equal(vm.epc.text, '0.00');
  assert.equal(vm.cpm, null);
});

test('buildCampaignRowVm maps server derived fields', () => {
  const metrics: CampaignListMetrics = {
    impressions: 10_000,
    clicks: 500,
    conversions: 40,
    ctr_pct: 5,
    roi_pct: 100 / 6,
    epc_micro: 2_000_000,
    cpm_usd: '0.06',
    cost_micro: 600_000,
    revenue_micro: 1_000_000,
    profit_micro: 400_000,
  };

  const vm = buildCampaignRowVm(
    { ...baseCampaign, budget_used_pct: 25 },
    metrics,
    baseMargin,
    {},
    {}
  );

  assert.equal(vm.ctr?.valPct, 5);
  assert.equal(vm.roi.text, '+16.67%');
  assert.equal(vm.epc.text, '2.00');
  assert.equal(vm.budgetPct, 25);
  assert.equal(vm.cpm, '0.06');
  assert.equal(vm.rowAlert, 'none');
});

test('buildCampaignRowVm_holdout formats negative server profit and roi', () => {
  const metrics: CampaignListMetrics = {
    revenue_micro: 2_670_000,
    cost_micro: 3_777_930_000,
    profit_micro: -3_775_260_000,
    roi_pct: -99.93,
  };
  const vm = buildCampaignRowVm(baseCampaign, metrics, undefined, {}, {});

  assert.equal(vm.profit.text, '-3,775.26');
  assert.equal(vm.roi.text, '-99.93%');
  assert.equal(vm.profitToneClass, adminMetricNegativeClass);
  assert.equal(vm.roiToneClass, adminMetricNegativeClass);
});

test('buildCampaignRowVm sets row alert for high budget usage', () => {
  const vm = buildCampaignRowVm(
    { ...baseCampaign, budget_used_pct: 92 },
    undefined,
    undefined,
    {},
    {}
  );

  assert.equal(vm.rowAlert, 'warning');
  assert.equal(vm.rowAccent, 'none');
});

test('buildCampaignRowVm_holdout shows zero economics before metrics batch', () => {
  const vm = buildCampaignRowVm(baseCampaign, undefined, undefined, {}, {});

  assert.equal(vm.revenue.text, '0.00');
  assert.equal(vm.cost.text, '0.00');
  assert.equal(vm.roi.text, '-');
});

test('buildCampaignRowVm_holdout uses server profit_micro without client recalculation', () => {
  const metrics: CampaignListMetrics = {
    revenue_micro: 2_670_000,
    cost_micro: 3_777_930_000,
    profit_micro: 0,
    roi_pct: 0,
  };
  const vm = buildCampaignRowVm(baseCampaign, metrics, undefined, {}, {});

  assert.equal(vm.profit.text, '0.00');
  assert.equal(vm.roi.text, '0.00%');
});

test('buildCampaignRowVm uses profit_micro for profit tone', () => {
  const metrics: CampaignListMetrics = { profit_micro: 500_000 };
  const margin: CampaignMargin = { ...baseMargin, operator_margin_micro: 0 };
  const vm = buildCampaignRowVm(baseCampaign, metrics, margin, {}, {});

  assert.match(vm.profitToneClass, /text-admin-positive/);
});
