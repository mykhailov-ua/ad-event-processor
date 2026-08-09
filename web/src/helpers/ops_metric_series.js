/** @typedef {{ ts: Float64Array, value: Float64Array, len: number }} MetricSoA */

/** @typedef {{ ts: number, value: number }} MetricPoint */

/** @type {Record<string, string>} */
export const OPS_METRIC_API_NAMES = {
  'outbox-pending': 'ad_management_outbox_pending_total',
  'rps-estimate': 'ad_http_requests_total',
  'drift-alert': 'ad_recon_drift_micro',
};

/** @type {Record<string, string>} */
export const OPS_METRIC_COLORS = {
  'outbox-pending': '--warning',
  'rps-estimate': '--success',
  'drift-alert': '--danger',
  'emergency-breaker': '--danger',
  'ingress-h1': '--info',
  'ingress-h2': '--accent',
  'ingress-h3': '--success',
  'edge-tarpit': '--warning',
  'edge-blacklist-stale': '--danger',
  'edge-fraud-tier': '--danger',
};

/** @type {string[]} */
export const OPS_DROP_COLORS = ['--warning', '--danger', '--info', '--accent', '--success'];

/** @type {Array<{ value: number, label: string }>} */
export const OPS_CHART_RANGE_OPTIONS = [
  { value: 1, label: '1h' },
  { value: 6, label: '6h' },
  { value: 12, label: '12h' },
  { value: 24, label: '24h' },
];

const HISTORY_MAX = 360;

/** @type {Map<string, MetricSoA>} */
const snapshotHistory = new Map();

/** Cold-path only — flat 2-point fallback (allocates; not used on paint path). */
function flatPair(t0, v0, t1, v1) {
  return [
    { ts: t0, value: v0 },
    { ts: t1, value: v1 },
  ];
}

/**
 * @returns {MetricSoA}
 */
function createSoABuffer() {
  return {
    ts: new Float64Array(HISTORY_MAX),
    value: new Float64Array(HISTORY_MAX),
    len: 0,
  };
}

/**
 * Return range length in milliseconds.
 *
 * @param {number} hours
 * @returns {number}
 */
export function rangeMsFromHours(hours) {
  const h = Number(hours) || 24;
  return h * 60 * 60 * 1000;
}

/**
 * @param {Array<{ ts?: string, value?: number }>} rows
 * @returns {MetricPoint[]}
 */
export function parseApiPoints(rows) {
  const out = [];
  const n = rows?.length ?? 0;
  for (let i = 0; i < n; i++) {
    const row = rows[i];
    const ts = Date.parse(row?.ts ?? '');
    if (!Number.isFinite(ts)) continue;
    out.push({ ts, value: Number(row?.value) || 0 });
  }
  out.sort((a, b) => a.ts - b.ts);
  return out;
}

/**
 * @param {MetricPoint[]} points
 * @returns {MetricPoint[]}
 */
export function toRateSeries(points) {
  const plen = points.length;
  if (plen < 2) {
    return plen === 1
      ? [{ ts: points[0].ts, value: 0 }]
      : [];
  }
  /** @type {MetricPoint[]} */
  const out = [];
  for (let i = 1; i < plen; i++) {
    const dt = (points[i].ts - points[i - 1].ts) / 1000;
    const dv = points[i].value - points[i - 1].value;
    out.push({
      ts: points[i].ts,
      value: dt > 0 ? Math.max(0, dv / dt) : 0,
    });
  }
  return out;
}

/**
 * Copy points within range into SoA output buffers.
 *
 * @param {Array<{ ts: number, value: number }>|null|undefined} points
 * @param {number} fallbackValue
 * @param {number} rangeMs
 * @param {number} [now]
 * @param {Float64Array} outTs
 * @param {Float64Array} outVal
 * @returns {number}
 */
export function copyMetricPoints(points, fallbackValue, rangeMs, outTs, outVal, now = Date.now()) {
  const cutoff = now - rangeMs;
  let n = 0;
  const plen = points?.length ?? 0;
  const cap = outTs.length;
  for (let i = 0; i < plen && n < cap; i++) {
    const p = points[i];
    const ts = p.ts;
    if (ts >= cutoff) {
      outTs[n] = ts;
      outVal[n] = p.value;
      n++;
    }
  }
  if (n >= 2) return n;
  const v = n > 0 ? outVal[n - 1] : fallbackValue;
  outTs[0] = cutoff;
  outVal[0] = v;
  outTs[1] = now;
  outVal[1] = v;
  return 2;
}

/**
 * Copy snapshot ring into SoA buffers (hot-path safe).
 *
 * @param {string} id
 * @param {number} fallbackValue
 * @param {number} rangeMs
 * @param {Float64Array} outTs
 * @param {Float64Array} outVal
 * @param {number} [now]
 * @returns {number}
 */
