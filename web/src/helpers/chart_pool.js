const POOL_SIZE = 2048;

/** @type {Float64Array} */
let tsPool = new Float64Array(POOL_SIZE);
/** @type {Float64Array} */
let valPool = new Float64Array(POOL_SIZE);

/**
 * @param {Array<{ hour: string, impressions?: number, clicks?: number, conversions?: number }>} hourly
 * @param {'impressions'|'clicks'|'conversions'} field
 * @returns {{ x: Float64Array, y: Float64Array, length: number }}
 */
export function seriesFromHourly(hourly, field) {
  const n = hourly.length;
  if (n > POOL_SIZE) {
    const x = new Float64Array(n);
    const y = new Float64Array(n);
    for (let i = 0; i < n; i++) {
      x[i] = new Date(hourly[i].hour).getTime() / 1000;
      y[i] = hourly[i][field] ?? 0;
    }
    return { x, y, length: n };
  }
  for (let i = 0; i < n; i++) {
    tsPool[i] = new Date(hourly[i].hour).getTime() / 1000;
    valPool[i] = hourly[i][field] ?? 0;
  }
  return { x: tsPool.subarray(0, n), y: valPool.subarray(0, n), length: n };
}

/**
 */
export function resetChartPools() {
  tsPool = new Float64Array(POOL_SIZE);
  valPool = new Float64Array(POOL_SIZE);
}
