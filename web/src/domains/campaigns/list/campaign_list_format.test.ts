import assert from 'node:assert/strict';
import test from 'node:test';

import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/list/campaign_metrics_shared';

import { sumCampaignListTotals } from './campaign_list_format.ts';

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
    },
  );

  assert.equal(totals.revenueMicro, 9_000_000);
  assert.equal(totals.costMicro, 4_000_000);
  assert.equal(totals.profitMicro, 5_000_000);
});
