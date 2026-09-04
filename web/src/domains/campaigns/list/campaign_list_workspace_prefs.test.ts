import assert from 'node:assert/strict';
import test from 'node:test';

import {
  defaultCampaignListColumnPrefs,
  saveCampaignListColumnPrefs,
} from './campaign_list_columns.ts';
import {
  defaultCampaignListWorkspacePrefs,
  resetCampaignListWorkspacePrefs,
} from './campaign_list_workspace_prefs.ts';

test('resetCampaignListWorkspacePrefs restores default column prefs', () => {
  const mutated = defaultCampaignListColumnPrefs();
  mutated.hidden = [...mutated.hidden, 'roi'];
  saveCampaignListColumnPrefs(mutated);

  const reset = resetCampaignListWorkspacePrefs();
  const defaults = defaultCampaignListWorkspacePrefs();

  assert.deepEqual(reset.columnPrefs.hidden, defaults.columnPrefs.hidden);
  assert.equal(reset.columnPrefs.hidden.includes('roi'), false);
});
