import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import {
  buildCampaignsListUrl,
  type CampaignListResponse,
  type CampaignSortField,
  type CampaignSortOrder,
} from '../helpers/campaigns_api.js';
import { can, isBuyerBoundUser } from '../helpers/permissions.js';
import { useResource } from '../helpers/use_resource.js';
import type { CampaignsFilterValues } from '../ui/campaigns/campaigns_filter.js';
import { CampaignsDirectory } from '../ui/campaigns/campaigns_directory.js';

const DEFAULT_LIMIT = 50;
const DEFAULT_SORT: CampaignSortField = 'updated_at';
const DEFAULT_ORDER: CampaignSortOrder = 'desc';

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

function parseSort(raw: string | null): CampaignSortField {
  if (raw === 'name' || raw === 'spend') return raw;
  return DEFAULT_SORT;
}

function parseOrder(raw: string | null): CampaignSortOrder {
  return raw === 'asc' ? 'asc' : DEFAULT_ORDER;
}

export function CampaignsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const user = auth.getUser();
  const permissions = user?.permissions ?? [];
  const canBulk = can(permissions, 'campaigns:write');
  const buyerBound = isBuyerBoundUser(user?.role);
  const boundCustomerId = user?.customer_id ?? '';

  const limit = parseLimit(searchParams.get('limit'));
  const offset = parseOffset(searchParams.get('offset'));
  const customer_id = searchParams.get('customer_id') ?? '';
  const status = searchParams.get('status') ?? '';
  const q = searchParams.get('q') ?? '';
  const pacing_mode = searchParams.get('pacing_mode') ?? '';
  const sort = parseSort(searchParams.get('sort'));
  const order = parseOrder(searchParams.get('order'));

  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set());

  useEffect(() => {
    if (buyerBound && boundCustomerId && !searchParams.get('customer_id')) {
      const next = new URLSearchParams(searchParams);
      next.set('customer_id', boundCustomerId);
      setSearchParams(next, { replace: true });
    }
  }, [buyerBound, boundCustomerId, searchParams, setSearchParams]);

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

  const filterValues: CampaignsFilterValues = useMemo(
    () => ({
      customer_id,
      status,
      q,
      pacing_mode,
      sort,
      order,
    }),
    [customer_id, status, q, pacing_mode, sort, order]
  );

  const listUrl = buildCampaignsListUrl({
    limit,
    offset,
    sort,
    order,
    customer_id: customer_id || undefined,
    status: status || undefined,
    q: q || undefined,
    pacing_mode: pacing_mode || undefined,
  });

  const { data, loading, error, reload } = useResource<CampaignListResponse>(listUrl);

  const onFilterApply = useCallback(
    (values: CampaignsFilterValues) => {
      patchParams({
        customer_id: values.customer_id || null,
        status: values.status || null,
        q: values.q || null,
        pacing_mode: values.pacing_mode || null,
        sort: values.sort,
        order: values.order,
        offset: '0',
      });
    },
    [patchParams]
  );

  const onSortHeader = useCallback(
    (field: CampaignSortField) => {
      const nextOrder: CampaignSortOrder =
        sort === field && order === 'asc' ? 'desc' : 'asc';
      patchParams({
        sort: field,
        order: nextOrder,
        offset: '0',
      });
    },
    [sort, order, patchParams]
  );

  const onOffsetChange = useCallback(
    (nextOffset: number) => {
      patchParams({ offset: String(nextOffset) });
    },
    [patchParams]
  );

  const onToggleRow = useCallback((id: string, checked: boolean) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (checked) next.add(id);
      else next.delete(id);
      return next;
    });
  }, []);

  const onToggleAll = useCallback((checked: boolean, ids: string[]) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      for (const id of ids) {
        if (checked) next.add(id);
        else next.delete(id);
      }
      return next;
    });
  }, []);

  const onClearSelection = useCallback(() => {
    setSelectedIds(new Set());
  }, []);

  const onBulkSuccess = useCallback(() => {
    reload();
  }, [reload]);

  useEffect(() => {
    setSelectedIds(new Set());
  }, [listUrl]);

  return (
    <CampaignsDirectory
      items={data?.items ?? []}
      total={data?.total ?? 0}
      limit={limit}
      offset={offset}
      filterValues={filterValues}
      loading={loading}
      error={error}
      canBulk={canBulk}
      customerScoped={Boolean(customer_id)}
      selectedIds={selectedIds}
      onFilterApply={onFilterApply}
      onSortHeader={onSortHeader}
      onOffsetChange={onOffsetChange}
      onToggleRow={onToggleRow}
      onToggleAll={onToggleAll}
      onClearSelection={onClearSelection}
      onBulkSuccess={onBulkSuccess}
    />
  );
}
