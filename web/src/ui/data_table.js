import { el } from '../lib/dom.js';
import { renderIcon } from './icon.js';

/**
 * @typedef {{ key: string, dir: 'asc' | 'desc' }} SortState
 */

/**
 * Create an empty table sort state object.
 *
 * @param {string} [key]
 * @param {'asc'|'desc'} [dir]
 * @returns {SortState}
 */
export function createSortState(key = '', dir = 'asc') {
  return { key, dir };
}

/**
 * Toggle or set sort key and direction on a sort state object.
 *
 * @param {SortState} state
 * @param {string} key
 * @returns {void}
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
 * Compare two sort keys for ascending order.
 *
 * @param {unknown} va
 * @param {unknown} vb
 * @returns {number}
 */
function compareValues(va, vb) {
  if (va == null && vb == null) return 0;
  if (va == null) return 1;
  if (vb == null) return -1;
  if (typeof va === 'number' && typeof vb === 'number') return va - vb;
  return String(va).localeCompare(String(vb), undefined, { sensitivity: 'base' });
}

/**
 * Sort table rows using accessors keyed by the active sort state.
 *
 * @param {unknown[]} rows
 * @param {SortState} state
 * @param {Record<string, (row: unknown) => unknown>} accessors
 * @param {{ rows?: unknown[], key?: string, dir?: string, sorted?: unknown[] }|null} [cache]
 * @returns {unknown[]}
 */
export function sortRows(rows, state, accessors, cache = null) {
  if (!state.key || !accessors[state.key]) return rows;
  if (cache
    && cache.rows === rows
    && cache.key === state.key
    && cache.dir === state.dir
    && cache.sorted) {
    return cache.sorted;
  }
  const acc = accessors[state.key];
  const dirMul = state.dir === 'desc' ? -1 : 1;
  const sorted = rows.slice();
  sorted.sort((a, b) => dirMul * compareValues(acc(a), acc(b)));
  if (cache) {
    cache.rows = rows;
    cache.key = state.key;
    cache.dir = state.dir;
    cache.sorted = sorted;
  }
  return sorted;
}

/**
 * Render a sortable table header cell with direction indicator.
 *
 * @param {string} label
 * @param {string} key
 * @param {SortState} sortState
 * @param {(key: string) => void} onSort
 * @returns {HTMLTableCellElement}
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
 * Build skeleton placeholder rows for a loading table.
 *
 * @param {number} colCount
 * @param {number} [rowCount]
 * @returns {HTMLTableRowElement[]}
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
 * Render a centered empty-state panel with optional action button.
 *
 * @param {{
 *   title: string,
 *   description?: string,
 *   icon?: string,
 *   actionLabel?: string,
 *   onAction?: () => void,
 * }} opts
 * @returns {HTMLElement}
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
