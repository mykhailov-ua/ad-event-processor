import { el } from '../lib/dom.js';
import { renderIcon } from './icon.js';
import { renderButton } from './button.js';

export type SortState = {
  key: string;
  dir: 'asc' | 'desc';
};

export type SortCache = {
  rows?: unknown[];
  key?: string;
  dir?: string;
  sorted?: unknown[];
};

/**
 * Create an empty table sort state object.
 */
export function createSortState(key = '', dir: 'asc' | 'desc' = 'asc'): SortState {
  return { key, dir };
}

/**
 * Toggle or set sort key and direction on a sort state object.
 */
export function toggleSort(state: SortState, key: string): void {
  if (state.key === key) {
    state.dir = state.dir === 'asc' ? 'desc' : 'asc';
  } else {
    state.key = key;
    state.dir = 'asc';
  }
}

/**
 * Compare two sort keys for ascending order.
 */
function compareValues(va: unknown, vb: unknown): number {
  if (va == null && vb == null) return 0;
  if (va == null) return 1;
  if (vb == null) return -1;
  if (typeof va === 'number' && typeof vb === 'number') return va - vb;
  return String(va).localeCompare(String(vb), undefined, { sensitivity: 'base' });
}

/**
 * Sort table rows using accessors keyed by the active sort state.
 */
export function sortRows<T>(
  rows: T[],
  state: SortState,
  accessors: Record<string, (row: T) => unknown>,
  cache: SortCache | null = null,
): T[] {
  if (!state.key || !accessors[state.key]) return rows;
  if (cache
    && cache.rows === rows
    && cache.key === state.key
    && cache.dir === state.dir
    && cache.sorted) {
    return cache.sorted as T[];
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
 */
export function sortableTh(
  label: string,
  key: string,
  sortState: SortState,
  onSort: (key: string) => void,
): HTMLTableCellElement {
  const active = sortState.key === key;
  const iconName = active ? (sortState.dir === 'asc' ? 'chevron-up' : 'chevron-down') : 'arrow-up-down';
  return el('th', {
    scope: 'col',
    className: [
      'data-table__th--sortable',
      active ? 'data-table__th--sorted' : '',
    ].filter(Boolean).join(' '),
    'aria-sort': active ? (sortState.dir === 'asc' ? 'ascending' : 'descending') : 'none',
    onClick: (e: Event) => {
      e.stopPropagation();
      onSort(key);
    },
    onKeydown: (e: Event) => {
      const ke = e as KeyboardEvent;
      if (ke.key === 'Enter' || ke.key === ' ') {
        ke.preventDefault();
        onSort(key);
      }
    },
    tabIndex: 0,
  },
    el('span', { className: 'data-table__th-label' }, label),
    renderIcon(iconName, { size: 13, className: 'data-table__sort-icon' }),
  ) as HTMLTableCellElement;
}

/**
 * Build skeleton placeholder rows for a loading table.
 */
export function tableSkeletonRows(colCount: number, rowCount = 5): HTMLTableRowElement[] {
  return Array.from({ length: rowCount }, () =>
    el('tr', { className: 'data-table__row--skeleton', 'aria-hidden': 'true' },
      Array.from({ length: colCount }, () =>
        el('td', null, el('span', { className: 'skeleton-bar' })),
      ),
    ) as HTMLTableRowElement,
  );
}

export type EmptyStateOpts = {
  title: string;
  description?: string;
  icon?: string;
  actionLabel?: string;
  onAction?: (() => void) | null;
};

/**
 * Render a centered empty-state panel with optional action button.
 */
export function renderEmptyState(opts: EmptyStateOpts): HTMLElement {
  return el('div', { className: 'empty-state' },
    renderIcon(opts.icon ?? 'file-text', { size: 28, className: 'empty-state__icon text-muted mb-2' }),
    el('div', { className: 'empty-state__title' }, opts.title),
    opts.description
      ? el('div', { className: 'empty-state__desc text-muted text-sm' }, opts.description)
      : null,
    opts.actionLabel && opts.onAction
      ? renderButton({
        label: opts.actionLabel,
        variant: 'secondary',
        size: 'sm',
        className: 'empty-state__action',
        onClick: opts.onAction,
      })
      : null,
  );
}

/**
 * Empty state inside a table cell (colspan row).
 */
export function renderEmptyTableCell(colSpan: number, opts: EmptyStateOpts): HTMLTableCellElement {
  return el('td', { colSpan, className: 'data-table__empty' },
    renderEmptyState(opts),
  ) as HTMLTableCellElement;
}

/**
 * Prev/next pagination with standardized buttons.
 */
export function renderPaginationBar(opts: {
  label: string;
  prevDisabled?: boolean;
  nextDisabled?: boolean;
  onPrev: () => void;
  onNext: () => void;
}): HTMLElement {
  return el('div', { className: 'pagination-bar cluster--actions' },
    renderButton({
      label: 'Prev',
      variant: 'secondary',
      size: 'sm',
      disabled: opts.prevDisabled,
      onClick: opts.onPrev,
    }),
    el('span', { className: 'text-muted text-xs pagination-bar__label' }, opts.label),
    renderButton({
      label: 'Next',
      variant: 'secondary',
      size: 'sm',
      disabled: opts.nextDisabled,
      onClick: opts.onNext,
    }),
  );
}
