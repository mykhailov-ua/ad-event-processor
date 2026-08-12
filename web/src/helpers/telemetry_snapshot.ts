import { probeReport } from './perf_probe.js';
import { apiTimingReport } from './api_timing.js';
import { guardTelemetryReport } from '../lib/async_guard.js';
import { memoryWatchSnapshot } from './memory_watch.js';

export type TelemetrySnapshot = {
  path: string;
  probes: ReturnType<typeof probeReport>;
  api: ReturnType<typeof apiTimingReport>;
  guards: ReturnType<typeof guardTelemetryReport>;
  memory: ReturnType<typeof memoryWatchSnapshot>;
  vitals?: unknown[];
};

/**
 * Build a client telemetry snapshot for RUM ingest and support bundle.
 */
export function buildTelemetrySnapshot(path = ''): TelemetrySnapshot {
  return {
    path: path || (typeof window !== 'undefined' ? window.location.pathname : ''),
    probes: probeReport(),
    api: apiTimingReport(),
    guards: guardTelemetryReport(),
    memory: memoryWatchSnapshot(),
  };
}
