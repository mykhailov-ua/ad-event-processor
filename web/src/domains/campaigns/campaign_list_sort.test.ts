import test from 'node:test';
import assert from 'node:assert/strict';

import { sortCampaignListItemsClient } from './campaign_list_sort.ts';

test('sortCampaignListItemsClient orders metric columns on current page', () => {
  const items = [
    { id: 'a', name: 'Alpha', customer_id: 'c1', status: 'ACTIVE', budget_limit: '0', current_spend: '0' },
    { id: 'b', name: 'Beta', customer_id: 'c1', status: 'ACTIVE', budget_limit: '0', current_spend: '0' },
  ];
  const metricsById = {
    a: { clicks: 10, conversions: 1, impressions: 20 },
    b: { clicks: 5, conversions: 3, impressions: 8 },
  };

  const sorted = sortCampaignListItemsClient(items, 'conversions', 'desc', metricsById, {});
  assert.deepEqual(sorted.map((item) => item.id), ['b', 'a']);
});
