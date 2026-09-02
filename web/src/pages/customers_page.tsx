import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';

import { listCustomers } from '@/api/customers_api';
import type { CustomerListQuery } from '@/api/types';
import {
  CustomersDirectory,
  type CustomerSortField,
  type SortOrder,
} from '@/domains/customers/customers_directory';
import { useResource } from '@/api/use_resource';
import { clampListLimit, DEFAULT_LIST_LIMIT, parseListLimit, parseListOffset } from '@/lib/list_query';

function parseSort(raw: string | null): CustomerSortField {
  switch (raw) {
    case 'created_at':
      return 'created_at';
    case 'balance':
      return 'balance';
    case 'active_campaigns':
      return 'active_campaigns';
    default:
      return 'name';
  }
}

function parseOrder(raw: string | null): SortOrder {
  return raw === 'desc' ? 'desc' : 'asc';
}

function buildListQuery(params: URLSearchParams): CustomerListQuery {
  const parsedSort = parseSort(params.get('sort'));
  const serverSort: CustomerListQuery['sort'] =
    parsedSort === 'created_at' ? 'created_at' : 'name';

  return {
    limit: parseListLimit(params.get('limit')),
    offset: parseListOffset(params.get('offset')),
    sort: serverSort,
    order: parseOrder(params.get('order')),
  };
}

export function CustomersPage() {
  const [searchParams, setSearchParams] = useSearchParams();

  const query = useMemo(() => buildListQuery(searchParams), [searchParams]);
  const appliedSort = parseSort(searchParams.get('sort'));
  const appliedOrder = parseOrder(searchParams.get('order'));

  const { data, error, fetching } = useResource(
    (signal) => listCustomers(query, signal),
    [query.limit, query.offset, query.sort, query.order],
  );

  const updateQuery = useCallback(
    (patch: Partial<Omit<CustomerListQuery, 'sort'>> & { sort?: CustomerSortField; order?: SortOrder }) => {
      const next = new URLSearchParams(searchParams);
      const merged = { ...query, ...patch };

      next.set('limit', String(merged.limit ?? DEFAULT_LIST_LIMIT));
      next.set('offset', String(merged.offset ?? 0));
      next.set('sort', merged.sort ?? 'name');
      next.set('order', merged.order ?? 'asc');

      setSearchParams(next, { replace: true });
    },
    [query, searchParams, setSearchParams],
  );

  const onPageChange = useCallback(
    (nextOffset: number) => {
      updateQuery({ offset: Math.max(0, nextOffset) });
    },
    [updateQuery],
  );

  const onLimitChange = useCallback(
    (limit: number) => {
      updateQuery({ limit: clampListLimit(limit), offset: 0 });
    },
    [updateQuery],
  );

  const onColumnSort = useCallback(
    (field: CustomerSortField) => {
      const nextOrder =
        appliedSort === field && appliedOrder === 'asc' ? 'desc' : 'asc';
      updateQuery({
        sort: field,
        order: nextOrder,
        offset: 0,
      });
    },
    [appliedOrder, appliedSort, updateQuery],
  );

  const freshnessLabel = data?.freshness_label;

  return (
    <CustomersDirectory
      items={data?.items ?? []}
      total={data?.total ?? 0}
      limit={data?.limit ?? query.limit ?? DEFAULT_LIST_LIMIT}
      offset={data?.offset ?? query.offset ?? 0}
      appliedSort={appliedSort}
      appliedOrder={appliedOrder}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
      freshnessLabel={freshnessLabel}
      onColumnSort={onColumnSort}
      onPageChange={onPageChange}
      onLimitChange={onLimitChange}
    />
  );
}
