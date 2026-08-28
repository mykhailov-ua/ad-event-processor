import { api } from './api_client.js';
import { buildTelemetrySnapshot, type TelemetrySnapshot } from './telemetry_snapshot.js';

const SAMPLE_RATE = 0.05;
const FLUSH_INTERVAL_MS = 60_000;

type VitalSample =
  | { name: 'LCP'; valueMs: number }
  | { name: 'CLS'; value: number }
  | { name: 'INP'; valueMs: number };

type LayoutShiftEntry = PerformanceEntry & {
  value: number;
  hadRecentInput: boolean;
};

type EventTimingEntry = PerformanceEntry & {
  processingStart: number;
};

let flushTimer: ReturnType<typeof setInterval> | null = null;

const vitalsBuffer: VitalSample[] = [];

function shouldSample(): boolean {
  return Math.random() < SAMPLE_RATE;
}

function observeVitals(): void {
  if (typeof PerformanceObserver === 'undefined') return;

  try {
    const lcp = new PerformanceObserver((list) => {
      const entries = list.getEntries();
      const last = entries[entries.length - 1];
      if (last) vitalsBuffer.push({ name: 'LCP', valueMs: Math.round(last.startTime) });
    });
    lcp.observe({ type: 'largest-contentful-paint', buffered: true });
  } catch {}

  try {
    const cls = new PerformanceObserver((list) => {
      let score = 0;
      for (const entry of list.getEntries()) {
        const shift = entry as LayoutShiftEntry;
        if (!shift.hadRecentInput) score += shift.value;
      }
      if (score > 0) vitalsBuffer.push({ name: 'CLS', value: Number(score.toFixed(4)) });
    });
    cls.observe({ type: 'layout-shift', buffered: true });
  } catch {}

  try {
    const inp = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        const timing = entry as EventTimingEntry;
        const delay = timing.processingStart - timing.startTime;
        vitalsBuffer.push({ name: 'INP', valueMs: Math.round(delay) });
      }
    });
    inp.observe({
      type: 'event',
      buffered: true,
      durationThreshold: 40,
    } as PerformanceObserverInit);
  } catch {}
}

async function flushRUM(): Promise<void> {
  if (!shouldSample()) return;
  const snapshot: TelemetrySnapshot = buildTelemetrySnapshot();
  if (vitalsBuffer.length > 0) {
    snapshot.vitals = [...vitalsBuffer];
    vitalsBuffer.length = 0;
  }
  try {
    await api('/api/v1/ops/rum', {
      method: 'POST',
      body: JSON.stringify(snapshot),
    });
  } catch {}
}

export type RumCollectorHandle = {
  stop: () => void;
};

export function startRUMCollector(): RumCollectorHandle {
  observeVitals();
  if (flushTimer) clearInterval(flushTimer);
  flushTimer = setInterval(() => {
    flushRUM();
  }, FLUSH_INTERVAL_MS);
  return {
    stop() {
      if (flushTimer) clearInterval(flushTimer);
      flushTimer = null;
    },
  };
}

export function flushRUMNow(): Promise<void> {
  return flushRUM();
}
