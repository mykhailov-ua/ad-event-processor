import assert from 'node:assert/strict';
import test from 'node:test';

import { applyDryRunToFlowPaths } from '@/domains/integrations/traffic_optimizer_apply';

test('applyDryRunToFlowPaths updates lander weights from server arms', () => {
  const result = applyDryRunToFlowPaths(
    [
      {
        weight: 100,
        landers: [
          { lander_id: '00000000-0000-4000-8000-000000000001', weight: 50 },
          { lander_id: '00000000-0000-4000-8000-000000000002', weight: 50 },
        ],
        offers: [{ offer_id: '00000000-0000-4000-8000-000000000101', weight: 100 }],
      },
    ],
    'lander',
    {
      stale_weights: false,
      arms: [
        {
          entity_id: '00000000-0000-4000-8000-000000000001',
          current_weight: 50,
          proposed_weight: 70,
          observed_value: 0.12,
        },
        {
          entity_id: '00000000-0000-4000-8000-000000000002',
          current_weight: 50,
          proposed_weight: 30,
          observed_value: 0.08,
        },
      ],
    }
  );
  assert.equal(result.ok, true);
  if (!result.ok) {
    return;
  }
  assert.equal(result.paths[0].landers?.[0].weight, 70);
  assert.equal(result.paths[0].landers?.[1].weight, 30);
});

test('applyDryRunToFlowPaths rejects empty arms', () => {
  const result = applyDryRunToFlowPaths([], 'offer', { stale_weights: false, arms: [] });
  assert.equal(result.ok, false);
});
