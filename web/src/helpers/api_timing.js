/** @type {Map<string, number[]>} */
const durations = new Map();

const UUID_RE = /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/gi;

/**
 * Normalize API paths for timing aggregation.
 *
 * @param {string} path
 * @returns {string}
 */
export function apiPathTemplate(path) {
  return path.split('?')[0].replace(UUID_RE, '{id}');
}

/**
 * Record one API round-trip duration in milliseconds.
 *
 * @param {string} path
 * @param {number} ms
 * @returns {void}
 */
export function recordApiTiming(path, ms) {
  const key = apiPathTemplate(path);
  const bucket = durations.get(key) ?? [];
  bucket.push(ms);
  if (bucket.length > 100) bucket.shift();
  durations.set(key, bucket);
}

/**
 * Compute percentile from a sorted numeric array.
 *
 * @param {number[]} sorted
 * @param {number} p
 * @returns {number}
 */
function percentile(sorted, p) {
  if (sorted.length === 0) return 0;
  const idx = Math.ceil((p / 100) * sorted.length) - 1;
  return sorted[Math.max(0, idx)];
}

/**
 * Return p50/p95 per path template and flag slow endpoints.
 *
 * @returns {{ paths: Record<string, { count: number, p50Ms: number, p95Ms: number }>, slowPaths: string[] }}
 */
export function apiTimingReport() {
  /** @type {Record<string, { count: number, p50Ms: number, p95Ms: number }>} */
  const paths = {};
  /** @type {string[]} */
  const slowPaths = [];
  for (const [key, values] of durations.entries()) {
    const sorted = [...values].sort((a, b) => a - b);
    const p50 = Math.round(percentile(sorted, 50));
    const p95 = Math.round(percentile(sorted, 95));
    paths[key] = { count: sorted.length, p50Ms: p50, p95Ms: p95 };
    if (p95 >= 500) slowPaths.push(key);
  }
  return { paths, slowPaths };
}

/**
 * Reset stored API timings.
 *
 * @returns {void}
 */
export function apiTimingReset() {
  durations.clear();
}
