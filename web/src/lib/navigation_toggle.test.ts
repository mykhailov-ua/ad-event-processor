import assert from 'node:assert/strict';
import test from 'node:test';

import { navigationExpanded } from './navigation_toggle.ts';

test('navigationExpanded follows desktop sidebar collapse', () => {
  assert.equal(navigationExpanded(true, false, false), true);
  assert.equal(navigationExpanded(true, true, false), false);
});

test('navigationExpanded_holdout ignores desktop collapse on mobile viewport', () => {
  assert.equal(navigationExpanded(false, false, false), false);
  assert.equal(navigationExpanded(false, true, true), true);
});
