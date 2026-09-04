import assert from 'node:assert/strict';
import test from 'node:test';

import { visibleMiddleColumnCount } from './campaign_list_columns.ts';
import { defaultCampaignListColumnPrefs } from './campaign_list_columns.ts';

test('visibleMiddleColumnCount excludes hidden middle columns only', () => {
  const prefs = defaultCampaignListColumnPrefs();
  const allVisible = visibleMiddleColumnCount(prefs);
  assert.ok(allVisible > 0);

  const withHidden = {
    ...prefs,
    hidden: [...prefs.hidden, 'bots', 'impressions'],
  };
  assert.equal(visibleMiddleColumnCount(withHidden), allVisible - 2);
});
