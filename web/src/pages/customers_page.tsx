import { useCallback, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import {
  buildCustomersListUrl,
  type CustomerListResponse,
  type CustomerSortField,
  type CustomerSortOrder,
} from '../helpers/customers_api.js';
import { isTenantUser } from '../helpers/permissions.js';
import { useResource } from '../helpers/use_resource.js';
import { CustomersDirectory } from '../ui/customers/customers_directory.js';

const DEFAULT_LIMIT = 50;
const DEFAULT_SORT: CustomerSortField = 'created_at';
const DEFAULT_ORDER: CustomerSortOrder = 'desc';

function parseLimit(raw: string | null): number {
  const value = Number.parseInt(raw ?? '', 10);
  if (!Number.isFinite(value) || value <= 0) return DEFAULT_LIMIT;
  return Math.min(value, 200);
}

function parseOffset(raw: string | null): number {
  const value = Number.parseInt(raw ?? '', 10);
  if (!Number.isFinite(value) || value < 0) return 0;
  return value;
}

function parseSort(raw: string | null): CustomerSortField {
  return raw === 'name' ? 'name' : DEFAULT_SORT;
}

function parseOrder(raw: string | null): CustomerSortOrder {
  return raw === 'asc' ? 'asc' : DEFAULT_ORDER;
}

export function CustomersPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const user = auth.getUser();
  const tenant = isTenantUser(user?.role);
  const tenantId = user?.customer_id;

  const limit = parseLimit(searchParams.get('limit'));
  const offset = parseOffset(searchParams.get('offset'));
  const sort = parseSort(searchParams.get('sort'));
  const order = parseOrder(searchParams.get('order'));

  useEffect(() => {
    if (tenant && tenantId) {
      navigate(`/customers/${tenantId}`, { replace: true });
    }
  }, [tenant, tenantId, navigate]);

  const listUrl = buildCustomersListUrl({ limit, offset, sort, order });

  const { data, loading, error } = useResource<CustomerListResponse>(
    tenant && tenantId ? null : listUrl,
    { skip: Boolean(tenant && tenantId) }
  );

  const patchParams = useCallback(
    (patch: Record<string, string | null>) => {
      const next = new URLSearchParams(searchParams);
      for (const [key, value] of Object.entries(patch)) {
        if (value === null || value === '') {
          next.delete(key);
        } else {
          next.set(key, value);
        }
      }
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  const onSortHeader = useCallback(
    (field: CustomerSortField) => {
      const nextOrder: CustomerSortOrder =
        sort === field && order === 'asc' ? 'desc' : 'asc';
      patchParams({
        sort: field,
        order: nextOrder,
        offset: '0',
      });
    },
    [sort, order, patchParams]
  );

  const onFilterApply = useCallback(
    (nextSort: CustomerSortField, nextOrder: CustomerSortOrder) => {
      patchParams({
        sort: nextSort,
        order: nextOrder,
        offset: '0',
      });
    },
    [patchParams]
  );

  const onOffsetChange = useCallback(
    (nextOffset: number) => {
      patchParams({ offset: String(nextOffset) });
    },
    [patchParams]
  );

  if (tenant && tenantId) {
    return <span className="text-muted">Redirecting...</span>;
  }

  return (
    <CustomersDirectory
      items={data?.items ?? []}
      total={data?.total ?? 0}
      limit={limit}
      offset={offset}
      sort={sort}
      order={order}
      loading={loading}
      error={error}
      onSortHeader={onSortHeader}
      onFilterApply={onFilterApply}
      onOffsetChange={onOffsetChange}
    />
  );
}
