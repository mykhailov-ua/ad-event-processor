import assert from 'node:assert/strict';
import test from 'node:test';

import {
  flowPathsToVisualRows,
  normalizeEntityRefWeights,
  validateVisualPathWeights,
  visualRowsToFlowPaths,
} from '@/domains/creative/flow_path_model';

test('flowPathsToVisualRows preserves multiple landers and offers', () => {
  const rows = flowPathsToVisualRows([
    {
      weight: 100,
      landers: [
        { lander_id: '00000000-0000-4000-8000-000000000001', weight: 60 },
        { lander_id: '00000000-0000-4000-8000-000000000002', weight: 40 },
      ],
      offers: [
        { offer_id: '00000000-0000-4000-8000-000000000101', weight: 70 },
        { offer_id: '00000000-0000-4000-8000-000000000102', weight: 30 },
      ],
    },
  ]);
  assert.equal(rows.length, 1);
  assert.equal(rows[0].landers.length, 2);
  assert.equal(rows[0].offers.length, 2);
  assert.equal(rows[0].landers[0].entity_id, '00000000-0000-4000-8000-000000000001');
  assert.equal(rows[0].landers[0].weight, 60);
});

test('visualRowsToFlowPaths round-trips multi-entity path', () => {
  const [row] = flowPathsToVisualRows([
    {
      weight: 100,
      landers: [{ lander_id: '00000000-0000-4000-8000-000000000003', weight: 100 }],
      offers: [
        { offer_id: '00000000-0000-4000-8000-000000000104', weight: 50 },
        { offer_id: '00000000-0000-4000-8000-000000000105', weight: 50 },
      ],
    },
  ]);
  row.offers = normalizeEntityRefWeights(row.offers);
  const paths = visualRowsToFlowPaths([row]);
  assert.equal(paths[0].offers?.length, 2);
  assert.equal(paths[0].offers?.[0].offer_id, '00000000-0000-4000-8000-000000000104');
  assert.equal(paths[0].offers?.[1].weight, 50);
});

test('visualRowsToFlowPaths emits rotation_mode when not weighted', () => {
  const [row] = flowPathsToVisualRows([
    {
      weight: 100,
      rotation_mode: 'fix_on',
      landers: [{ lander_id: '00000000-0000-4000-8000-000000000003', weight: 100 }],
      offers: [{ offer_id: '00000000-0000-4000-8000-000000000104', weight: 100 }],
    },
  ]);
  const paths = visualRowsToFlowPaths([row]);
  assert.equal(paths[0].rotation_mode, 'fix_on');
});

test('validateVisualPathWeights rejects offer weights not summing to 100', () => {
  const [row] = flowPathsToVisualRows([
    {
      weight: 100,
      landers: [{ lander_id: '00000000-0000-4000-8000-000000000003', weight: 100 }],
      offers: [
        { offer_id: '00000000-0000-4000-8000-000000000104', weight: 40 },
        { offer_id: '00000000-0000-4000-8000-000000000105', weight: 40 },
      ],
    },
  ]);
  const error = validateVisualPathWeights([row]);
  assert.match(error ?? '', /offer weights must sum to 100/i);
});
