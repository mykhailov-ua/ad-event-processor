import test from 'node:test';
import assert from 'node:assert/strict';

import {
  defaultCampaignListColumnPrefs,
  moveMiddleColumn,
  parseCampaignListColumnPrefs,
  setMiddleColumnVisible,
  visibleCampaignListColumns,
} from './campaign_list_columns.ts';

test('visibleCampaignListColumns uses level-1 default set', () => {
  assert.deepEqual(visibleCampaignListColumns(defaultCampaignListColumnPrefs()), [
    'name',
    'status',
    'budget',
    'spend',
    'clicks',
    'conversions',
    'actions',
  ]);
});

test('parseCampaignListColumnPrefs merges unknown and duplicate middle columns', () => {
  const prefs = parseCampaignListColumnPrefs(
    JSON.stringify({
      middleOrder: ['updated', 'status', 'unknown', 'status', 'budget'],
      hidden: ['customer', 'bogus'],
    }),
  );

  assert.deepEqual(prefs.middleOrder, [
    'updated',
    'status',
    'budget',
    'spend',
    'clicks',
    'conversions',
    'pacing',
    'customer',
  ]);
  assert.deepEqual(prefs.hidden, ['customer']);
  assert.deepEqual(visibleCampaignListColumns(prefs), [
    'name',
    'updated',
    'status',
    'budget',
    'spend',
    'clicks',
    'conversions',
    'pacing',
    'actions',
  ]);
});

test('moveMiddleColumn reorders middle metrics', () => {
  const order = defaultCampaignListColumnPrefs().middleOrder;
  assert.deepEqual(moveMiddleColumn(order, 0, 2), [
    'budget',
    'spend',
    'status',
    'clicks',
    'conversions',
    'pacing',
    'customer',
    'updated',
  ]);
});

test('setMiddleColumnVisible toggles hidden set in canonical order', () => {
  const hidden = setMiddleColumnVisible([], 'pacing', false);
  assert.deepEqual(hidden, ['pacing']);
  assert.deepEqual(setMiddleColumnVisible(hidden, 'pacing', true), []);
});
