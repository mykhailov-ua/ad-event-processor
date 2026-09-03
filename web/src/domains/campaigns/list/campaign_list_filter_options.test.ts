import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildCampaignListCountryOptions,
  buildCampaignListOwnerEmailById,
  buildCampaignListOwnerOptions,
} from './campaign_list_filter_options.ts';
import { sortFieldForCampaignColumn } from './campaign_list_sort.ts';

test('buildCampaignListCountryOptions unions countries and applied filter', () => {
  const options = buildCampaignListCountryOptions(['US', 'CA', 'GB'], 'DE');

  assert.deepEqual(
    options.map((option) => option.value),
    ['__all__', 'CA', 'DE', 'GB', 'US'],
  );
});

test('buildCampaignListOwnerOptions dedupes owners and keeps applied owner', () => {
  const options = buildCampaignListOwnerOptions(
    [
      { user_id: 'user-a', email: 'buyer@example.com' },
      { user_id: 'user-a', email: 'buyer@example.com' },
      { user_id: 'user-b', email: 'ops@example.com' },
    ],
    'user-c',
  );

  assert.deepEqual(
    options.map((option) => option.value),
    ['__all__', 'user-a', 'user-b', 'user-c'],
  );
  assert.equal(options[1]?.label, 'buyer@example.com');
});

test('buildCampaignListOwnerEmailById maps owner emails', () => {
  assert.deepEqual(
    buildCampaignListOwnerEmailById([
      { user_id: 'user-a', email: 'buyer@example.com' },
      { user_id: 'user-b', email: '' },
    ]),
    { 'user-a': 'buyer@example.com' },
  );
});

test('buildCampaignListOwnerOptions keeps applied owner email from facets', () => {
  const options = buildCampaignListOwnerOptions(
    [{ user_id: 'user-a', email: 'buyer@example.com' }],
    'user-a',
  );

  assert.equal(options.find((option) => option.value === 'user-a')?.label, 'buyer@example.com');
});

test('sortFieldForCampaignColumn omits tags until list API exposes tag values', () => {
  assert.equal(sortFieldForCampaignColumn('tags'), undefined);
});
