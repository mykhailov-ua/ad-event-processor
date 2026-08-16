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
