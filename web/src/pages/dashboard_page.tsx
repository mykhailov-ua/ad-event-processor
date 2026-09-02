import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';

import { listCampaigns } from '@/api/campaigns_api';
import { listCustomers } from '@/api/customers_api';
import { getRoleDashboard, isDashboardRole } from '@/api/dashboards_api';
import { ApiError } from '@/api/client';
import type { DashboardRole } from '@/api/types';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';
import {
  RoleDashboardView,
} from '@/domains/dashboards/role_dashboard_view';
import type { DashboardRangePreset } from '@/domains/dashboards/buyer_dashboard_types';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';
import { dashboardPresetRange } from '@/lib/dashboard_range';
import { defaultReportRange } from '@/lib/report_paths';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';

function resolveDefaultRole(sessionRole: string | undefined, paramRole: string | undefined): DashboardRole {
  const normalized = (paramRole ?? sessionRole ?? 'buyer').toLowerCase();
  if (isDashboardRole(normalized)) {
    return normalized;
  }
  return 'buyer';
}

export function DashboardPage() {
  const { role: roleParam } = useParams<{ role: string }>();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const { session } = useSession();

  const defaultRange = useMemo(() => defaultReportRange('7d'), []);
  const role = resolveDefaultRole(session?.role, roleParam);

  const appliedCustomerId =
    searchParams.get('customer_id') ?? session?.default_customer_id ?? '';
  const appliedCampaignId = searchParams.get('campaign_id') ?? '';
  const appliedFrom = searchParams.get('from') ?? defaultRange.from;
  const appliedTo = searchParams.get('to') ?? defaultRange.to;

  const [draftRole, setDraftRole] = useState<DashboardRole>(role);
  const [draftCustomerId, setDraftCustomerId] = useState(appliedCustomerId);
  const [draftCampaignId, setDraftCampaignId] = useState(appliedCampaignId);
  const [draftFrom, setDraftFrom] = useState(toDatetimeLocalValue(appliedFrom));
  const [draftTo, setDraftTo] = useState(toDatetimeLocalValue(appliedTo));
  const [rangePreset, setRangePreset] = useState<DashboardRangePreset>('7d');

  const { data: customersData } = useResource(
    (signal) => listCustomers({ limit: 100, offset: 0, sort: 'name', order: 'asc' }, signal),
    [],
  );

  const customerOptions = useMemo((): CustomerComboboxOption[] => {
    return (customersData?.items ?? [])
      .filter((customer) => customer.id)
      .map((customer) => ({
        id: customer.id as string,
        name: customer.name ?? customer.id ?? '',
      }));
  }, [customersData?.items]);

  const { data: campaignsData } = useResource(
    (signal) => {
      const customerId = appliedCustomerId.trim();
      if (!customerId) {
        return Promise.resolve(undefined);
      }
      return listCampaigns({ customer_id: customerId, limit: 200, offset: 0 }, signal);
    },
    [appliedCustomerId],
  );

  const campaignOptions = useMemo(() => {
    return (campaignsData?.items ?? [])
      .filter((campaign) => campaign.id)
      .map((campaign) => ({
        id: campaign.id as string,
        name: campaign.name ?? campaign.id ?? '',
      }));
  }, [campaignsData?.items]);

  useEffect(() => {
    setDraftRole(role);
    setDraftCustomerId(appliedCustomerId);
    setDraftCampaignId(appliedCampaignId);
    setDraftFrom(toDatetimeLocalValue(appliedFrom));
    setDraftTo(toDatetimeLocalValue(appliedTo));
  }, [appliedCampaignId, appliedCustomerId, appliedFrom, appliedTo, role]);

  const shouldFetch = Boolean(appliedCustomerId);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return getRoleDashboard(
        role,
        {
          customer_id: appliedCustomerId,
          campaign_id: appliedCampaignId || undefined,
          from: appliedFrom,
          to: appliedTo,
        },
        signal,
      );
    },
    [appliedCampaignId, appliedCustomerId, appliedFrom, appliedTo, role, shouldFetch],
  );

  const licenseGated = error instanceof ApiError && error.status === 403;

  const commitFilters = useCallback(
    (next: {
      customerId: string;
      campaignId: string;
      from: string;
      to: string;
      nextRole?: DashboardRole;
    }) => {
      const nextRole = next.nextRole ?? draftRole;
      if (nextRole !== roleParam) {
        navigate(`/dashboards/${nextRole}`, { replace: true });
      }
      const params = new URLSearchParams();
      const customerId = next.customerId.trim();
      if (customerId) {
        params.set('customer_id', customerId);
      }
      const campaignId = next.campaignId.trim();
      if (campaignId) {
        params.set('campaign_id', campaignId);
      }
      params.set('from', next.from);
      params.set('to', next.to);
      setSearchParams(params, { replace: true });
    },
    [draftRole, navigate, roleParam, setSearchParams],
  );

  const onApply = useCallback(() => {
    commitFilters({
      customerId: draftCustomerId,
      campaignId: draftCampaignId,
      from: fromDatetimeLocalValue(draftFrom) ?? defaultRange.from,
      to: fromDatetimeLocalValue(draftTo) ?? defaultRange.to,
      nextRole: draftRole,
    });
  }, [
    commitFilters,
    defaultRange.from,
    defaultRange.to,
    draftCampaignId,
    draftCustomerId,
    draftFrom,
    draftRole,
    draftTo,
  ]);

  const onDraftRangeChange = useCallback(
    (from: string, to: string) => {
      setDraftFrom(from);
      setDraftTo(to);
      if (!draftCustomerId.trim()) {
        return;
      }
      const fromIso = fromDatetimeLocalValue(from);
      const toIso = fromDatetimeLocalValue(to);
      if (!fromIso || !toIso) {
        return;
      }
      commitFilters({
        customerId: draftCustomerId,
        campaignId: draftCampaignId,
        from: fromIso,
        to: toIso,
      });
    },
    [commitFilters, draftCampaignId, draftCustomerId],
  );

  const onRangePresetChange = useCallback(
    (preset: DashboardRangePreset) => {
      setRangePreset(preset);
      if (preset === 'custom') {
        return;
      }
      const range = dashboardPresetRange(preset);
      setDraftFrom(toDatetimeLocalValue(range.from));
      setDraftTo(toDatetimeLocalValue(range.to));
    },
    [],
  );

  const onDraftFromChange = useCallback((value: string) => {
    setRangePreset('custom');
    setDraftFrom(value);
  }, []);

  const onDraftToChange = useCallback((value: string) => {
    setRangePreset('custom');
    setDraftTo(value);
  }, []);

  const onDraftCampaignIdChange = useCallback(
    (value: string) => {
      setDraftCampaignId(value);
      if (!draftCustomerId.trim()) {
        return;
      }
      commitFilters({
        customerId: draftCustomerId,
        campaignId: value,
        from: fromDatetimeLocalValue(draftFrom) ?? defaultRange.from,
        to: fromDatetimeLocalValue(draftTo) ?? defaultRange.to,
      });
    },
    [commitFilters, defaultRange.from, defaultRange.to, draftCustomerId, draftFrom, draftTo],
  );

  const showApply = useMemo(() => {
    return draftCustomerId.trim() !== appliedCustomerId.trim();
  }, [appliedCustomerId, draftCustomerId]);

  const clickLogHref = useMemo(() => {
    if (!appliedCustomerId.trim()) {
      return undefined;
    }
    const params = new URLSearchParams();
    params.set('customer_id', appliedCustomerId);
    params.set('from', appliedFrom);
    params.set('to', appliedTo);
    if (appliedCampaignId) {
      params.set('campaign_id', appliedCampaignId);
    }
    return `/reports/click-log?${params.toString()}`;
  }, [appliedCampaignId, appliedCustomerId, appliedFrom, appliedTo]);

  return (
    <RoleDashboardView
      role={role}
      draftRole={draftRole}
      draftCustomerId={draftCustomerId}
      draftCampaignId={draftCampaignId}
      draftFrom={draftFrom}
      draftTo={draftTo}
      rangePreset={rangePreset}
      customerOptions={customerOptions}
      campaignOptions={campaignOptions}
      payload={data as Record<string, unknown> | undefined}
      fetching={fetching}
      error={licenseGated ? undefined : error}
      hasSnapshot={!shouldFetch || data != null || licenseGated}
      licenseGated={licenseGated}
      clickLogHref={clickLogHref}
      showApply={showApply}
      onDraftRoleChange={setDraftRole}
      onDraftCustomerIdChange={setDraftCustomerId}
      onDraftCampaignIdChange={onDraftCampaignIdChange}
      onDraftRangeChange={onDraftRangeChange}
      onRangePresetChange={onRangePresetChange}
      onDraftFromChange={onDraftFromChange}
      onDraftToChange={onDraftToChange}
      onApply={onApply}
    />
  );
}
