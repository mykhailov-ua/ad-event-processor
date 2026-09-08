import assert from 'node:assert/strict';
import test from 'node:test';

import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/list/campaign_metrics_shared';

import { formatTableMoneyFromMicro, sumCampaignListTotals } from './campaign_list_format.ts';

test('formatTableMoneyFromMicro formats negative micro amounts', () => {
  assert.equal(formatTableMoneyFromMicro(-3_775_260_000).text, '-3,775.26');
});

test('sumCampaignListTotals_holdout prefers metrics batch micros over margin', () => {
  const totals = sumCampaignListTotals(
    [{ id: 'a' } as CampaignWithMoneyDisplay],
    {
      a: {
        clicks: 10,
        revenue_micro: 9_000_000,
        cost_micro: 4_000_000,
        profit_micro: 5_000_000,
      },
    },
    {
      a: {
        operator_margin_micro: 1,
        rtb_cost_micro: 2,
        advertiser_spend_micro: 3,
      },
    }
  );

  assert.equal(totals.revenueMicro, 9_000_000);
  assert.equal(totals.costMicro, 4_000_000);
  assert.equal(totals.profitMicro, 5_000_000);
});
