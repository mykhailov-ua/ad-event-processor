import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildDashboardMockSeries,
  DASHBOARD_MOCK_DEFAULT_FROM,
  DASHBOARD_MOCK_DEFAULT_TO,
} from '@/domains/dashboards/dashboard_series_mock';
import { DASHBOARD_MOCK_SERIES_BUDGET, PERF_TOLERANCE_RATIO } from '@/lib/perf/budgets';
import { measureMedianMs } from '@/lib/perf/measure';

test('buildDashboardMockSeries holdout: 63-day series within Node perf budget', () => {
  const medianMs = measureMedianMs(() => {
    buildDashboardMockSeries(DASHBOARD_MOCK_DEFAULT_FROM, DASHBOARD_MOCK_DEFAULT_TO);
  }, 5);

  const limitMs = DASHBOARD_MOCK_SERIES_BUDGET.medianMs * PERF_TOLERANCE_RATIO;
  assert.ok(
    medianMs <= limitMs,
    `buildDashboardMockSeries ${medianMs.toFixed(3)} ms exceeds ${limitMs.toFixed(3)} ms`
  );
});
