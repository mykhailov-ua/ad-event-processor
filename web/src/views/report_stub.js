import { el, replaceChildren } from '../lib/dom.js';
import { renderStubBanner } from '../ui/stub_banner.js';

/** @type {Record<string, string>} */
const STUB_TITLES = {
  'campaign-unit-economics': 'Unit economics',
  'source-margin': 'Source margin',
  'traffic-sources': 'Traffic sources',
  'source-quality': 'Source quality',
  'spend-velocity': 'Spend velocity',
  'campaign-geo-device': 'Geo / device',
  'geo-roi': 'Geo ROI',
  'daypart-heatmap': 'Daypart heatmap',
  'pacing-drift': 'Pacing drift',
  'postback-reconciliation': 'Postback reconciliation',
  'ivt-by-source': 'IVT by source',
  'discrepancy-buy-sell': 'Buy/sell discrepancy',
  'campaign-overview': 'Campaign overview',
  'customer-portfolio': 'Customer portfolio',
};

/**
 * @param {HTMLElement} container
 * @param {{ params: Record<string, string> }} ctx
 */
export function mount(container, ctx) {
  const reportKey = ctx.params.reportKey;
  const title = STUB_TITLES[reportKey] ?? reportKey ?? 'Report';

  replaceChildren(container,
    el('div', { className: 'page-header' },
      el('div', { className: 'page-header__row' },
        el('h1', { className: 'page-header__title' }, title),
      ),
    ),
    renderStubBanner({
      message: 'Endpoint is planned but not implemented (501). Use placements or keywords reports.',
    }),
    el('p', { style: { marginTop: 16, fontSize: 13 } },
      el('a', { href: '/reports/placements', style: { color: 'var(--accent)' } },
        'Report: placements',
      ),
      ' · ',
      el('a', { href: '/reports/keywords', style: { color: 'var(--accent)' } }, 'Keywords'),
    ),
  );

  return {};
}
