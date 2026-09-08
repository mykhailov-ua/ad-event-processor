import test from 'node:test';
import assert from 'node:assert/strict';

import {
  campaignListSortNeedsMetricWindow,
  campaignListSortStartsDesc,
  campaignListSortShowsAscIcon,
  campaignListSortToApi,
  sortFieldForCampaignColumn,
} from '@/domains/campaigns/list/campaign_list_sort';

test('campaignListSortToApi maps UI id column to id', () => {
  assert.equal(campaignListSortToApi('id'), 'id');
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

test('campaignListSortStartsDesc defaults metrics to highest-first', () => {
  assert.equal(campaignListSortStartsDesc('roi'), true);
  assert.equal(campaignListSortStartsDesc('name'), false);
  assert.equal(campaignListSortStartsDesc('status'), false);
});

test('campaignListSortShowsAscIcon points up for top metric values', () => {
  assert.equal(campaignListSortShowsAscIcon('desc', true), true);
  assert.equal(campaignListSortShowsAscIcon('asc', true), false);
  assert.equal(campaignListSortShowsAscIcon('asc', false), true);
  assert.equal(campaignListSortShowsAscIcon('desc', false), false);
});
