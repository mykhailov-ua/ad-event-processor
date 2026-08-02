import { el } from '../lib/dom.js';
import { statusClassFor } from '../helpers/status.js';

/**
 * @param {string} status
 * @param {{ kind?: 'campaign'|'service'|'invoice', label?: string }} [opts]
 */
export function renderStatusBadge(status, opts = {}) {
  const kind = opts.kind ?? 'campaign';
  const label = opts.label ?? status ?? '—';
  const mod = statusClassFor(status, kind);
  return el('span', { className: `status-badge status-badge--${mod}` }, label);
}
