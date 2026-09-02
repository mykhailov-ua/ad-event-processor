import test from 'node:test';
import assert from 'node:assert/strict';

import { campaignDisplayId } from './campaign_display_id.ts';

test('campaignDisplayId uses eight-digit display_id from API when present', () => {
  assert.equal(
    campaignDisplayId({
      id: '6ba7b810-9dad-11d1-80b4-00c04fd430c8',
      display_id: '12345678',
    }),
    '12345678',
  );
});

test('campaignDisplayId falls back to deterministic eight-digit hash', () => {
  const got = campaignDisplayId({ id: '6ba7b810-9dad-11d1-80b4-00c04fd430c8' });
  assert.match(got, /^\d{8}$/);
  assert.equal(got, campaignDisplayId({ id: '6ba7b810-9dad-11d1-80b4-00c04fd430c8' }));
});
