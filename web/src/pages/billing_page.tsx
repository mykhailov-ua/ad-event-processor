import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import {
  buildInvoicesListUrl,
  getBillingSummary,
  type BillingSummary,
  type InvoiceListResponse,
} from '../helpers/billing_api.js';
import { can, isBuyerBoundUser } from '../helpers/permissions.js';
import { useResource } from '../helpers/use_resource.js';
import { to } from '../lib/to.js';
import type { BillingFilterValues } from '../ui/billing/billing_filter.js';
import { BillingDirectory } from '../ui/billing/billing_directory.js';

const DEFAULT_LIMIT = 50;

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

function parseMinTotal(raw: string | null): number | undefined {
  if (!raw) return undefined;
  const value = Number.parseInt(raw, 10);
  if (!Number.isFinite(value)) return undefined;
  return value;
}

export function BillingPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const user = auth.getUser();
  const permissions = user?.permissions ?? [];
  const canSummary = can(permissions, 'shards:read');
  const buyerBound = isBuyerBoundUser(user?.role);
  const boundCustomerId = user?.customer_id ?? '';

  const limit = parseLimit(searchParams.get('limit'));
  const offset = parseOffset(searchParams.get('offset'));
  const customer_id = searchParams.get('customer_id') ?? '';
  const status = searchParams.get('status') ?? '';
  const month = searchParams.get('month') ?? '';
  const min_total = parseMinTotal(searchParams.get('min_total'));

  const [summary, setSummary] = useState<BillingSummary | null>(null);

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

  const filterValues: BillingFilterValues = useMemo(
    () => ({
      customer_id,
      status,
      month,
      min_total: searchParams.get('min_total') ?? '',
    }),
    [customer_id, status, month, searchParams]
  );

  const listUrl = buildInvoicesListUrl({
    limit,
    offset,
    customer_id: customer_id || undefined,
    status: status || undefined,
    month: month || undefined,
    min_total,
  });

  const { data, loading, error } = useResource<InvoiceListResponse>(listUrl);

  useEffect(() => {
    if (!canSummary) {
      setSummary(null);
      return undefined;
    }
    const ctrl = new AbortController();
    let cancelled = false;
    void (async () => {
      const [result, err] = await to(getBillingSummary(ctrl.signal));
      if (cancelled) return;
      if (err) {
        setSummary(null);
        return;
      }
      setSummary(result ?? null);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [canSummary, listUrl]);

  const onFilterApply = useCallback(
    (values: BillingFilterValues) => {
      patchParams({
        customer_id: values.customer_id || null,
        status: values.status || null,
        month: values.month || null,
        min_total: values.min_total || null,
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

  return (
    <BillingDirectory
      items={data?.items ?? []}
      total={data?.total ?? 0}
      limit={limit}
      offset={offset}
      filterValues={filterValues}
      summary={summary}
      loading={loading}
      error={error}
      onFilterApply={onFilterApply}
      onOffsetChange={onOffsetChange}
    />
  );
}
