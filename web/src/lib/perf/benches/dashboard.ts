import {
  buildDateAxisTicks,
  buildMoneyAxisScale,
  buildVolumeAxisScale,
} from '@/domains/dashboards/dashboard_chart_scale';
import {
  buildDashboardMockSeries,
  DASHBOARD_MOCK_DEFAULT_FROM,
  DASHBOARD_MOCK_DEFAULT_TO,
} from '@/domains/dashboards/dashboard_series_mock';
import {
  DASHBOARD_AXIS_SCALE_BUDGET,
  DASHBOARD_MOCK_SERIES_BUDGET,
} from '@/lib/perf/budgets';
import { assertPerfBudget, formatBenchResult, runPerfBudget, type BenchResult } from '@/lib/perf/measure';

export function benchDashboardHotPath(): BenchResult[] {
  const results: BenchResult[] = [];

  results.push(
    runPerfBudget(DASHBOARD_MOCK_SERIES_BUDGET, () => {
      buildDashboardMockSeries(DASHBOARD_MOCK_DEFAULT_FROM, DASHBOARD_MOCK_DEFAULT_TO);
    }),
  );

  const series = buildDashboardMockSeries(DASHBOARD_MOCK_DEFAULT_FROM, DASHBOARD_MOCK_DEFAULT_TO);
  const labels = series.map((point) => point.label ?? '');
  const maxClicks = Math.max(...series.map((point) => point.clicks ?? 0));
  const spendMicro = series.map((point) => point.spend_micro ?? point.spend_micros ?? 0);
  const revenueMicro = series.map((point) => point.revenue_micro ?? 0);
  const minMoney = Math.min(...spendMicro, ...revenueMicro);
  const maxMoney = Math.max(...spendMicro, ...revenueMicro);

  results.push(
    runPerfBudget(DASHBOARD_AXIS_SCALE_BUDGET, () => {
      buildVolumeAxisScale(maxClicks);
      buildMoneyAxisScale(minMoney, maxMoney);
      buildDateAxisTicks(labels);
    }),
  );

  return results;
}

export function runDashboardHotPathBench(): void {
  const results = benchDashboardHotPath();
  for (const result of results) {
    console.log(`bench ok: ${formatBenchResult(result)}`);
    assertPerfBudget(result);
  }
}
