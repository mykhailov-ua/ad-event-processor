import assert from 'node:assert/strict';
import test from 'node:test';

import { campaignListTableCardClass } from '@/domains/campaigns/list/campaign_list_classes.ts';
import { uiSurfaces } from '@/lib/ui_surfaces.ts';

test('campaignListTableCardClass uses shared table host surface', () => {
  assert.ok(campaignListTableCardClass.includes(uiSurfaces.tableHost));
});
