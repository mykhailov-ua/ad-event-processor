import assert from 'node:assert/strict';
import test from 'node:test';

import { FILTER_PANEL_FLAT_CLASS } from './filter_panel.ts';

test('FILTER_PANEL_FLAT_CLASS_holdout: keeps filter panel chrome padding', () => {
  assert.equal(FILTER_PANEL_FLAT_CLASS.includes('bg-transparent'), false);
  assert.equal(FILTER_PANEL_FLAT_CLASS.includes('p-0'), false);
  assert.ok(FILTER_PANEL_FLAT_CLASS.includes('gap-3'));
});
