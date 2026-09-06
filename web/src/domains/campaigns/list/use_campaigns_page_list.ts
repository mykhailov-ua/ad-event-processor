// L2 fetch orchestrator for campaigns directory (pages/campaigns_page.tsx).
// L3 selection/export/column prefs: use_campaigns_directory_workspace.ts.
// Campaign list workspace fetch fan-out (RF-9; Phase 6 trim):
// Refresh lanes (up to 4 when paginated and filter totals not capped):
// - GET list campaigns
// - GET width-probe list (after main list snapshot; skipped when page rows cover probe dataset)
// - POST metrics batch
// - POST filter totals (skipped when total > CAMPAIGN_LIST_FILTER_TOTALS_MAX)
// Mount-once / scoped (not on list refreshToken):
// - GET list facets (customer_id only)
// - GET customers combobox (session cache)
// - GET self-serve templates (create overlay open only)
import { useEffect, useMemo } from 'react';

import {
  fetchCampaignListFacets,
  fetchCampaignListMetricsBatch,
  fetchCampaignListMetricsTotals,
  listCampaigns,
} from '@/api/campaigns_api';
import { listSelfServeTemplates } from '@/api/selfserve_api';
import { fetchCustomersComboboxCached } from '@/lib/customers_combobox_cache';
import type { CampaignListQuery } from '@/api/types';
import { useResource } from '@/api/use_resource';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';
import type { CampaignListColumnWidthProbe } from '@/domains/campaigns/list/campaigns_directory_types';
import {
  buildCampaignListCountryOptions,
  buildCampaignListOwnerEmailById,
  buildCampaignListOwnerOptions,
} from '@/domains/campaigns/list/campaign_list_filter_options';
import { isCampaignListAuxEndpointUnavailable } from '@/domains/campaigns/list/campaign_list_aux_error';
import { resolveCampaignListFacets } from '@/domains/campaigns/list/campaign_list_facets_source';
import {
  buildCampaignListWidthProbeQuery,
  listResponseCoversWidthProbeDataset,
  mergeCampaignIdsForMetricsBatch,
  shouldFetchCampaignListWidthProbe,
} from '@/domains/campaigns/list/campaign_list_width_probe';
import { campaignStatsQueryForRange } from '@/domains/campaigns/list/campaign_list_date_range';
import { campaignListFilterTotalsFromApi } from '@/domains/campaigns/list/campaign_list_filter_totals';
import { CAMPAIGN_LIST_FILTER_TOTALS_MAX } from '@/domains/campaigns/list/campaign_list_limits';
import {
  fetchCampaignListCached,
  invalidateCampaignListResponseCache,
} from '@/domains/campaigns/list/campaign_list_response_cache';

export type UseCampaignsPageListArgs = {
  query: CampaignListQuery;
  statsQuery: ReturnType<typeof campaignStatsQueryForRange>;
  refreshToken: number;
  customerId: string | undefined;
  createCustomerId: string;
  appliedOwnerUserId: string;
  createSectionOpen: boolean;
  templatesRefreshToken: number;
};

