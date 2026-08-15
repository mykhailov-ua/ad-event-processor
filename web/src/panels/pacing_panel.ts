import { el } from '../lib/dom.js';

/**
 * Render buyer pacing health skeleton (no financial fields).
 *
 * @param {{
 *   status?: string,
 *   pacingMode?: string,
 *   impressions7d?: number,
 *   deliveryPct?: number|null,
 * }} input
 * @returns {HTMLElement}
 */
export function renderPacingPanel(input: any) {
  const status = String(input.status ?? '').toUpperCase();
  const mode = input.pacingMode ?? 'even';
  const impr = Number(input.impressions7d ?? 0);
  const delivery = input.deliveryPct;

  let health = 'on-track';
  let detail = 'Delivery within expected range for the period.';
  if (status === 'PAUSED') {
    health = 'paused';
    detail = 'Campaign is paused; no delivery expected.';
  } else if (mode !== 'even' && mode !== '') {
    health = 'drift';
    detail = `Non-even pacing mode (${mode}); monitor delivery closely.`;
  } else if (impr === 0 && status === 'ACTIVE') {
    health = 'underspend';
    detail = 'No impressions in the last 7 days.';
  }

  return el('section', { 'data-testid': 'pacing-panel' },
    el('h3', null, 'Pacing health'),
    el('dl', null,
      el('dt', null, 'Status'),
      el('dd', null, health),
      el('dt', null, 'Pacing mode'),
      el('dd', null, mode),
      el('dt', null, 'Impressions (7d)'),
      el('dd', null, String(impr)),
      delivery != null
        ? el('dt', null, 'Delivery vs expected')
        : null,
      delivery != null
        ? el('dd', null, `${delivery}%`)
        : null,
    ),
    el('p', null, detail),
    el('p', null,
      el('a', { href: '/campaigns/portfolio' }, 'Portfolio (pacing drift)'),
      ' · ',
      el('a', { href: '/reports/placements' }, 'Placements'),
    ),
  );
}
