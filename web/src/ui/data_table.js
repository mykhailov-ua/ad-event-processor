import { el } from '../lib/dom.js';
import { renderIcon } from './icon.js';

/**
 * @typedef {{ key: string, dir: 'asc' | 'desc' }} SortState
 */

/**
 * @param {string} [key]
 * @param {'asc'|'desc'} [dir]
 * @returns {SortState}
 */
export function createSortState(key = '', dir = 'asc') {
  return { key, dir };
}

/**
 * @param {SortState} state
 * @param {string} key
 */
export function toggleSort(state, key) {
  if (state.key === key) {
    state.dir = state.dir === 'asc' ? 'desc' : 'asc';
  } else {
    state.key = key;
    state.dir = 'asc';
  }
}

/**
 * @param {unknown[]} rows
 * @param {SortState} state
 * @param {Record<string, (row: unknown) => unknown>} accessors
 */
export function sortRows(rows, state, accessors) {
  if (!state.key || !accessors[state.key]) return rows;
  const acc = accessors[state.key];
  const sorted = [...rows].sort((a, b) => {
    const va = acc(a);
    const vb = acc(b);
    if (va == null && vb == null) return 0;
    if (va == null) return 1;
    if (vb == null) return -1;
    if (typeof va === 'number' && typeof vb === 'number') return va - vb;
    return String(va).localeCompare(String(vb), undefined, { sensitivity: 'base' });
  });
  if (state.dir === 'desc') sorted.reverse();
  return sorted;
}

/**
 * @param {string} label
 * @param {string} key
 * @param {SortState} sortState
 * @param {(key: string) => void} onSort
 */
export function sortableTh(label, key, sortState, onSort) {
  const active = sortState.key === key;
  const iconName = active ? (sortState.dir === 'asc' ? 'chevron-up' : 'chevron-down') : 'arrow-up-down';
  return el('th', {
    scope: 'col',
    className: [
      'data-table__th--sortable',
      active ? 'data-table__th--sorted' : '',
    ].filter(Boolean).join(' '),
    'aria-sort': active ? (sortState.dir === 'asc' ? 'ascending' : 'descending') : 'none',
    onClick: (e) => {
      e.stopPropagation();
      onSort(key);
    },
    onKeydown: (e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        onSort(key);
      }
    },
    tabIndex: 0,
  },
    el('span', { className: 'data-table__th-label' }, label),
    renderIcon(iconName, { size: 13, className: 'data-table__sort-icon' }),
  );
}

/**
 * @param {number} colCount
 * @param {number} [rowCount]
 */
export function tableSkeletonRows(colCount, rowCount = 5) {
  return Array.from({ length: rowCount }, () =>
    el('tr', { className: 'data-table__row--skeleton', 'aria-hidden': 'true' },
      Array.from({ length: colCount }, () =>
        el('td', null, el('span', { className: 'skeleton-bar' })),
      ),
    ),
  );
}

/**
 * @param {{
 *   title: string,
 *   description?: string,
 *   icon?: string,
 *   actionLabel?: string,
 *   onAction?: () => void,
 * }} opts
 */
export function renderEmptyState(opts) {
  return el('div', { className: 'empty-state', style: { padding: '28px 16px', textAlign: 'center' } },
    renderIcon(opts.icon ?? 'file-text', { size: 28, className: 'empty-state__icon text-muted mb-2' }),
    el('div', { className: 'empty-state__title', style: { fontWeight: 600 } }, opts.title),
    opts.description
      ? el('div', { className: 'empty-state__desc text-muted', style: { fontSize: 13, marginTop: 4 } }, opts.description)
      : null,
    opts.actionLabel && opts.onAction
      ? el('button', {
        type: 'button',
        className: 'btn btn--secondary btn--sm empty-state__action',
        style: { marginTop: 12 },
        onClick: opts.onAction,
      }, opts.actionLabel)
      : null,
  );
}
