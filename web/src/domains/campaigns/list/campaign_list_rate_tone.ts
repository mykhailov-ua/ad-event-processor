/** CTR / CR benchmark tone for campaign list cells (font color only). */

const LOW_RATE_WARN_PCT = 1;

export function percentRate(numerator: number, denominator: number): number | null {
  if (denominator <= 0 || numerator <= 0) {
    return null;
  }
  return (numerator / denominator) * 100;
}

export function rateBenchmarkToneClass(ratePct: number | null): string | undefined {
  if (ratePct == null) {
    return undefined;
  }
  if (ratePct < LOW_RATE_WARN_PCT) {
    return 'admin-metric-rate-warn';
  }
  return undefined;
}
