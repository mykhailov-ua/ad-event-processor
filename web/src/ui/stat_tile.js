import { el } from '../lib/dom.js';

/**
 * Render a single KPI tile for report grids.
 *
 * @param {string} label
 * @param {string|number} value
 * @returns {HTMLElement}
 */
export function renderStatTile(label, value) {
  return el('div', { className: 'stat-tile' },
    el('div', { className: 'stat-tile__label' }, label),
    el('div', { className: 'stat-tile__value' }, String(value ?? 0)),
  );
}

/**
 * Render a horizontal row of stat tiles with standard spacing.
 *
 * @param {...(HTMLElement|null|undefined|false)} tiles
 * @returns {HTMLElement}
 */
export function renderStatsRow(...tiles) {
  return el('div', { className: 'stats-row section-block' }, ...tiles.filter(Boolean));
}

/**
 * Wrap content in a spaced section block below page header/filters.
 *
 * @param {string} title
 * @param {...(HTMLElement|null|undefined|false)} children
 * @returns {HTMLElement}
 */
export function renderSubsection(title, ...children) {
  return el('section', { className: 'section-block' },
    el('h2', { className: 'subsection-title' }, title),
    ...children.filter(Boolean),
  );
}
