import { probeReport } from './perf_probe.js';
import { apiTimingReport } from './api_timing.js';
import { guardTelemetryReport } from '../lib/async_guard.js';
import { memoryWatchSnapshot } from './memory_watch.js';

/**
 * Build a client telemetry snapshot for RUM ingest and support bundle.
 *
 * @param {string} [path]
 * @returns {object}
 */
export function buildTelemetrySnapshot(path = '') {
  return {
    path: path || (typeof window !== 'undefined' ? window.location.pathname : ''),
    probes: probeReport(),
    api: apiTimingReport(),
    guards: guardTelemetryReport(),
    memory: memoryWatchSnapshot(),
  };
}
