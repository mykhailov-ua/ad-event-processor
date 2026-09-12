import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildCampaignBulkPatchPayload,
  EMPTY_CAMPAIGN_BULK_PATCH_DRAFT,
  EMPTY_CAMPAIGN_BULK_PATCH_ENABLED,
  validateCampaignBulkPatchDraft,
} from '@/domains/campaigns/list/campaign_bulk_patch_build';

test('buildCampaignBulkPatchPayload returns undefined when nothing enabled', () => {
  const patch = buildCampaignBulkPatchPayload(
    EMPTY_CAMPAIGN_BULK_PATCH_DRAFT,
    EMPTY_CAMPAIGN_BULK_PATCH_ENABLED
  );
  assert.equal(patch, undefined);
});

test('buildCampaignBulkPatchPayload maps enabled target_url and countries', () => {
  const enabled = {
    ...EMPTY_CAMPAIGN_BULK_PATCH_ENABLED,
    target_url: true,
    target_countries: true,
  };
  const draft = {
    ...EMPTY_CAMPAIGN_BULK_PATCH_DRAFT,
    target_url: 'https://offer.example/click',
    target_countries: 'us, de',
  };
  const patch = buildCampaignBulkPatchPayload(draft, enabled);
  assert.deepEqual(patch, {
    target_url: 'https://offer.example/click',
    target_countries: ['US', 'DE'],
  });
});

test('validateCampaignBulkPatchDraft rejects empty selection', () => {
  const message = validateCampaignBulkPatchDraft(
    EMPTY_CAMPAIGN_BULK_PATCH_DRAFT,
    EMPTY_CAMPAIGN_BULK_PATCH_ENABLED
  );
  assert.match(message ?? '', /at least one field/i);
});