export function useCampaignsPageList({
  query,
  statsQuery,
  refreshToken,
  customerId,
  createCustomerId,
  appliedOwnerUserId,
  createSectionOpen,
  templatesRefreshToken,
}: UseCampaignsPageListArgs) {
  useEffect(() => {
    invalidateCampaignListResponseCache();
  }, [refreshToken]);

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
    ]
  );

  const { data, error, fetching } = useResource(
    (signal) => fetchCampaignListCached(query, statsQuery.from, statsQuery.to, signal),
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
    ]
  );

  const listCoversWidthProbeDataset = useMemo(
    () => listResponseCoversWidthProbeDataset(data),
    [data]
  );

  const shouldFetchWidthProbeList = shouldFetchCampaignListWidthProbe(data);

  const { data: widthProbeData } = useResource(
    (signal) => {
      if (!shouldFetchWidthProbeList) {
        return Promise.resolve(undefined);
      }
      return listCampaigns(widthProbeQuery, signal);
    },
    [
      shouldFetchWidthProbeList,
      widthProbeQuery.budget_min_micro,
      widthProbeQuery.budget_max_micro,
      widthProbeQuery.country,
      widthProbeQuery.customer_id,
      widthProbeQuery.owner_user_id,
      widthProbeQuery.pacing_mode,
      widthProbeQuery.q,
      widthProbeQuery.status,
      refreshToken,
    ]
  );

  const campaignIds = useMemo(() => data?.items?.map((campaign) => campaign.id), [data?.items]);

  const widthProbeIds = useMemo(
    () => widthProbeData?.items?.map((campaign) => campaign.id),
    [widthProbeData?.items]
  );

  const metricsCampaignIds = useMemo(
    () =>
      listCoversWidthProbeDataset
        ? (campaignIds ?? [])
        : mergeCampaignIdsForMetricsBatch(campaignIds ?? [], widthProbeIds ?? []),
    [campaignIds, listCoversWidthProbeDataset, widthProbeIds]
  );

  const { data: metricsBatch, error: metricsError } = useResource(
    (signal) => fetchCampaignListMetricsBatch(metricsCampaignIds, statsQuery, signal),
    [metricsCampaignIds?.join(',') ?? '', refreshToken, statsQuery.from, statsQuery.to]
  );

  const filterTotalsQuery = useMemo(
    () => ({
      customer_id: query.customer_id,
      status: query.status,
      q: query.q,
      pacing_mode: query.pacing_mode,
      budget_min_micro: query.budget_min_micro,
      budget_max_micro: query.budget_max_micro,
      owner_user_id: query.owner_user_id,
      country: query.country,
    }),
    [
      query.budget_max_micro,
      query.budget_min_micro,
      query.country,
      query.customer_id,
      query.owner_user_id,
      query.pacing_mode,
      query.q,
      query.status,
    ]
  );

  const filterTotalsCapped = (data?.total ?? 0) > CAMPAIGN_LIST_FILTER_TOTALS_MAX;

  const { data: metricsTotalsResponse, error: filterTotalsError } = useResource(
    (signal) => {
      if (filterTotalsCapped) {
        return Promise.resolve(undefined);
      }
      return fetchCampaignListMetricsTotals(filterTotalsQuery, statsQuery, signal).catch((err) => {
        if (isCampaignListAuxEndpointUnavailable(err)) {
          return undefined;
        }
        throw err;
      });
    },
    [
      filterTotalsQuery.customer_id,
      filterTotalsQuery.status,
      filterTotalsQuery.q,
      filterTotalsQuery.pacing_mode,
      filterTotalsQuery.budget_min_micro,
      filterTotalsQuery.budget_max_micro,
      filterTotalsQuery.owner_user_id,
      filterTotalsQuery.country,
      filterTotalsCapped,
      refreshToken,
      statsQuery.from,
      statsQuery.to,
    ]
  );

  const filterTotals = useMemo(
    () => campaignListFilterTotalsFromApi(metricsTotalsResponse),
    [metricsTotalsResponse]
  );

  const metricsById = metricsBatch?.metricsById;
  const marginsById = metricsBatch?.marginsById;

  const columnWidthProbe = useMemo((): CampaignListColumnWidthProbe | undefined => {
    const probeItems = listCoversWidthProbeDataset ? data?.items : widthProbeData?.items;
    if (!probeItems?.length) {
      return undefined;
    }
    return {
      items: probeItems,
      metricsById: metricsById ?? {},
      marginsById: marginsById ?? {},
    };
  }, [data?.items, listCoversWidthProbeDataset, marginsById, metricsById, widthProbeData?.items]);

  const { data: listFacetsFromApi, fetching: listFacetsFetching } = useResource(
    (signal) =>
      fetchCampaignListFacets(customerId, signal).catch((err) => {
        if (isCampaignListAuxEndpointUnavailable(err)) {
          return undefined;
        }
        throw err;
      }),
    [customerId]
  );

  const { facets: listFacets, degraded: listFacetsDegraded } = useMemo(
    () => resolveCampaignListFacets(listFacetsFromApi, listFacetsFetching),
    [listFacetsFromApi, listFacetsFetching]
  );

  const countryOptions = useMemo(
    () => buildCampaignListCountryOptions(listFacets?.countries ?? [], query.country),
    [listFacets?.countries, query.country]
  );

  const ownerEmailById = useMemo(
    () => buildCampaignListOwnerEmailById(listFacets?.owners ?? []),
    [listFacets?.owners]
  );

  const ownerOptions = useMemo(
    () => buildCampaignListOwnerOptions(listFacets?.owners ?? [], appliedOwnerUserId),
    [appliedOwnerUserId, listFacets?.owners]
  );

  const statusTotals = data?.status_totals;
  const statusTotalsFetching = fetching && statusTotals == null;

  const { data: customersData, fetching: customersFetching } = useResource(
    (signal) => fetchCustomersComboboxCached(signal),
    []
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

  const templateCustomerId = createCustomerId.trim() || customerId;
  const shouldLoadTemplates = createSectionOpen && Boolean(templateCustomerId);

  const {
    data: templatesData,
    error: templatesError,
    fetching: templatesFetching,
  } = useResource(
    (signal) => {
      if (!shouldLoadTemplates || !templateCustomerId) {
        return Promise.resolve(undefined);
      }
      return listSelfServeTemplates(templateCustomerId, signal);
    },
    [templateCustomerId, templatesRefreshToken, shouldLoadTemplates]
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
    listFacetsFetching,
    listFacetsDegraded,
    filterTotals,
    filterTotalsCapped,
    filterTotalsError,
    metricsError,
    metricsStale: metricsBatch?.stale ?? false,
  };
}
