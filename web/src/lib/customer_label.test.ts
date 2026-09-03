import assert from 'node:assert/strict';
import test from 'node:test';

import {
  isHumanCustomerLabel,
  isPlaceholderSeedUuid,
  resolveCustomerLabel,
} from './customer_label.ts';

test('resolveCustomerLabel returns mapped name and never raw uuid', () => {
  const label = resolveCustomerLabel('cbef2aa7-73b6-5572-8007-16be19302faf', {
    'cbef2aa7-73b6-5572-8007-16be19302faf': 'Horizon Media Group',
  });
  assert.equal(label, 'Horizon Media Group');
});

test('resolveCustomerLabel rejects uuid-shaped map values', () => {
  const label = resolveCustomerLabel('00000000-0000-0000-0000-000000000041', {
    '00000000-0000-0000-0000-000000000041': '00000000-0000-0000-0000-000000000041',
  });
  assert.equal(label, undefined);
});

test('isPlaceholderSeedUuid detects load-test style ids', () => {
  assert.equal(isPlaceholderSeedUuid('00000000-0000-0000-0000-000000000041'), true);
  assert.equal(isPlaceholderSeedUuid('cbef2aa7-73b6-5572-8007-16be19302faf'), false);
});

test('isHumanCustomerLabel rejects uuid strings', () => {
  assert.equal(isHumanCustomerLabel('Pacific Ads Studio'), true);
  assert.equal(isHumanCustomerLabel('00000000-0000-0000-0000-000000000001'), false);
});
