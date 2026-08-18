import { el } from '../lib/dom.js';
import { probeReport } from './perf_probe.js';

export function perfOverlayEnabled(): boolean {
  if (typeof window === 'undefined') return false;
  return new URLSearchParams(window.location.search).has('perf');
}

export function renderPerfBlock(id: string): HTMLElement | null {
  if (!perfOverlayEnabled()) return null;
  return el(
    'pre',
    { id, 'aria-label': 'Critical path metrics' },
    JSON.stringify(probeReport(), null, 2)
  );
}
