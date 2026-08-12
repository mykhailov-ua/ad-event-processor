import { el } from '../lib/dom.js';
import { probeReport } from './perf_probe.js';

/**
 * Test whether critical-path perf overlay is enabled (?perf on URL).
 */
export function perfOverlayEnabled(): boolean {
  if (typeof window === 'undefined') return false;
  return new URLSearchParams(window.location.search).has('perf');
}

/**
 * Build a perf metrics pre element when ?perf is set on the URL.
 */
export function renderPerfBlock(id: string): HTMLElement | null {
  if (!perfOverlayEnabled()) return null;
  return el('pre', { id, 'aria-label': 'Critical path metrics' },
    JSON.stringify(probeReport(), null, 2),
  );
}
