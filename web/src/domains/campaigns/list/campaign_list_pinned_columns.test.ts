import assert from 'node:assert/strict';
import test from 'node:test';

import {
  campaignListPinnedColumnLeftPx,
  isCampaignListLastPinnedColumn,
} from './campaign_list_pinned_columns.ts';

test('campaignListPinnedColumnLeftPx offsets pinned columns by prior widths', () => {
  const columns = ['select', 'id', 'name', 'clicks'] as const;
  const widths = {
    select: 48,
    id: 80,
    name: 200,
    clicks: 96,
  };
  assert.equal(campaignListPinnedColumnLeftPx('select', columns, widths), 0);
  assert.equal(campaignListPinnedColumnLeftPx('id', columns, widths), 48);
  assert.equal(campaignListPinnedColumnLeftPx('name', columns, widths), 128);
});

test('isCampaignListLastPinnedColumn marks name when pinned trio is visible', () => {
  const columns = ['select', 'id', 'name', 'status'] as const;
  assert.equal(isCampaignListLastPinnedColumn('name', columns), true);
  assert.equal(isCampaignListLastPinnedColumn('id', columns), false);
});
