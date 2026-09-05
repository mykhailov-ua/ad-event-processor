import assert from 'node:assert/strict';
import test from 'node:test';

import { formatCampaignJsonKey } from '@/domains/campaigns/editor/campaign_json_labels';

test('formatCampaignJsonKey maps geo summary wire keys', () => {
  assert.equal(formatCampaignJsonKey('included_label'), 'Included countries');
  assert.equal(formatCampaignJsonKey('excluded_label'), 'Excluded countries');
});

test('formatCampaignJsonKey title-cases unknown slugs', () => {
  assert.equal(formatCampaignJsonKey('estimated_outbox_events'), 'Estimated Outbox Events');
});
