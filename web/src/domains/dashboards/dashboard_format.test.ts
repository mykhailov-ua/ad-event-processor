import assert from 'node:assert/strict';
import test from 'node:test';

import {
  derivePortfolioRoiPct,
  formatDashboardUsdFromMicro,
  resolveBreakdownProfitMicro,
  resolveBreakdownRoiPct,
  resolvePortfolioProfitMicro,
} from './dashboard_format.ts';
import type { BuyerPortfolio } from './buyer_dashboard_types.ts';

test('formatDashboardUsdFromMicro formats micro wire values as USD', () => {
  assert.equal(formatDashboardUsdFromMicro(2_221_050_000), '$2,221.05');
});

test('derivePortfolioRoiPct returns -100 when revenue is zero and cost is positive', () => {
  const portfolio: BuyerPortfolio = {
    kpis: {
      cost_micro: 2_221_050_000,
      revenue_micro: 0,
      profit_micro: 0,
      roi_pct: 0,
    },
  };
  assert.equal(resolvePortfolioProfitMicro(portfolio), -2_221_050_000);
  assert.equal(derivePortfolioRoiPct(portfolio), -100);
});

test('resolveBreakdownProfitMicro_holdout ignores stale profit_micro when revenue and cost exist', () => {
  assert.equal(
    resolveBreakdownProfitMicro({
      revenue_micro: 2_670_000,
      cost_micro: 3_777_930_000,
      profit_micro: 0,
    }),
    -3_775_260_000
  );
  assert.equal(
    resolveBreakdownRoiPct({
      revenue_micro: 2_670_000,
      cost_micro: 3_777_930_000,
      profit_micro: 0,
      roi_pct: 0,
    }),
    (-3_775_260_000 / 3_777_930_000) * 100
  );
});
