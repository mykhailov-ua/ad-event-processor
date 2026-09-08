import assert from 'node:assert/strict';
import test from 'node:test';

import { resolveEconomicsProfitMicro, resolveEconomicsRoiPct } from './economics.ts';

test('resolveEconomicsProfitMicro_holdout ignores stale zero profit when revenue and spend exist', () => {
  assert.equal(
    resolveEconomicsProfitMicro({
      revenue_micro: 2_670_000,
      spend_micro: 3_777_930_000,
      profit_micro: 0,
    }),
    -3_775_260_000
  );
});

test('resolveEconomicsRoiPct derives roi from revenue minus cost', () => {
  assert.equal(
    resolveEconomicsRoiPct({
      revenue_micro: 1_000_000,
      cost_micro: 600_000,
      profit_micro: 0,
      roi_pct: 0,
    }),
    66.66666666666666
  );
});

test('resolveEconomicsProfitMicro uses explicit fallback when cost is missing', () => {
  assert.equal(resolveEconomicsProfitMicro({ revenue_micro: 1_000_000 }, 500_000), 500_000);
});
