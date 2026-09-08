/** Admin SPA microbench budgets (Node CPU). Browser paint uses separate Playwright gates. */

export const PERF_TOLERANCE_RATIO = 1.15;

export type PerfBudget = {
  name: string;
  /** Median milliseconds per iteration (or per batch unit when batchSize > 1). */
  medianMs: number;
  warmupIterations: number;
  measureIterations: number;
  batchSize?: number;
};

/** Node microbench on dev host (~100 rows x 30 cols width probe). Not browser first-paint SLA. */
export const CAMPAIGN_LIST_WIDTHS_100_BUDGET: PerfBudget = {
  name: 'computeCampaignListColumnWidths(100 rows x ~32 cols)',
  medianMs: 650,
  warmupIterations: 3,
  measureIterations: 7,
};

export const CAMPAIGN_LIST_ROW_VM_100_BUDGET: PerfBudget = {
  name: 'buildCampaignRowVm x100',
  medianMs: 0.5,
  warmupIterations: 3,
  measureIterations: 7,
  batchSize: 100,
};

export const CAMPAIGN_LIST_MOVE_COLUMN_BUDGET: PerfBudget = {
  name: 'moveDataColumn x1000',
  medianMs: 0.05,
  warmupIterations: 3,
  measureIterations: 7,
  batchSize: 1000,
};

export const DASHBOARD_MOCK_SERIES_BUDGET: PerfBudget = {
  name: 'buildDashboardMockSeries(63 days)',
  medianMs: 3,
  warmupIterations: 5,
  measureIterations: 10,
};

export const DASHBOARD_AXIS_SCALE_BUDGET: PerfBudget = {
  name: 'dashboard axis scales (vol+money+date)',
  medianMs: 1,
  warmupIterations: 5,
  measureIterations: 10,
};

export const NAV_FILTER_1K_BUDGET: PerfBudget = {
  name: 'filterNavItems x1000 on 1k catalog',
  medianMs: 0.12,
  warmupIterations: 3,
  measureIterations: 7,
  batchSize: 1000,
};

export const JSON_PARSE_LARGE_PREFS_BUDGET: PerfBudget = {
  name: 'JSON.parse campaign column prefs (~5 KiB)',
  medianMs: 0.5,
  warmupIterations: 5,
  measureIterations: 20,
};

/** Browser wall-clock budgets (Playwright). See web/e2e/perf/helpers_perf.js. */
export const PLAYWRIGHT_CHART_REGION_MS = 3000;
export const PLAYWRIGHT_CAMPAIGN_TABLE_MS = 3000;
export const PLAYWRIGHT_PALETTE_OPEN_MS = 500;
