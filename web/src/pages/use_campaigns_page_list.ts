import { useMemo } from 'react';

import {
  fetchCampaignListMetricsBatch,
  listCampaigns,
} from '@/api/campaigns_api';
import { listCustomers } from '@/api/customers_api';
import { listSelfServeTemplates } from '@/api/selfserve_api';
import { listTeamMembers } from '@/api/team_api';
import type { CampaignListQuery } from '@/api/types';
import { useResource } from '@/api/use_resource';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';
import type { CampaignListColumnWidthProbe } from '@/domains/campaigns/list/campaigns_directory_types';
import type { CampaignsListFilterOption } from '@/domains/campaigns/list/campaigns_list_filter_select';
import {
  buildCampaignListWidthProbeQuery,
  listResponseCoversWidthProbeDataset,
  mergeCampaignIdsForMetricsBatch,
} from '@/domains/campaigns/list/campaign_list_width_probe';
import { campaignStatsQueryForRange } from '@/domains/campaigns/list/campaign_list_date_range';

export type UseCampaignsPageListArgs = {
  query: CampaignListQuery;
  statsQuery: ReturnType<typeof campaignStatsQueryForRange>;
  refreshToken: number;
  customerId: string | undefined;
  appliedOwnerUserId: string;
  createSectionOpen: boolean;
  templatesRefreshToken: number;
};

