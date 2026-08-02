import { api } from './api_client.js';
import { buildTelemetrySnapshot } from './telemetry_snapshot.js';

const SAMPLE_RATE = 0.05;
const FLUSH_INTERVAL_MS = 60_000;

/** @type {ReturnType<typeof setInterval>|null} */
let flushTimer = null;

/** @type {object[]} */
const vitalsBuffer = [];

/**
 * Return whether this session should sample RUM events.
 *
 * @returns {boolean}
 */
function shouldSample() {
  return Math.random() < SAMPLE_RATE;
}

/**
 * Observe Web Vitals when supported by the browser.
 *
 * @returns {void}
 */
function observeVitals() {
  if (typeof PerformanceObserver === 'undefined') return;

  try {
    const lcp = new PerformanceObserver((list) => {
      const entries = list.getEntries();
      const last = entries[entries.length - 1];
      if (last) vitalsBuffer.push({ name: 'LCP', valueMs: Math.round(last.startTime) });
    });
    lcp.observe({ type: 'largest-contentful-paint', buffered: true });
  } catch { /* unsupported */ }

  try {
    const cls = new PerformanceObserver((list) => {
      let score = 0;
      for (const entry of list.getEntries()) {
        if (!entry.hadRecentInput) score += entry.value;
      }
      if (score > 0) vitalsBuffer.push({ name: 'CLS', value: Number(score.toFixed(4)) });
    });
    cls.observe({ type: 'layout-shift', buffered: true });
  } catch { /* unsupported */ }

  try {
    const inp = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        const delay = entry.processingStart - entry.startTime;
        vitalsBuffer.push({ name: 'INP', valueMs: Math.round(delay) });
      }
    });
    inp.observe({ type: 'event', buffered: true, durationThreshold: 40 });
  } catch { /* unsupported */ }
}

/**
 * Flush buffered vitals and telemetry to the control plane.
 *
 * @returns {Promise<void>}
 */
async function flushRUM() {
  if (!shouldSample()) return;
  const snapshot = buildTelemetrySnapshot();
  if (vitalsBuffer.length > 0) {
    snapshot.vitals = [...vitalsBuffer];
    vitalsBuffer.length = 0;
  }
  try {
    await api('/api/v1/ops/rum', {
      method: 'POST',
      body: JSON.stringify(snapshot),
    });
  } catch {
    // RUM must not break the admin UI.
  }
}

/**
 * Start periodic RUM sampling for the admin session.
 *
 * @returns {{ stop: () => void }}
 */
export function startRUMCollector() {
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

/**
 * Force an immediate RUM flush (e.g. before support bundle download).
 *
 * @returns {Promise<void>}
 */
export function flushRUMNow() {
  return flushRUM();
}
