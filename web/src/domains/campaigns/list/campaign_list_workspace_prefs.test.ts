import assert from 'node:assert/strict';
import test from 'node:test';

import { defaultCampaignListColumnPrefs } from './campaign_list_columns.ts';
import { defaultCampaignListRowDisplayPrefs } from './campaign_list_row_display.ts';
import {
  defaultCampaignListWorkspacePrefs,
  resetCampaignListWorkspacePrefs,
} from './campaign_list_workspace_prefs.ts';

test('defaultCampaignListWorkspacePrefs matches column and row defaults', () => {
  const prefs = defaultCampaignListWorkspacePrefs();
  assert.deepEqual(prefs.columnPrefs, defaultCampaignListColumnPrefs());
  assert.deepEqual(prefs.rowDisplayPrefs, defaultCampaignListRowDisplayPrefs());
});

test('resetCampaignListWorkspacePrefs returns defaults', () => {
  const prefs = resetCampaignListWorkspacePrefs();
  assert.deepEqual(prefs.columnPrefs.hidden, defaultCampaignListColumnPrefs().hidden);
  assert.equal(prefs.rowDisplayPrefs.highlightActiveRows, false);
});
