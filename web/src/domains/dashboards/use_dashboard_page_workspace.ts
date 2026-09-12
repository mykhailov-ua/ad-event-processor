import { useCallback, useEffect, useMemo, useState } from 'react';

import { getAdopsDashboard, getBuyerDashboard } from '@/api/dashboards_api';
import { useResource } from '@/api/use_resource';
import type {
  AdopsDashboardPayload,
  BuyerDashboardPayload,
} from '@/domains/dashboards/dashboard_types';
import {
  buildBuyerDashboardChartMock,
  isChartMockPreviewEnabled,
} from '@/domains/dashboards/dashboard_series_mock';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useSession } from '@/hooks/use_session';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';
import {
  dashboardPresetRange,
  type DashboardRangePreset,
  DASHBOARD_RANGE_PRESETS,
} from '@/lib/dashboard_range';
import { requireNonEmpty, toastValidationError, validationError } from '@/lib/admin_validation_error';

export type DashboardPageRole = 'buyer' | 'adops';

export type DashboardPageWorkspace = {
  role: DashboardPageRole;
  draftCustomerId: string;
  draftFrom: string;
  draftTo: string;
  draftPreset: DashboardRangePreset;
  appliedCustomerId: string;
  appliedFrom: string;
  appliedTo: string;
  data: BuyerDashboardPayload | AdopsDashboardPayload | undefined;
  fetching: boolean;
  revalidating: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  sessionStaleBanner?: string;
  exportTrueRoiHref: string;
  exportCampaignOverviewHref: string;
  chartMockPreview?: boolean;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
  onDraftPresetChange: (preset: DashboardRangePreset) => void;
  onApply: () => void;
  onRefresh: () => void;
};

function buildExportHref(
  reportKey: string,
  customerId: string,
  from: string,
  to: string
): string {
  const params = new URLSearchParams();
  params.set('report_key', reportKey);
  if (customerId) {
    params.set('customer_id', customerId);
  }
  if (from) {
    params.set('from', from);
  }
  if (to) {
    params.set('to', to);
  }
  return `/exports?${params.toString()}`;
}

export function useDashboardPageWorkspace(role: DashboardPageRole): DashboardPageWorkspace {
  const { session } = useSession();
  const [searchParams, { replaceSearchParams }] = useTransitionSearchParams();
  const { refreshToken, bumpRefresh } = useRefreshToken();

  const defaultRange = dashboardPresetRange('7d');
  const appliedCustomerId = searchParams.get('customer_id') ?? session?.default_customer_id ?? '';
  const appliedFrom = searchParams.get('from') ?? defaultRange.from;
  const appliedTo = searchParams.get('to') ?? defaultRange.to;
  const chartMockPreview =
    role === 'buyer' && isChartMockPreviewEnabled(searchParams.toString());

  const [draftCustomerId, setDraftCustomerId] = useState(appliedCustomerId);
  const [draftFrom, setDraftFrom] = useState(() => toDatetimeLocalValue(appliedFrom));
  const [draftTo, setDraftTo] = useState(() => toDatetimeLocalValue(appliedTo));
  const [draftPreset, setDraftPreset] = useState<DashboardRangePreset>('custom');

  useEffect(() => {
    setDraftCustomerId(appliedCustomerId);
    setDraftFrom(toDatetimeLocalValue(appliedFrom));
    setDraftTo(toDatetimeLocalValue(appliedTo));
  }, [appliedCustomerId, appliedFrom, appliedTo]);

  const queryDeps = [
    role,
    appliedCustomerId,
    appliedFrom,
    appliedTo,
    refreshToken,
    chartMockPreview,
  ] as const;

  const { data, error, fetching, revalidating } = useResource<
    BuyerDashboardPayload | AdopsDashboardPayload
  >(
    (signal) => {
      if (chartMockPreview) {
        return Promise.resolve(
          buildBuyerDashboardChartMock(appliedCustomerId, appliedFrom, appliedTo)
        );
      }
      const query = {
        customer_id: appliedCustomerId,
        from: appliedFrom,
        to: appliedTo,
      };
      if (role === 'buyer') {
        return getBuyerDashboard(query, signal);
      }
      return getAdopsDashboard(query, signal);
    },
    queryDeps
  );

  const onDraftPresetChange = useCallback((preset: DashboardRangePreset) => {
    setDraftPreset(preset);
    if (preset === 'custom') {
      return;
    }
    const range = dashboardPresetRange(preset);
    setDraftFrom(toDatetimeLocalValue(range.from));
    setDraftTo(toDatetimeLocalValue(range.to));
  }, []);

  const onApply = useCallback(() => {
    const customerCheck = requireNonEmpty(draftCustomerId, 'Customer ID', 'customer_id');
    if (!customerCheck.ok) {
      toastValidationError(customerCheck.error);
      return;
    }
    const fromIso = fromDatetimeLocalValue(draftFrom);
    const toIso = fromDatetimeLocalValue(draftTo);
    if (!fromIso || !toIso) {
      toastValidationError(
        validationError('Enter valid from and to timestamps', { field: 'from' })
      );
      return;
    }
    const next = new URLSearchParams(searchParams);
    next.set('customer_id', draftCustomerId.trim());
    next.set('from', fromIso);
    next.set('to', toIso);
    replaceSearchParams(next);
  }, [draftCustomerId, draftFrom, draftTo, replaceSearchParams, searchParams]);

  const onRefresh = useCoalescedBumpRefresh(bumpRefresh, fetching || revalidating);

  const exportTrueRoiHref = useMemo(
    () => buildExportHref('true-roi', appliedCustomerId, appliedFrom, appliedTo),
    [appliedCustomerId, appliedFrom, appliedTo]
  );

  const exportCampaignOverviewHref = useMemo(
    () => buildExportHref('campaign-overview', appliedCustomerId, appliedFrom, appliedTo),
    [appliedCustomerId, appliedFrom, appliedTo]
  );

  return {
    role,
    draftCustomerId,
    draftFrom,
    draftTo,
    draftPreset,
    appliedCustomerId,
    appliedFrom,
    appliedTo,
    data,
    fetching,
    revalidating,
    error,
    hasSnapshot: data !== undefined,
    sessionStaleBanner: session?.stale_banner,
    exportTrueRoiHref,
    exportCampaignOverviewHref,
    chartMockPreview,
    onDraftCustomerIdChange: setDraftCustomerId,
    onDraftFromChange: setDraftFrom,
    onDraftToChange: setDraftTo,
    onDraftPresetChange,
    onApply,
    onRefresh,
  };
}

export { DASHBOARD_RANGE_PRESETS };
