import assert from 'node:assert/strict';
import test from 'node:test';

import {
  applySplitPreset,
  normalizeVisualPathWeights,
  sumVisualPathWeights,
  validateVisualPathWeights,
  visualRowsToFlowPaths,
} from './flow_path_model.ts';

test('validateVisualPathWeights rejects bad sum', () => {
  const error = validateVisualPathWeights([
    { row_id: 'a', weight: 40, lander_id: 'l1', offer_id: 'o1', countries: '', devices: [] },
    { row_id: 'b', weight: 40, lander_id: 'l2', offer_id: 'o2', countries: '', devices: [] },
  ]);
  assert.match(error ?? '', /sum to 100/i);
});

test('normalizeVisualPathWeights_holdout', () => {
  const rows = normalizeVisualPathWeights([
    { row_id: 'a', weight: 1, lander_id: 'l1', offer_id: 'o1', countries: '', devices: [] },
    { row_id: 'b', weight: 1, lander_id: 'l2', offer_id: 'o2', countries: '', devices: [] },
  ]);
  assert.equal(sumVisualPathWeights(rows), 100);
});

test('visualRowsToFlowPaths maps filters', () => {
  const paths = visualRowsToFlowPaths([
    {
      row_id: 'a',
      weight: 100,
      lander_id: '11111111-1111-4111-8111-111111111111',
      offer_id: '22222222-2222-4222-8222-222222222222',
      countries: 'us, ca',
      devices: ['mobile'],
    },
  ]);
  assert.equal(paths[0]?.weight, 100);
  assert.equal(paths[0]?.landers?.[0]?.lander_id, '11111111-1111-4111-8111-111111111111');
  assert.deepEqual(paths[0]?.filters?.countries, ['US', 'CA']);
});

test('applySplitPreset sets 50/50', () => {
  const rows = applySplitPreset(
    [{ row_id: 'a', weight: 100, lander_id: 'l1', offer_id: 'o1', countries: '', devices: [] }],
    [50, 50]
  );
  assert.equal(rows.length, 2);
  assert.equal(sumVisualPathWeights(rows), 100);
});
