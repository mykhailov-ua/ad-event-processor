type ProbeSample = {
  count: number;
  totalNs: number;
  totalAllocs: number;
  heapBytes: number;
};

export type ProbeHandle = {
  name: string;
  t0: number;
  heap0: number;
};

export type ProbeEndMeta = {
  allocs?: number;
  bytes?: number;
};

export type ProbeEndResult = {
  name: string;
  ns: number;
  heapDelta: number;
  allocs: number;
  bytes: number;
};

export type ProbeReportRow = {
  count: number;
  nsPerOp: number;
  allocPerOp: number;
  bytesPerOp: number;
};

export type ProbeReport = Record<string, ProbeReportRow>;

type PerformanceWithMemory = Performance & {
  memory?: { usedJSHeapSize?: number };
};

type ProbeDevtools = {
  report: () => ProbeReport;
  reset: () => void;
};

const samples = new Map<string, ProbeSample>();

let devtoolsInstalled = false;

/**
 * Expose probe report on window in dev builds only.
 */
function installDevtoolsProbe(): void {
  if (devtoolsInstalled || typeof window === 'undefined') return;
  devtoolsInstalled = true;
  const host = window.location?.hostname ?? '';
  if (host === 'localhost' || host === '127.0.0.1') {
    (window as Window & { __AD_EVENT_PROCESSOR_PROBE__?: ProbeDevtools }).__AD_EVENT_PROCESSOR_PROBE__ = {
      report: () => probeReport(),
      reset: () => probeReset(),
    };
  }
}

/**
 * Start a critical-path timing probe.
 */
export function probeStart(name: string): ProbeHandle {
  installDevtoolsProbe();
  return {
    name,
    t0: performance.now(),
    heap0: heapBytes(),
  };
}

/**
 * Record probe duration and optional heap delta.
 */
export function probeEnd(handle: ProbeHandle, meta: ProbeEndMeta = {}): ProbeEndResult {
  const ns = Math.round((performance.now() - handle.t0) * 1e6);
  const heapDelta = heapBytes() - handle.heap0;
  const allocs = meta.allocs ?? (heapDelta > 0 ? 1 : 0);
  const bytes = meta.bytes ?? Math.max(0, heapDelta);
  const prev = samples.get(handle.name) ?? { count: 0, totalNs: 0, totalAllocs: 0, heapBytes: 0 };
  prev.count += 1;
  prev.totalNs += ns;
  prev.totalAllocs += allocs;
  prev.heapBytes += bytes;
  samples.set(handle.name, prev);
  return { name: handle.name, ns, heapDelta, allocs, bytes };
}

/**
 * Return aggregated probe metrics keyed by operation name.
 */
export function probeReport(): ProbeReport {
  const out: ProbeReport = {};
  for (const [name, row] of samples.entries()) {
    const count = row.count || 1;
    out[name] = {
      count,
      nsPerOp: Math.round(row.totalNs / count),
      allocPerOp: Number((row.totalAllocs / count).toFixed(2)),
      bytesPerOp: Math.round(row.heapBytes / count),
    };
  }
  return out;
}

/**
 * Reset all stored probe samples.
 */
export function probeReset(): void {
  samples.clear();
}

/**
 * Record a route navigation probe.
 */
export function probeRouteChange(path: string): void {
  probeEnd(probeStart(`route:${path}`));
}

/**
 * Record a chart mount probe.
 */
export function probeChartMount(chartName: string, pointCount = 0): void {
  probeEnd(probeStart(`chart:${chartName}`), { bytes: pointCount });
}

/**
 * Record a worker round-trip probe.
 */
export function probeWorkerRoundTrip(workerName: string, bytes = 0): void {
  probeEnd(probeStart(`worker:${workerName}`), { bytes });
}

/**
 * Read heap usage when the runtime exposes it.
 */
function heapBytes(): number {
  if (typeof performance !== 'undefined') {
    const mem = (performance as PerformanceWithMemory).memory?.usedJSHeapSize;
    if (mem) return mem;
  }
  return 0;
}
