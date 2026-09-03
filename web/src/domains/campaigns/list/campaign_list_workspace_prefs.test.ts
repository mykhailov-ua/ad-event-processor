import assert from 'node:assert/strict';
import test from 'node:test';

import { defaultCampaignListColumnPrefs } from './campaign_list_columns.ts';
import {
  defaultCampaignListWorkspacePrefs,
  resetCampaignListWorkspacePrefs,
} from './campaign_list_workspace_prefs.ts';

test('defaultCampaignListWorkspacePrefs matches column defaults', () => {
  const prefs = defaultCampaignListWorkspacePrefs();
  assert.deepEqual(prefs.columnPrefs, defaultCampaignListColumnPrefs());
});

test('resetCampaignListWorkspacePrefs returns defaults', () => {
  const prefs = resetCampaignListWorkspacePrefs();
  assert.deepEqual(prefs.columnPrefs.hidden, defaultCampaignListColumnPrefs().hidden);
});
