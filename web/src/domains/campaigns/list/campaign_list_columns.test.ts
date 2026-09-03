import test from 'node:test';
import assert from 'node:assert/strict';

import {
  CAMPAIGN_LIST_DEFAULT_HIDDEN,
  clampCampaignListColumnWidthPx,
  defaultCampaignListColumnPrefs,
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
  const widths = normalizeColumnWidthPx({ name: 4096, clicks: 80 });
  assert.equal(widths.name, 320);
  assert.equal(widths.clicks, 80);
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

test('parseCampaignListColumnPrefs migrates h_leads to hold_leads', () => {
  const prefs = parseCampaignListColumnPrefs(
    JSON.stringify({
      dataColumnOrder: ['name', 'h_leads', 'approved'],
      hidden: ['h_leads'],
    }),
  );

  assert.equal(prefs.dataColumnOrder.includes('hold_leads'), true);
  assert.equal(prefs.dataColumnOrder.includes('h_leads' as never), false);
  assert.equal(prefs.hidden.includes('hold_leads'), true);
});

test('parseCampaignListColumnPrefs merges unknown and duplicate middle columns', () => {
  const prefs = parseCampaignListColumnPrefs(
    JSON.stringify({
      dataColumnOrder: ['name', 'group', 'tags', 'unknown', 'tags', 'clicks'],
      hidden: ['roi', 'bogus'],
    }),
  );

  assert.equal(prefs.dataColumnOrder[0], 'name');
  assert.equal(prefs.hidden.includes('roi'), true);
  assert.equal(visibleCampaignListColumns(prefs).includes('roi'), false);
});

test('moveMiddleColumn reorders middle metrics', () => {
  const order = defaultCampaignListColumnPrefs().dataColumnOrder.filter(
    (columnId) => columnId !== 'name',
  );
  const moved = moveMiddleColumn(order, 0, 2);
  assert.equal(moved[0], order[1]);
});

test('moveDataColumn reorders draggable columns', () => {
  const order = defaultCampaignListColumnPrefs().dataColumnOrder;
  const clicksIndex = order.indexOf('clicks');
  const roiIndex = order.indexOf('roi');
  assert.ok(clicksIndex >= 0);
  assert.ok(roiIndex >= 0);
  assert.notEqual(clicksIndex, roiIndex);

  const moved = moveDataColumn(order, 'roi', 'clicks');
  const nextRoiIndex = moved.indexOf('roi');
  const nextClicksIndex = moved.indexOf('clicks');

  assert.ok(nextRoiIndex >= 0);
  assert.ok(nextClicksIndex >= 0);
  assert.equal(nextRoiIndex, nextClicksIndex - 1);
});

test('setMiddleColumnVisible toggles hidden set', () => {
  const hidden = setMiddleColumnVisible([], 'impressions', false);
  assert.equal(hidden.includes('impressions'), true);
  const shown = setMiddleColumnVisible(hidden, 'impressions', true);
  assert.equal(shown.includes('impressions'), false);
});
