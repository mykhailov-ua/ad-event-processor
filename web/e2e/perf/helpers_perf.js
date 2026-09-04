/** Playwright perf budgets (wall-clock; separate from Node microbench in web/src/lib/perf/budgets.ts). */

export const PERF_TOLERANCE_RATIO = 1.15;

export const PLAYWRIGHT_CHART_REGION_MS = 3000;
export const PLAYWRIGHT_CAMPAIGN_TABLE_MS = 3000;
export const PLAYWRIGHT_PALETTE_OPEN_MS = 500;

/**
 * Warm-up + median wall-clock for Playwright perf specs (not tracker SLA).
 */
export async function medianWallClockMs(run, { warmup = 2, samples = 5 } = {}) {
  for (let i = 0; i < warmup; i += 1) {
    await run();
  }
  const values = [];
  for (let i = 0; i < samples; i += 1) {
    const t0 = Date.now();
    await run();
    values.push(Date.now() - t0);
  }
  values.sort((a, b) => a - b);
  const mid = Math.floor(values.length / 2);
  if (values.length % 2 === 0) {
    return (values[mid - 1] + values[mid]) / 2;
  }
  return values[mid];
}

export function assertPlaywrightBudget(name, medianMs, budgetMs) {
  const limitMs = budgetMs * PERF_TOLERANCE_RATIO;
  if (medianMs > limitMs) {
    throw new Error(
      `${name}: median ${medianMs.toFixed(1)} ms exceeds budget ${budgetMs} ms (limit ${limitMs.toFixed(1)} ms)`,
    );
  }
}

export const PLAYWRIGHT_BUDGETS = {
  chartRegionMs: PLAYWRIGHT_CHART_REGION_MS,
  campaignTableMs: PLAYWRIGHT_CAMPAIGN_TABLE_MS,
  paletteOpenMs: PLAYWRIGHT_PALETTE_OPEN_MS,
};
