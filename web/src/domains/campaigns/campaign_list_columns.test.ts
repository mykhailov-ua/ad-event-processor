import test from 'node:test';
import assert from 'node:assert/strict';

import {
  defaultCampaignListColumnPrefs,
  moveDataColumn,
  moveMiddleColumn,
  parseCampaignListColumnPrefs,
  setMiddleColumnVisible,
  visibleCampaignListColumns,
} from './campaign_list_columns.ts';

test('visibleCampaignListColumns uses campaigns list default set', () => {
  assert.deepEqual(visibleCampaignListColumns(defaultCampaignListColumnPrefs()), [
    'select',
    'id',
    'name',
    'source',
    'cr',
    'flows',
    'clicks',
    'conversions',
    'revenue',
    'cost',
    'profit',
    'roi',
    'group',
  ]);
});

test('parseCampaignListColumnPrefs merges unknown and duplicate middle columns', () => {
  const prefs = parseCampaignListColumnPrefs(
    JSON.stringify({
      middleOrder: ['group', 'source', 'unknown', 'source', 'clicks'],
      hidden: ['roi', 'bogus'],
    }),
  );

  assert.deepEqual(prefs.dataColumnOrder, [
    'name',
    'group',
    'source',
    'clicks',
    'cr',
    'flows',
    'conversions',
    'revenue',
    'cost',
    'profit',
    'roi',
  ]);
  assert.deepEqual(prefs.hidden, ['roi']);
  assert.deepEqual(visibleCampaignListColumns(prefs), [
    'select',
    'id',
    'name',
    'group',
    'source',
    'clicks',
    'cr',
    'flows',
    'conversions',
    'revenue',
    'cost',
    'profit',
  ]);
});

test('moveMiddleColumn reorders middle metrics', () => {
  const order = defaultCampaignListColumnPrefs().dataColumnOrder.filter(
    (columnId) => columnId !== 'name',
  );
  assert.deepEqual(moveMiddleColumn(order, 0, 2), [
    'cr',
    'flows',
    'source',
    'clicks',
    'conversions',
    'revenue',
    'cost',
    'profit',
    'roi',
    'group',
  ]);
});

test('moveDataColumn reorders name and middle columns', () => {
  const order = defaultCampaignListColumnPrefs().dataColumnOrder;
  assert.deepEqual(moveDataColumn(order, 'source', 'name'), [
    'source',
    'name',
    'cr',
    'flows',
    'clicks',
    'conversions',
    'revenue',
    'cost',
    'profit',
    'roi',
    'group',
  ]);
});

test('setMiddleColumnVisible toggles hidden set in canonical order', () => {
  const hidden = setMiddleColumnVisible([], 'cr', false);
  assert.deepEqual(hidden, ['cr']);
  assert.deepEqual(setMiddleColumnVisible(hidden, 'cr', true), []);
});
