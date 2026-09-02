import assert from 'node:assert/strict';
import test from 'node:test';

import { percentRate, rateBenchmarkToneClass } from './campaign_list_rate_tone.ts';

test('percentRate returns null for empty denominators', () => {
  assert.equal(percentRate(0, 100), null);
  assert.equal(percentRate(10, 0), null);
});

test('rateBenchmarkToneClass warns below 1%', () => {
  assert.equal(rateBenchmarkToneClass(0.5), 'admin-metric-rate-warn');
  assert.equal(rateBenchmarkToneClass(3.2), undefined);
});
