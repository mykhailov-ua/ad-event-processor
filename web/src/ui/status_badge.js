import { el } from '../lib/dom.js';
import { statusClassFor } from '../helpers/status.js';
import { displayLabel } from '../helpers/display_labels.js';

/**
 * Render a colored status badge for a domain-specific status value.
 *
 * @param {string} status
 * @param {{ kind?: 'campaign'|'service'|'invoice', label?: string }} [opts]
 * @returns {HTMLElement}
 */
export function renderStatusBadge(status, opts = {}) {
  const kind = opts.kind ?? 'campaign';
  const label = opts.label ?? displayLabel(status);
  const mod = statusClassFor(status, kind);
  return el('span', { className: `status-badge status-badge--${mod}` }, label);
}
