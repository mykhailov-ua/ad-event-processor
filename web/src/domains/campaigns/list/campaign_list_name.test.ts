import assert from 'node:assert/strict';
import test from 'node:test';

import { parseCampaignListName } from './campaign_list_name.ts';

test('parseCampaignListName splits middle-dot segments', () => {
  const parts = parseCampaignListName('APAC Crypto Swap \u00b7 GB \u00b7 Alpha desk');
  assert.equal(parts.title, 'APAC Crypto Swap');
  assert.deepEqual(parts.meta, ['GB', 'Alpha desk']);
});

test('parseCampaignListName keeps plain names intact', () => {
  const parts = parseCampaignListName('Single campaign title');
  assert.equal(parts.title, 'Single campaign title');
  assert.deepEqual(parts.meta, []);
});