export function useCampaignsPageList({
  query,
  statsQuery,
  refreshToken,
  customerId,
  appliedOwnerUserId,
  createSectionOpen,
  templatesRefreshToken,
}: UseCampaignsPageListArgs) {
  const widthProbeQuery = useMemo(
    () => buildCampaignListWidthProbeQuery(query),
    [
      query.budget_max_micro,
      query.budget_min_micro,
      query.country,
      query.customer_id,
      query.owner_user_id,
      query.pacing_mode,
      query.q,
      query.status,
    ],
  );

  const { data, error, fetching } = useResource(
    (signal) => listCampaigns(query, signal),
    [
      query.customer_id,
      query.status,
      query.q,
      query.pacing_mode,
      query.budget_min_micro,
      query.budget_max_micro,
      query.owner_user_id,
      query.country,
      query.limit,
      query.offset,
      query.sort,
      query.order,
      query.from,
      query.to,
      refreshToken,
    ],
  );

  const listCoversWidthProbeDataset = useMemo(
    () => listResponseCoversWidthProbeDataset(data),
    [data],
  );

  const { data: widthProbeData } = useResource(
    (signal) => {
      if (listCoversWidthProbeDataset) {
        return Promise.resolve(undefined);
      }
      return listCampaigns(widthProbeQuery, signal);
    },
    [
      listCoversWidthProbeDataset,
      widthProbeQuery.budget_max_micro,
      widthProbeQuery.budget_min_micro,
      widthProbeQuery.country,
      widthProbeQuery.customer_id,
      widthProbeQuery.owner_user_id,
      widthProbeQuery.pacing_mode,
      widthProbeQuery.q,
      widthProbeQuery.status,
      refreshToken,
    ],
  );

  const campaignIds = useMemo(
    () => (data?.items ?? []).map((campaign) => campaign.id),
    [data?.items],
  );

  const widthProbeIds = useMemo(
    () => (widthProbeData?.items ?? []).map((campaign) => campaign.id),
    [widthProbeData?.items],
  );

  const metricsCampaignIds = useMemo(
    () =>
      listCoversWidthProbeDataset
        ? campaignIds
        : mergeCampaignIdsForMetricsBatch(campaignIds, widthProbeIds),
    [campaignIds, listCoversWidthProbeDataset, widthProbeIds],
  );

  const { data: metricsBatch } = useResource(
    (signal) => fetchCampaignListMetricsBatch(metricsCampaignIds, statsQuery, signal),
    [metricsCampaignIds.join(','), refreshToken, statsQuery.from, statsQuery.to],
  );

  const metricsById = metricsBatch?.metricsById;
  const marginsById = metricsBatch?.marginsById;

  const columnWidthProbe = useMemo((): CampaignListColumnWidthProbe | undefined => {
    const probeItems = listCoversWidthProbeDataset
      ? (data?.items ?? [])
      : (widthProbeData?.items ?? []);
    if (!probeItems.length) {
      return undefined;
    }
    return {
      items: probeItems,
      metricsById: metricsById ?? {},
      marginsById: marginsById ?? {},
    };
  }, [data?.items, listCoversWidthProbeDataset, marginsById, metricsById, widthProbeData?.items]);

  const countryOptions = useMemo((): CampaignsListFilterOption[] => {
    const codes = new Set<string>();
    for (const campaign of data?.items ?? []) {
      for (const code of campaign.target_countries ?? []) {
        if (code) {
          codes.add(code);
        }
      }
    }
    return [
      { value: '__all__', label: 'All countries' },
      ...[...codes].sort().map((code) => ({ value: code, label: code })),
    ];
  }, [data?.items]);

  const { data: teamMembersData } = useResource(
    (signal) => {
      if (!customerId) {
        return Promise.resolve(undefined);
      }
      return listTeamMembers({ customer_id: customerId, limit: 200 }, signal);
    },
    [customerId],
  );

  const ownerEmailById = useMemo(() => {
    const map: Record<string, string> = {};
    for (const member of teamMembersData?.items ?? []) {
      const userId = member.user_id;
      if (userId && member.email?.trim()) {
        map[userId] = member.email.trim();
      }
    }
    return map;
  }, [teamMembersData?.items]);

  const ownerOptions = useMemo((): CampaignsListFilterOption[] => {
    const options: CampaignsListFilterOption[] = [{ value: '__all__', label: 'All owners' }];
    const seen = new Set<string>(['__all__']);

    const addOwner = (userId: string, label: string) => {
      if (!userId || seen.has(userId)) {
        return;
      }
      seen.add(userId);
      options.push({ value: userId, label: label.trim() || userId.slice(0, 8) });
    };

    for (const member of teamMembersData?.items ?? []) {
      addOwner(member.user_id ?? '', member.email?.trim() || member.user_id || '');
    }
    for (const campaign of data?.items ?? []) {
      const userId = campaign.owner_user_id ?? '';
      addOwner(userId, ownerEmailById[userId] ?? userId);
    }
    if (appliedOwnerUserId) {
      addOwner(appliedOwnerUserId, ownerEmailById[appliedOwnerUserId] ?? appliedOwnerUserId);
    }
    return options;
  }, [appliedOwnerUserId, data?.items, ownerEmailById, teamMembersData?.items]);

  const statusTotals = data?.status_totals;
  const statusTotalsFetching = fetching && statusTotals == null;

  const { data: customersData, fetching: customersFetching } = useResource(
    (signal) => listCustomers({ limit: 100, sort: 'name', order: 'asc' }, signal),
    [],
  );

  const customerOptions = useMemo((): CustomerComboboxOption[] => {
    return (customersData?.items ?? [])
      .filter((customer) => customer.id)
      .map((customer) => ({
        id: customer.id!,
        name: customer.name?.trim() || customer.id!,
      }));
  }, [customersData?.items]);

  const customerNameById = useMemo(() => {
    const names: Record<string, string> = {};
    for (const customer of customerOptions) {
      names[customer.id] = customer.name;
    }
    return names;
  }, [customerOptions]);

  const shouldLoadTemplates = createSectionOpen && Boolean(customerId);

  const {
    data: templatesData,
    error: templatesError,
    fetching: templatesFetching,
  } = useResource(
    (signal) => {
      if (!shouldLoadTemplates || !customerId) {
        return Promise.resolve(undefined);
      }
      return listSelfServeTemplates(customerId, signal);
    },
    [customerId, templatesRefreshToken, shouldLoadTemplates],
  );

  return {
    data,
    error,
    fetching,
    columnWidthProbe,
    metricsById,
    marginsById,
    countryOptions,
    ownerOptions,
    ownerEmailById,
    statusTotals,
    statusTotalsLoading: statusTotalsFetching && statusTotals == null,
    customerOptions,
    customersLoading: customersFetching && customersData == null,
    customerNameById,
    templates: templatesData?.items ?? [],
    templatesError,
    templatesLoading: templatesFetching,
  };
}
