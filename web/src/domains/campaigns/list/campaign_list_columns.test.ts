import test from 'node:test';
import assert from 'node:assert/strict';

import {
  CAMPAIGN_LIST_DEFAULT_HIDDEN,
  clampCampaignListColumnWidthPx,
  defaultCampaignListColumnPrefs,
  isCampaignListColumnResizable,
  moveDataColumn,
  moveMiddleColumn,
  normalizeColumnWidthPx,
  parseCampaignListColumnPrefs,
  setMiddleColumnVisible,
  visibleCampaignListColumns,
} from './campaign_list_columns.ts';

test('clampCampaignListColumnWidthPx caps oversized name column', () => {
  assert.equal(clampCampaignListColumnWidthPx('name', 2000), 320);
  assert.equal(clampCampaignListColumnWidthPx('clicks', 12), 60);
});

test('normalizeColumnWidthPx clamps saved widths from localStorage', () => {
  const widths = normalizeColumnWidthPx({ name: 4096, clicks: 80, roi: 900 });
  assert.equal(widths.name, 640);
  assert.equal(widths.clicks, 80);
  assert.equal(widths.roi, 480);
});

test('visibleCampaignListColumns hides legacy placeholder columns by default', () => {
  const visible = visibleCampaignListColumns(defaultCampaignListColumnPrefs());
  for (const hiddenId of CAMPAIGN_LIST_DEFAULT_HIDDEN) {
    assert.equal(visible.includes(hiddenId), false, `expected hidden: ${hiddenId}`);
  }
  assert.equal(visible.includes('status'), true);
  assert.equal(visible.includes('approved'), true);
  assert.equal(visible.includes('ctr'), true);
});

test('visibleCampaignListColumns pins select id name then reorderable metrics', () => {
  const visible = visibleCampaignListColumns(defaultCampaignListColumnPrefs());
  assert.deepEqual(visible.slice(0, 3), ['select', 'id', 'name']);
  assert.equal(visible[3], 'status');
  assert.equal(visible[4], 'roi');
  assert.equal(visible[5], 'profit');
  assert.ok(visible.indexOf('roi') < visible.indexOf('countries'));
});

test('moveDataColumn reorders status among metrics', () => {
  const order = defaultCampaignListColumnPrefs().dataColumnOrder;
  const moved = moveDataColumn(order, 'status', 'roi');
  const visible = visibleCampaignListColumns({
    ...defaultCampaignListColumnPrefs(),
    dataColumnOrder: moved,
  });
  assert.ok(visible.indexOf('roi') < visible.indexOf('status'));
});

test('parseCampaignListColumnPrefs migrates h_leads to hold_leads', () => {
  const prefs = parseCampaignListColumnPrefs(
    JSON.stringify({
      dataColumnOrder: ['name', 'h_leads', 'approved'],
      hidden: ['h_leads'],
    })
  );

  assert.equal(prefs.dataColumnOrder.includes('hold_leads'), true);
  assert.equal(prefs.dataColumnOrder.includes('h_leads' as never), false);
  assert.equal(prefs.hidden.includes('hold_leads'), true);
});

test('parseCampaignListColumnPrefs merges unknown and drops legacy tags column', () => {
  const prefs = parseCampaignListColumnPrefs(
    JSON.stringify({
      dataColumnOrder: ['name', 'group', 'tags', 'unknown', 'tags', 'clicks'],
      hidden: ['roi', 'bogus'],
    })
  );

  assert.equal(prefs.dataColumnOrder[0], 'name');
  assert.equal(prefs.dataColumnOrder.includes('tags' as never), false);
  assert.equal(prefs.hidden.includes('roi'), true);
  assert.equal(visibleCampaignListColumns(prefs).includes('roi'), false);
});

test('moveMiddleColumn reorders middle metrics', () => {
  const order = defaultCampaignListColumnPrefs().dataColumnOrder.filter(
    (columnId) => columnId !== 'name'
  );
  const moved = moveMiddleColumn(order, 0, 2);
  assert.equal(moved[0], order[1]);
});

test('moveDataColumn reorders draggable columns', () => {
  const order = defaultCampaignListColumnPrefs().dataColumnOrder;
  const countriesIndex = order.indexOf('countries');
  const roiIndex = order.indexOf('roi');
  assert.ok(countriesIndex >= 0);
  assert.ok(roiIndex >= 0);
  assert.ok(countriesIndex > roiIndex);

  const moved = moveDataColumn(order, 'countries', 'roi');
  const nextCountriesIndex = moved.indexOf('countries');
  const nextRoiIndex = moved.indexOf('roi');

  assert.ok(nextCountriesIndex >= 0);
  assert.ok(nextRoiIndex >= 0);
  assert.equal(nextCountriesIndex, nextRoiIndex - 1);
});

test('setMiddleColumnVisible toggles hidden set', () => {
  const hidden = setMiddleColumnVisible([], 'impressions', false);
  assert.equal(hidden.includes('impressions'), true);
  const shown = setMiddleColumnVisible(hidden, 'impressions', true);
  assert.equal(shown.includes('impressions'), false);
});

test('isCampaignListColumnResizable blocks select id status and trailing column', () => {
  const columns = ['select', 'id', 'name', 'status', 'countries'] as const;

  assert.equal(isCampaignListColumnResizable('select', columns), false);
  assert.equal(isCampaignListColumnResizable('id', columns), false);
  assert.equal(isCampaignListColumnResizable('status', columns), false);
  assert.equal(isCampaignListColumnResizable('countries', columns), false);
  assert.equal(isCampaignListColumnResizable('name', columns), true);
});
