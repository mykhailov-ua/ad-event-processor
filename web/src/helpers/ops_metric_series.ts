export type MetricSoA = {
  ts: Float64Array;
  value: Float64Array;
  len: number;
};

export type MetricPoint = {
  ts: number;
  value: number;
};

export type ApiMetricRow = {
  ts?: string;
  value?: number;
};

export const OPS_METRIC_API_NAMES: Record<string, string> = {
  'outbox-pending': 'ad_control_outbox_pending_total',
  'rps-estimate': 'ad_http_requests_total',
  'drift-alert': 'ad_recon_drift_micro',
};

export const OPS_METRIC_COLORS: Record<string, string> = {
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

export const OPS_DROP_COLORS: string[] = [
  '--warning',
  '--danger',
  '--info',
  '--accent',
  '--success',
];

export const OPS_CHART_RANGE_OPTIONS: Array<{ value: number; label: string }> = [
  { value: 1, label: '1h' },
  { value: 6, label: '6h' },
  { value: 12, label: '12h' },
  { value: 24, label: '24h' },
];

const HISTORY_MAX = 360;

const snapshotHistory = new Map<string, MetricSoA>();

function flatPair(t0: number, v0: number, t1: number, v1: number): MetricPoint[] {
  return [
    { ts: t0, value: v0 },
    { ts: t1, value: v1 },
  ];
}

function createSoABuffer(): MetricSoA {
  return {
    ts: new Float64Array(HISTORY_MAX),
    value: new Float64Array(HISTORY_MAX),
    len: 0,
  };
}

export function rangeMsFromHours(hours: number): number {
  const h = Number(hours) || 24;
  return h * 60 * 60 * 1000;
}

export function parseApiPoints(rows: ApiMetricRow[] | null | undefined): MetricPoint[] {
  const out: MetricPoint[] = [];
  const n = rows?.length ?? 0;
  for (let i = 0; i < n; i++) {
    const row = rows![i];
    const ts = Date.parse(row?.ts ?? '');
    if (!Number.isFinite(ts)) continue;
    out.push({ ts, value: Number(row?.value) || 0 });
  }
  out.sort((a, b) => a.ts - b.ts);
  return out;
}

export function toRateSeries(points: MetricPoint[]): MetricPoint[] {
  const plen = points.length;
  if (plen < 2) {
    return plen === 1 ? [{ ts: points[0].ts, value: 0 }] : [];
  }
  const out: MetricPoint[] = [];
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

export function copyMetricPoints(
  points: Array<{ ts: number; value: number }> | null | undefined,
  fallbackValue: number,
  rangeMs: number,
  outTs: Float64Array,
  outVal: Float64Array,
  now = Date.now()
): number {
  const cutoff = now - rangeMs;
  let n = 0;
  const plen = points?.length ?? 0;
  const cap = outTs.length;
  for (let i = 0; i < plen && n < cap; i++) {
    const p = points![i];
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

export function copySnapshotSeriesSoA(
  id: string,
  fallbackValue: number,
  rangeMs: number,
  outTs: Float64Array,
  outVal: Float64Array,
  now = Date.now()
): number {
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

export function filterSeriesByRange(
  points: MetricPoint[] | null | undefined,
  rangeMs: number
): MetricPoint[] {
  const cutoff = Date.now() - rangeMs;
  const out: MetricPoint[] = [];
  const plen = points?.length ?? 0;
  for (let i = 0; i < plen; i++) {
    const p = points![i];
    if (p.ts >= cutoff) out.push(p);
  }
  return out;
}

export function recordSnapshot(id: string, value: number): void {
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

export function snapshotSeries(id: string, value: number, rangeMs: number): MetricPoint[] {
  const scratchTs = new Float64Array(2);
  const scratchVal = new Float64Array(2);
  const n = copySnapshotSeriesSoA(id, value, rangeMs, scratchTs, scratchVal);
  if (n >= 2 && (snapshotHistory.get(id)?.len ?? 0) >= 2) {
    const ring = snapshotHistory.get(id)!;
    const cutoff = Date.now() - rangeMs;
    const out: MetricPoint[] = [];
    for (let i = 0; i < ring.len; i++) {
      if (ring.ts[i] >= cutoff) {
        out.push({ ts: ring.ts[i], value: ring.value[i] });
      }
    }
    return out.length >= 2
      ? out
      : flatPair(scratchTs[0], scratchVal[0], scratchTs[1], scratchVal[1]);
  }
  return flatPair(scratchTs[0], scratchVal[0], scratchTs[1], scratchVal[1]);
}

export function normalizeSeries(
  points: MetricPoint[] | null | undefined,
  fallbackValue = 0,
  rangeMs = rangeMsFromHours(24)
): MetricPoint[] {
  const now = Date.now();
  const scratchTs = new Float64Array(2);
  const scratchVal = new Float64Array(2);
  const n = copyMetricPoints(points, fallbackValue, rangeMs, scratchTs, scratchVal, now);
  if (n >= 2 && (points?.length ?? 0) >= 2) {
    const cutoff = now - rangeMs;
    const out: MetricPoint[] = [];
    const plen = points!.length;
    for (let i = 0; i < plen; i++) {
      if (points![i].ts >= cutoff) out.push(points![i]);
    }
    return out.length >= 2
      ? out
      : flatPair(scratchTs[0], scratchVal[0], scratchTs[1], scratchVal[1]);
  }
  return flatPair(scratchTs[0], scratchVal[0], scratchTs[1], scratchVal[1]);
}

export function metricColorToken(id: string, dropIndex = 0): string {
  if (OPS_METRIC_COLORS[id]) return OPS_METRIC_COLORS[id];
  if (id.startsWith('drop-')) {
    return OPS_DROP_COLORS[dropIndex % OPS_DROP_COLORS.length];
  }
  return '--accent';
}
