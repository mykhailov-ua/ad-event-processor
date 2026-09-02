import assert from 'node:assert/strict';
import test from 'node:test';

import { countryFlagAssetPath } from './country_flag_assets.ts';

test('countryFlagAssetPath resolves bundled SVG paths for known ISO codes', () => {
  assert.equal(countryFlagAssetPath('us'), '/src/flags/3x2/US.svg');
  assert.equal(countryFlagAssetPath(' DE '), '/src/flags/3x2/DE.svg');
});

test('countryFlagAssetPath rejects invalid or unknown codes', () => {
  assert.equal(countryFlagAssetPath('USA'), null);
  assert.equal(countryFlagAssetPath(''), null);
  assert.equal(countryFlagAssetPath('XX'), null);
});
