import { el } from '../lib/dom.js';
import { statusClassFor } from '../helpers/status.js';
import { displayLabel } from '../helpers/display_labels.js';

export type StatusBadgeOpts = {
  kind?: 'campaign' | 'service' | 'invoice';
  label?: string;
};

/**
 * Render a colored status badge for a domain-specific status value.
 */
export function renderStatusBadge(status: string, opts: StatusBadgeOpts = {}): HTMLElement {
  const kind = opts.kind ?? 'campaign';
  const label = opts.label ?? displayLabel(status);
  const mod = statusClassFor(status, kind);
  return el('span', { className: `status-badge status-badge--${mod}` }, label);
}
