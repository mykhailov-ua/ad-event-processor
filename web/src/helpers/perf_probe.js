/** @type {Map<string, { count: number, totalNs: number, totalAllocs: number, heapBytes: number }>} */
const samples = new Map();

let devtoolsInstalled = false;

/**
 * Expose probe report on window in dev builds only.
 *
 * @returns {void}
 */
function installDevtoolsProbe() {
  if (devtoolsInstalled || typeof window === 'undefined') return;
  devtoolsInstalled = true;
  const host = window.location?.hostname ?? '';
  if (host === 'localhost' || host === '127.0.0.1') {
    window.__ESPX_PROBE__ = {
      report: () => probeReport(),
      reset: () => probeReset(),
    };
  }
}

/**
 * Start a critical-path timing probe.
 *
 * @param {string} name
 * @returns {{ name: string, t0: number, heap0: number }}
 */
export function probeStart(name) {
  installDevtoolsProbe();
  return {
    name,
    t0: performance.now(),
    heap0: heapBytes(),
  };
}

/**
 * Record probe duration and optional heap delta.
 *
 * @param {{ name: string, t0: number, heap0: number }} handle
 * @param {{ allocs?: number, bytes?: number }} [meta]
 * @returns {{ name: string, ns: number, heapDelta: number, allocs: number, bytes: number }}
 */
export function probeEnd(handle, meta = {}) {
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
 *
 * @returns {Record<string, { count: number, nsPerOp: number, allocPerOp: number, bytesPerOp: number }>}
 */
export function probeReport() {
  /** @type {Record<string, { count: number, nsPerOp: number, allocPerOp: number, bytesPerOp: number }>} */
  const out = {};
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
 *
 * @returns {void}
 */
export function probeReset() {
  samples.clear();
}

/**
 * Record a route navigation probe.
 *
 * @param {string} path
 * @returns {void}
 */
export function probeRouteChange(path) {
  probeEnd(probeStart(`route:${path}`));
}

/**
 * Record a chart mount probe.
 *
 * @param {string} chartName
 * @param {number} pointCount
 * @returns {void}
 */
export function probeChartMount(chartName, pointCount = 0) {
  probeEnd(probeStart(`chart:${chartName}`), { bytes: pointCount });
}

/**
 * Record a worker round-trip probe.
 *
 * @param {string} workerName
 * @param {number} bytes
 * @returns {void}
 */
export function probeWorkerRoundTrip(workerName, bytes = 0) {
  probeEnd(probeStart(`worker:${workerName}`), { bytes });
}

/**
 * Read heap usage when the runtime exposes it.
 *
 * @returns {number}
 */
function heapBytes() {
  if (typeof performance !== 'undefined' && performance.memory?.usedJSHeapSize) {
    return performance.memory.usedJSHeapSize;
  }
  return 0;
}
