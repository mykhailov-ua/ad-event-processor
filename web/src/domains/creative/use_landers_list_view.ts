import { useMemo, useState } from 'react';

import type { Lander } from '@/api/types';
import {
  resolveLanderHostingKind,
  resolveLanderPrimaryUrl,
} from '@/domains/creative/lander_list_helpers';
import type { LanderHostingFilter } from '@/domains/creative/landers_list_types';
import { listPageRange } from '@/lib/list_page_stats';

const DEFAULT_PAGE_SIZE = 25;

export type LandersHostingCounts = {
  total: number;
  external: number;
  hosted: number;
  unconfigured: number;
};

export function countLandersByHosting(items: Lander[]): LandersHostingCounts {
  let external = 0;
  let hosted = 0;
  let unconfigured = 0;
  for (const row of items) {
    const kind = resolveLanderHostingKind(row);
    if (kind === 'hosted') {
      hosted += 1;
    } else if (kind === 'external') {
      external += 1;
    } else {
      unconfigured += 1;
    }
  }
  return {
    total: items.length,
    external,
    hosted,
    unconfigured,
  };
}

export function filterLanders(
  items: Lander[],
  search: string,
  hosting: LanderHostingFilter
): Lander[] {
  const query = search.trim().toLowerCase();
  return items.filter((row) => {
    const kind = resolveLanderHostingKind(row);
    if (hosting === 'external' && kind !== 'external') {
      return false;
    }
    if (hosting === 'hosted' && kind !== 'hosted') {
      return false;
    }
    if (!query) {
      return true;
    }
    const primaryUrl = resolveLanderPrimaryUrl(row).toLowerCase();
    return (
      row.name.toLowerCase().includes(query) ||
      row.id.toLowerCase().includes(query) ||
      primaryUrl.includes(query)
    );
  });
}

export function useLandersListView(items: Lander[] | undefined) {
  const [draftSearch, setDraftSearch] = useState('');
  const [hostingFilter, setHostingFilter] = useState<LanderHostingFilter>('');
  const [limit, setLimit] = useState(DEFAULT_PAGE_SIZE);
  const [offset, setOffset] = useState(0);

  const allItems = items ?? [];
  const hostingCounts = useMemo(() => countLandersByHosting(allItems), [allItems]);

  const filteredItems = useMemo(
    () => filterLanders(allItems, draftSearch, hostingFilter),
    [allItems, draftSearch, hostingFilter]
  );

  const total = filteredItems.length;
  const pageItems = useMemo(
    () => filteredItems.slice(offset, offset + limit),
    [filteredItems, limit, offset]
  );

  const filtersActive = draftSearch.trim().length > 0 || hostingFilter !== '';

  const resetPage = () => setOffset(0);

  const onDraftSearchChange = (value: string) => {
    setDraftSearch(value);
    setOffset(0);
  };

  const onHostingFilterChange = (value: LanderHostingFilter) => {
    setHostingFilter(value);
    setOffset(0);
  };

  const onPageChange = (nextOffset: number) => {
    setOffset(Math.max(0, nextOffset));
  };

  const onPageSizeChange = (nextLimit: number) => {
    setLimit(nextLimit);
    setOffset(0);
  };

  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;
  const page = Math.floor(offset / limit) + 1;
  const pageCount = total === 0 ? 1 : Math.ceil(total / limit);
  const { rangeStart, rangeEnd } = listPageRange(total, limit, offset, pageItems.length);
  const rangeLabel = total === 0 ? '0 of 0' : `Showing ${rangeStart}-${rangeEnd} of ${total}`;

  return {
    pageItems,
    filteredTotal: total,
    hostingCounts,
    filtersActive,
    draftSearch,
    hostingFilter,
    limit,
    offset,
    canGoPrev,
    canGoNext,
    page,
    pageCount,
    rangeLabel,
    onDraftSearchChange,
    onHostingFilterChange,
    onPageChange,
    onPageSizeChange,
    resetPage,
  };
}
