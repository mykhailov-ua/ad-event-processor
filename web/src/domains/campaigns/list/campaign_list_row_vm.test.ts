import assert from 'node:assert/strict';
import test from 'node:test';

import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin } from '@/api/types';
import { buildCampaignRowVm } from '@/domains/campaigns/list/campaign_list_row_vm.ts';

const baseCampaign = {
  id: '00000000-0000-4000-8000-000000000001',
  name: 'Alpha',
  status: 'ACTIVE',
  budget_limit: '100.00',
  current_spend: '25.00',
  customer_id: '00000000-0000-4000-8000-000000000010',
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

  const vm = buildCampaignRowVm(baseCampaign, metrics, baseMargin, {}, {}, false, false);

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
  };

  const vm = buildCampaignRowVm(
    { ...baseCampaign, budget_used_pct: 25 },
    metrics,
    baseMargin,
    {},
    {},
    false,
    false,
  );

  assert.equal(vm.ctr?.valPct, 5);
  assert.equal(vm.roi.text, '+16.67%');
  assert.equal(vm.epc.text, '2.00');
  assert.equal(vm.budgetPct, 25);
  assert.equal(vm.cpm, '0.06');
});
