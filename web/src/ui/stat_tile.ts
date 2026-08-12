import { el } from '../lib/dom.js';

type OptionalChild = HTMLElement | null | undefined | false;

/**
 * Render a single KPI tile for report grids.
 */
export function renderStatTile(label: string, value: string | number): HTMLElement {
  return el('div', { className: 'stat-tile' },
    el('div', { className: 'stat-tile__label' }, label),
    el('div', { className: 'stat-tile__value' }, String(value ?? 0)),
  );
}

/**
 * Render a horizontal row of stat tiles with standard spacing.
 */
export function renderStatsRow(...tiles: OptionalChild[]): HTMLElement {
  return el('div', { className: 'stats-row section-block' }, ...tiles.filter(Boolean));
}

/**
 * Wrap content in a spaced section block below page header/filters.
 */
export function renderSubsection(title: string, ...children: OptionalChild[]): HTMLElement {
  return el('section', { className: 'section-block' },
    el('h2', { className: 'subsection-title' }, title),
    ...children.filter(Boolean),
  );
}
