import assert from 'node:assert/strict';
import test from 'node:test';

import { resolveEconomicsProfitMicro, resolveEconomicsRoiPct } from './economics.ts';

test('resolveEconomicsProfitMicro returns wire profit_micro', () => {
  assert.equal(
    resolveEconomicsProfitMicro({
      revenue_micro: 2_670_000,
      spend_micro: 3_777_930_000,
      profit_micro: -3_775_260_000,
    }),
    -3_775_260_000
  );
});

test('resolveEconomicsRoiPct returns wire roi_pct', () => {
  assert.equal(
    resolveEconomicsRoiPct({
      revenue_micro: 1_000_000,
      cost_micro: 600_000,
      roi_pct: -99.93,
    }),
    -99.93
  );
});

test('resolveEconomicsProfitMicro uses explicit fallback when profit_micro missing', () => {
  assert.equal(resolveEconomicsProfitMicro({ revenue_micro: 1_000_000 }, 500_000), 500_000);
});

test('resolveEconomicsRoiPct uses explicit fallback when roi_pct missing', () => {
  assert.equal(resolveEconomicsRoiPct({}, 12.5), 12.5);
});
