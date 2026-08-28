import { useCallback, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { buildDisputesListUrl } from '../helpers/disputes_api.js';
import { isBuyerBoundUser } from '../helpers/permissions.js';
import { useResource } from '../helpers/use_resource.js';
import type { DisputeListResponse } from '../helpers/disputes_api.js';
import { DisputesPanel } from '../ui/settings/disputes_panel.js';

const DEFAULT_LIMIT = 25;

function parseLimit(raw: string | null): number {
  const value = Number.parseInt(raw ?? '', 10);
  if (!Number.isFinite(value) || value <= 0) return DEFAULT_LIMIT;
  return Math.min(value, 100);
}

function parseOffset(raw: string | null): number {
  const value = Number.parseInt(raw ?? '', 10);
  if (!Number.isFinite(value) || value < 0) return 0;
  return value;
}

export function SettingsDisputesPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const user = auth.getUser();
  const buyerBound = isBuyerBoundUser(user?.role);
  const boundCustomerId = user?.customer_id ?? '';

  const limit = parseLimit(searchParams.get('limit'));
  const offset = parseOffset(searchParams.get('offset'));
  const customerId = searchParams.get('customer_id') ?? '';

  useEffect(() => {
    if (buyerBound && boundCustomerId && !searchParams.get('customer_id')) {
      const next = new URLSearchParams(searchParams);
      next.set('customer_id', boundCustomerId);
      setSearchParams(next, { replace: true });
    }
  }, [buyerBound, boundCustomerId, searchParams, setSearchParams]);

  const listUrl = buildDisputesListUrl({
    limit,
    offset,
    customer_id: customerId || undefined,
  });

  const { data, loading, error } = useResource<DisputeListResponse>(listUrl);

  const onCustomerApply = useCallback(
    (nextCustomerId: string) => {
      const next = new URLSearchParams(searchParams);
      if (nextCustomerId) next.set('customer_id', nextCustomerId);
      else next.delete('customer_id');
      next.delete('offset');
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  const onPageChange = useCallback(
    (nextOffset: number) => {
      const next = new URLSearchParams(searchParams);
      if (nextOffset > 0) next.set('offset', String(nextOffset));
      else next.delete('offset');
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  return (
    <DisputesPanel
      customerId={customerId}
      disputes={data?.disputes ?? []}
      total={data?.total ?? 0}
      limit={limit}
      offset={offset}
      loading={loading}
      error={error}
      onCustomerApply={onCustomerApply}
      onPageChange={onPageChange}
    />
  );
}