export function copySnapshotSeriesSoA(id, fallbackValue, rangeMs, outTs, outVal, now = Date.now()) {
  const ring = snapshotHistory.get(id);
  if (!ring || ring.len === 0) {
    return copyMetricPoints(null, fallbackValue, rangeMs, outTs, outVal, now);
  }
  const cutoff = now - rangeMs;
  let n = 0;
  const cap = outTs.length;
  for (let i = 0; i < ring.len && n < cap; i++) {
    const ts = ring.ts[i];
    if (ts >= cutoff) {
      outTs[n] = ts;
      outVal[n] = ring.value[i];
      n++;
    }
  }
  if (n >= 2) return n;
  const v = n > 0 ? outVal[n - 1] : fallbackValue;
  outTs[0] = cutoff;
  outVal[0] = v;
  outTs[1] = now;
  outVal[1] = v;
  return 2;
}

/**
 * @param {MetricPoint[]} points
 * @param {number} rangeMs
 * @returns {MetricPoint[]}
 */
export function filterSeriesByRange(points, rangeMs) {
  const cutoff = Date.now() - rangeMs;
  const out = [];
  const plen = points?.length ?? 0;
  for (let i = 0; i < plen; i++) {
    const p = points[i];
    if (p.ts >= cutoff) out.push(p);
  }
  return out;
}

/**
 * @param {string} id
 * @param {number} value
 */
export function recordSnapshot(id, value) {
  let ring = snapshotHistory.get(id);
  if (!ring) {
    ring = createSoABuffer();
    snapshotHistory.set(id, ring);
  }
  const now = Date.now();
  const len = ring.len;
  if (len > 0 && now - ring.ts[len - 1] < 5000) {
    ring.ts[len - 1] = now;
    ring.value[len - 1] = value;
    return;
  }
  if (len >= HISTORY_MAX) {
    ring.ts.copyWithin(0, 1, len);
    ring.value.copyWithin(0, 1, len);
    ring.len = len - 1;
  }
  const write = ring.len;
  ring.ts[write] = now;
  ring.value[write] = value;
  ring.len = write + 1;
}

/**
 * Cold-path AoS view for specs / legacy callers (reuses two slot objects).
 *
 * @param {string} id
 * @param {number} value
 * @param {number} rangeMs
 * @returns {MetricPoint[]}
 */
export function snapshotSeries(id, value, rangeMs) {
  const scratchTs = new Float64Array(2);
  const scratchVal = new Float64Array(2);
  const n = copySnapshotSeriesSoA(id, value, rangeMs, scratchTs, scratchVal);
  if (n >= 2 && snapshotHistory.get(id)?.len >= 2) {
    const ring = snapshotHistory.get(id);
    const cutoff = Date.now() - rangeMs;
    const out = [];
    for (let i = 0; i < ring.len; i++) {
      if (ring.ts[i] >= cutoff) {
        out.push({ ts: ring.ts[i], value: ring.value[i] });
      }
    }
    return out.length >= 2 ? out : flatPair(scratchTs[0], scratchVal[0], scratchTs[1], scratchVal[1]);
  }
  return flatPair(scratchTs[0], scratchVal[0], scratchTs[1], scratchVal[1]);
}

/**
 * @param {MetricPoint[]} points
 * @param {number} fallbackValue
 * @param {number} rangeMs
 * @returns {MetricPoint[]}
 */
export function normalizeSeries(points, fallbackValue = 0, rangeMs = rangeMsFromHours(24)) {
  const now = Date.now();
  const scratchTs = new Float64Array(2);
  const scratchVal = new Float64Array(2);
  const n = copyMetricPoints(points, fallbackValue, rangeMs, scratchTs, scratchVal, now);
  if (n >= 2 && (points?.length ?? 0) >= 2) {
    const cutoff = now - rangeMs;
    const out = [];
    const plen = points.length;
    for (let i = 0; i < plen; i++) {
      if (points[i].ts >= cutoff) out.push(points[i]);
    }
    return out.length >= 2 ? out : flatPair(scratchTs[0], scratchVal[0], scratchTs[1], scratchVal[1]);
  }
  return flatPair(scratchTs[0], scratchVal[0], scratchTs[1], scratchVal[1]);
}

/**
 * @param {string} id
 * @param {number} [dropIndex]
 * @returns {string}
 */
export function metricColorToken(id, dropIndex = 0) {
  if (OPS_METRIC_COLORS[id]) return OPS_METRIC_COLORS[id];
  if (id.startsWith('drop-')) {
    return OPS_DROP_COLORS[dropIndex % OPS_DROP_COLORS.length];
  }
  return '--accent';
}
