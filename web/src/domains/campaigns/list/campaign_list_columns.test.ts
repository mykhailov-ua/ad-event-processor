import test from 'node:test';
import assert from 'node:assert/strict';

import {
  CAMPAIGN_LIST_DEFAULT_HIDDEN,
  defaultCampaignListColumnPrefs,
  defaultCampaignListExportDataColumns,
  isCampaignListMiddleColumnId,
  visibleCampaignListColumns,
} from './campaign_list_columns.ts';

test('visibleCampaignListColumns hides legacy placeholder columns by default', () => {
  const visible = visibleCampaignListColumns(defaultCampaignListColumnPrefs());
  assert.ok(visible.includes('select'));
  assert.ok(visible.includes('id'));
  assert.ok(visible.includes('name'));
  assert.ok(visible.includes('clicks'));
  for (const hidden of CAMPAIGN_LIST_DEFAULT_HIDDEN) {
    assert.equal(visible.includes(hidden), false, `expected ${hidden} hidden by default`);
  }
});

test('defaultCampaignListExportDataColumns omits select column', () => {
  const columns = defaultCampaignListExportDataColumns();
  assert.equal(columns.includes('select' as never), false);
  assert.ok(columns.includes('id'));
  assert.ok(columns.includes('name'));
});

test('isCampaignListMiddleColumnId recognizes metric columns', () => {
  assert.equal(isCampaignListMiddleColumnId('clicks'), true);
  assert.equal(isCampaignListMiddleColumnId('select'), false);
});
