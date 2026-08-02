import { parseIsoUnixSeconds } from './iso_time.js';

/**
 * Build owned time-series x/y typed arrays from hourly metrics.
 *
 * @param {Array<{ hour: string, impressions?: number, clicks?: number, conversions?: number }>} hourly
 * @param {'impressions'|'clicks'|'conversions'} field
 * @returns {{ x: Float64Array, y: Float64Array, length: number }}
 */
export function seriesFromHourly(hourly, field) {
  const n = hourly.length;
  const x = new Float64Array(n);
  const y = new Float64Array(n);
  for (let i = 0; i < n; i++) {
    x[i] = parseIsoUnixSeconds(hourly[i].hour);
    y[i] = hourly[i][field] ?? 0;
  }
  return { x, y, length: n };
}
