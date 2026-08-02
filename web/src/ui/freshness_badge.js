import { el } from '../lib/dom.js';

/**
 * Render a ClickHouse data freshness indicator when lag or staleness is known.
 *
 * @param {{ stale?: boolean, lagSeconds?: number }} opts
 * @returns {HTMLElement|null}
 */
export function renderFreshnessBadge(opts) {
  const lag = opts.lagSeconds ?? 0;
  const stale = Boolean(opts.stale);
  const lagText = lag > 0 ? `${lag}s lag` : 'no lag data';
  const title = stale
    ? `ClickHouse data may be stale. ${lagText}.`
    : `ClickHouse data is fresh. ${lagText}.`;

  if (!stale && lag === 0) return null;

  return el('span', {
    className: stale ? 'freshness-badge freshness-badge--stale' : 'freshness-badge freshness-badge--ok',
    title,
  }, stale ? `Stale · ${lagText}` : `Fresh · ${lagText}`);
}
