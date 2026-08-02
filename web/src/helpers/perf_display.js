import { el } from '../lib/dom.js';
import { probeReport } from './perf_probe.js';

/**
 * Test whether critical-path perf overlay is enabled (?perf on URL).
 *
 * @returns {boolean}
 */
export function perfOverlayEnabled() {
  if (typeof window === 'undefined') return false;
  return new URLSearchParams(window.location.search).has('perf');
}

/**
 * Build a perf metrics pre element when ?perf is set on the URL.
 *
 * @param {string} id
 * @returns {HTMLElement|null}
 */
export function renderPerfBlock(id) {
  if (!perfOverlayEnabled()) return null;
  return el('pre', { id, 'aria-label': 'Critical path metrics' },
    JSON.stringify(probeReport(), null, 2),
  );
}
