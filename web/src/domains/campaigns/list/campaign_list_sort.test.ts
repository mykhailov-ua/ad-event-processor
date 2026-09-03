import test from 'node:test';
import assert from 'node:assert/strict';

import {
  campaignListSortNeedsMetricWindow,
  campaignListSortToApi,
  sortFieldForCampaignColumn,
} from './campaign_list_sort.ts';

test('campaignListSortToApi maps UI id column to updated_at', () => {
  assert.equal(campaignListSortToApi('id'), 'updated_at');
  assert.equal(campaignListSortToApi('updated_at'), 'updated_at');
});

test('campaignListSortToApi keeps metric and metadata sort fields', () => {
  assert.equal(campaignListSortToApi('leads'), 'leads');
  assert.equal(campaignListSortToApi('cost'), 'cost');
  assert.equal(campaignListSortToApi('status'), 'status');
});

test('campaignListSortNeedsMetricWindow matches backend metric window sorts', () => {
  assert.equal(campaignListSortNeedsMetricWindow('clicks'), true);
  assert.equal(campaignListSortNeedsMetricWindow('roi'), true);
  assert.equal(campaignListSortNeedsMetricWindow('name'), false);
  assert.equal(campaignListSortNeedsMetricWindow('budget_pct'), false);
});

test('sortFieldForCampaignColumn maps cost to cost not spend', () => {
  assert.equal(sortFieldForCampaignColumn('cost'), 'cost');
});
