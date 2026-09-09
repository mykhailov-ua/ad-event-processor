// campaigns page owner: URL searchParams are applied filters; draft* until commit; delegates list fetch to useCampaignsPageList.
import { useCallback, useEffect, useMemo, useState } from 'react';

import {
  microQueryParamToUsdInput,
  usdInputToMicroQueryParam,
} from '@/domains/campaigns/list/campaign_list_format';
import {
  campaignStatsQueryForRange,
  defaultCampaignListStatsRange,
  resolveCampaignListStatsRange,
} from '@/domains/campaigns/list/campaign_list_date_range';
import {
  applyCampaignListQueryPatch,
  buildCampaignListQuery,
  campaignListFilterQueryFromListQuery,
  campaignListFiltersActive,
  parseCampaignListOrder,
  parseCampaignListPacing,
  parseCampaignListSearchQuery,
  parseCampaignListSort,
  parseCampaignListStatus,
  validateCampaignListStatsDraft,
  type CampaignListQueryPatch,
} from '@/domains/campaigns/list/campaigns_list_query';
import { campaignListSelectionScopeKey } from '@/domains/campaigns/list/campaign_list_selection_scope';
import type { CampaignsDirectoryProps } from '@/domains/campaigns/list/campaigns_directory_types';
import type {
  CampaignPacingFilter,
  CampaignSortField,
  CampaignStatusFilter,
} from '@/domains/campaigns/list/campaigns_list_types';
import { useCampaignsPageList } from '@/domains/campaigns/list/use_campaigns_page_list';
import { useCampaignsPageMutations } from '@/domains/campaigns/list/use_campaigns_page_mutations';
import { useSession } from '@/hooks/use_session';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { useTrackerHeaderSearchRegistration } from '@/lib/tracker_header_context';
import { DEFAULT_LIST_LIMIT } from '@/lib/list_query';
import { toDatetimeLocalValue } from '@/lib/datetime_range';

export function useCampaignsPage(): CampaignsDirectoryProps {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const { session } = useSession();
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const [createSectionOpen, setCreateSectionOpen] = useState(false);
  const [templatesRefreshToken, setTemplatesRefreshToken] = useState(0);
  const [draftTemplateId, setDraftTemplateId] = useState('');
  const [draftCreateCustomerId, setDraftCreateCustomerId] = useState('');
  const [draftCreateName, setDraftCreateName] = useState('');
  const [draftBudgetLimitMicro, setDraftBudgetLimitMicro] = useState('');
  const [creating, setCreating] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();

  const query = useMemo(
    () => buildCampaignListQuery(searchParams, session?.default_customer_id),
    [searchParams, session?.default_customer_id]
  );
  const exportFilterQuery = useMemo(() => campaignListFilterQueryFromListQuery(query), [query]);

  const customerId = query.customer_id;
  const appliedCustomerId = searchParams.get('customer_id') ?? '';
  const appliedStatus = parseCampaignListStatus(searchParams.get('status'));
  const appliedQ = parseCampaignListSearchQuery(searchParams.get('q'));
  const appliedSort = parseCampaignListSort(searchParams.get('sort'));
  const appliedOrder = parseCampaignListOrder(searchParams.get('order'));
  const appliedPacing = parseCampaignListPacing(searchParams.get('pacing_mode'));
  const appliedOwnerUserId = searchParams.get('owner_user_id') ?? '';
  const appliedCountry = searchParams.get('country') ?? '';
  const appliedBudgetMinMicro = searchParams.get('budget_min_micro') ?? '';
  const appliedBudgetMaxMicro = searchParams.get('budget_max_micro') ?? '';
  const defaultStatsRange = useMemo(() => defaultCampaignListStatsRange(), []);
  const appliedStatsRange = useMemo(
    () =>
      resolveCampaignListStatsRange(
        searchParams.get('stats_from'),
        searchParams.get('stats_to'),
        searchParams.get('stats_range')
      ),
    [searchParams]
  );
  const statsQuery = useMemo(
    () => campaignStatsQueryForRange(appliedStatsRange),
    [appliedStatsRange]
  );
  const listScopeKey = useMemo(
    () =>
      campaignListSelectionScopeKey({
        query,
        statsFrom: statsQuery.from,
        statsTo: statsQuery.to,
      }),
    [query, statsQuery.from, statsQuery.to]
  );
  const [draftStatsFrom, setDraftStatsFrom] = useState(
    toDatetimeLocalValue(appliedStatsRange.from)
  );
  const [draftStatsTo, setDraftStatsTo] = useState(toDatetimeLocalValue(appliedStatsRange.to));
  const filtersActive = campaignListFiltersActive(
    searchParams,
    appliedQ,
    appliedSort,
    appliedOrder
  );

  const [draftCustomerId, setDraftCustomerId] = useState(appliedCustomerId);
  const [draftStatus, setDraftStatus] = useState<CampaignStatusFilter>(appliedStatus);
  const [draftPacing, setDraftPacing] = useState<CampaignPacingFilter>(appliedPacing);
  const [draftQ, setDraftQ] = useState(appliedQ);
  const [draftOwnerUserId, setDraftOwnerUserId] = useState(appliedOwnerUserId);
  const [draftCountry, setDraftCountry] = useState(appliedCountry);
  const [draftBudgetMinUsd, setDraftBudgetMinUsd] = useState(
    microQueryParamToUsdInput(appliedBudgetMinMicro)
  );
  const [draftBudgetMaxUsd, setDraftBudgetMaxUsd] = useState(
    microQueryParamToUsdInput(appliedBudgetMaxMicro)
  );

  useEffect(() => {
    setDraftCustomerId(appliedCustomerId);
    setDraftStatus(appliedStatus);
    setDraftPacing(appliedPacing);
    setDraftQ(appliedQ);
    setDraftOwnerUserId(appliedOwnerUserId);
    setDraftCountry(appliedCountry);
    setDraftBudgetMinUsd(microQueryParamToUsdInput(appliedBudgetMinMicro));
    setDraftBudgetMaxUsd(microQueryParamToUsdInput(appliedBudgetMaxMicro));
    setDraftStatsFrom(toDatetimeLocalValue(appliedStatsRange.from));
    setDraftStatsTo(toDatetimeLocalValue(appliedStatsRange.to));
  }, [
    appliedBudgetMaxMicro,
    appliedBudgetMinMicro,
    appliedCountry,
    appliedCustomerId,
    appliedOwnerUserId,
    appliedPacing,
    appliedQ,
    appliedStatsRange.from,
    appliedStatsRange.to,
    appliedStatus,
  ]);

  const {
    data,
    error,
    fetching,
    listRevalidating,
    metricsById,
    marginsById,
    countryOptions,
    ownerOptions,
    ownerEmailById,
    statusTotals,
    statusTotalsLoading,
    customerOptions,
    customersLoading,
    customerNameById,
    templates,
    templatesError,
    templatesLoading,
    listFacetsFetching,
    listFacetsDegraded,
    filterTotals,
    filterTotalsCapped,
    filterTotalsError,
    metricsError,
    metricsStale,
    listLastUpdatedAt,
  } = useCampaignsPageList({
    query,
    statsQuery,
    refreshToken,
    customerId,
    createCustomerId: draftCreateCustomerId,
    appliedOwnerUserId,
    createSectionOpen,
    templatesRefreshToken,
  });

  const refreshList = useCoalescedBumpRefresh(bumpRefresh, fetching || listRevalidating);

  const onCreateSectionOpenChange = useCallback(
    (open: boolean) => {
      if (open) {
        setDraftCreateCustomerId(appliedCustomerId || customerId || '');
      }
      setCreateSectionOpen(open);
    },
    [appliedCustomerId, customerId]
  );

  const { onLoadTemplates, onCreateCampaign } = useCampaignsPageMutations({
    customerId,
    appliedCustomerId,
    templates,
    templatesFetching: templatesLoading,
    draftTemplateId,
    setDraftTemplateId,
    draftCreateCustomerId,
    draftCreateName,
    setDraftCreateName,
    draftBudgetLimitMicro,
    setDraftBudgetLimitMicro,
    setCreating,
    setActionError,
    setTemplatesRefreshToken,
    setCreateSectionOpen,
    createSectionOpen,
    refreshList,
  });

  useEffect(() => {
    if (createSectionOpen) {
      return;
    }
    setActionError(undefined);
    setDraftCreateName('');
    setDraftBudgetLimitMicro('');
  }, [createSectionOpen]);

  const updateQuery = useCallback(
    (patch: CampaignListQueryPatch) => {
      const next = applyCampaignListQueryPatch(searchParams, query, patch);
      replaceSearchParams(next);
    },
    [query, replaceSearchParams, searchParams]
  );

  const onPageChange = useCallback(
    (nextOffset: number) => {
      updateQuery({ offset: Math.max(0, nextOffset) });
    },
    [updateQuery]
  );

  const onPageSizeChange = useCallback(
    (size: number) => {
      updateQuery({ limit: size, offset: 0 });
    },
    [updateQuery]
  );

  const onDraftCustomerIdChange = useCallback((nextCustomerId: string) => {
    setDraftCustomerId(nextCustomerId);
  }, []);

  const onDraftStatusChange = useCallback((status: CampaignStatusFilter) => {
    setDraftStatus(status);
  }, []);

  const onDraftPacingChange = useCallback((pacing: CampaignPacingFilter) => {
    setDraftPacing(pacing);
  }, []);

  const onDraftOwnerUserIdChange = useCallback((ownerUserId: string) => {
    setDraftOwnerUserId(ownerUserId);
  }, []);

  const onDraftCountryChange = useCallback((country: string) => {
    setDraftCountry(country);
  }, []);

  const onDirectoryFiltersApply = useCallback(() => {
    updateQuery({
      customer_id: draftCustomerId.trim() || undefined,
      status: draftStatus || undefined,
      pacing_mode: draftPacing || undefined,
      owner_user_id: draftOwnerUserId || undefined,
      country: draftCountry || undefined,
      budget_min_micro: usdInputToMicroQueryParam(draftBudgetMinUsd),
      budget_max_micro: usdInputToMicroQueryParam(draftBudgetMaxUsd),
      offset: 0,
    });
  }, [
    draftBudgetMaxUsd,
    draftBudgetMinUsd,
    draftCountry,
    draftCustomerId,
    draftOwnerUserId,
    draftPacing,
    draftStatus,
    updateQuery,
  ]);

  const onStatsRangeChange = useCallback(
    (from: string, to: string) => {
      const validated = validateCampaignListStatsDraft(from, to);
      if (!validated.ok) {
        if (validated.error === 'empty') {
          setDraftStatsFrom(toDatetimeLocalValue(defaultStatsRange.from));
          setDraftStatsTo(toDatetimeLocalValue(defaultStatsRange.to));
          setActionError(undefined);
          updateQuery({ stats_from: undefined, stats_to: undefined, offset: 0 });
          return;
        }
        setActionError(new Error(validated.error));
        return;
      }

      setActionError(undefined);
      setDraftStatsFrom(from);
      setDraftStatsTo(to);
      updateQuery({ stats_from: validated.from, stats_to: validated.to, offset: 0 });
    },
    [defaultStatsRange.from, defaultStatsRange.to, updateQuery]
  );

  const onSearchApply = useCallback(() => {
    updateQuery({
      q: draftQ.trim() || undefined,
      offset: 0,
    });
  }, [draftQ, updateQuery]);

  const headerSearchConfig = useMemo(
    () => ({
      value: draftQ,
      onChange: setDraftQ,
      onApply: onSearchApply,
      disabled: fetching,
      placeholder: 'id, name, url',
    }),
    [draftQ, fetching, onSearchApply]
  );
  useTrackerHeaderSearchRegistration(headerSearchConfig);

  return {
    items: data?.items,
    total: data?.total ?? 0,
    limit: data?.limit ?? query.limit ?? DEFAULT_LIST_LIMIT,
    offset: data?.offset ?? query.offset ?? 0,
    statusTotals,
    statusTotalsLoading,
    customerOptions,
    customersLoading,
    customerNameById,
    metricsById: metricsById ?? {},
    marginsById: marginsById ?? {},
    appliedCustomerId,
    appliedStatus,
    appliedSort,
    appliedOrder,
    draftCustomerId,
    draftStatus,
    draftPacing,
    draftOwnerUserId,
    draftCountry,
    draftBudgetMinUsd,
    draftBudgetMaxUsd,
    draftStatsFrom,
    draftStatsTo,
    ownerOptions,
    ownerEmailById,
    countryOptions,
    listFacetsFetching,
    listFacetsDegraded,
    filterTotals,
    filterTotalsCapped,
    filteredTotal: data?.total ?? 0,
    metricsStale,
    listLastUpdatedAt,
    listScopeKey,
    statsQuery,
    exportFilterQuery,
    fetching,
    listRevalidating: listRevalidating || listQueryPending,
    error,
    metricsError,
    filterTotalsError,
    hasSnapshot: data != null,
    filtersActive,
    customerId,
    createCustomerId: draftCreateCustomerId,
    createSectionOpen,
    onCreateSectionOpenChange,
    templates,
    templatesLoading,
    templatesError,
    draftTemplateId,
    draftCreateName,
    draftBudgetLimitMicro,
    creating,
    actionError,
    onDraftCustomerIdChange,
    onDraftStatusChange,
    onDraftPacingChange,
    onDraftOwnerUserIdChange,
    onDraftCountryChange,
    onDirectoryFiltersApply,
    onDraftBudgetMinUsdChange: setDraftBudgetMinUsd,
    onDraftBudgetMaxUsdChange: setDraftBudgetMaxUsd,
    onStatsRangeChange,
    onRefreshList: refreshList,
    onPageChange,
    onPageSizeChange,
    onDraftTemplateIdChange: setDraftTemplateId,
    onDraftCreateCustomerIdChange: setDraftCreateCustomerId,
    onDraftCreateNameChange: setDraftCreateName,
    onDraftBudgetLimitMicroChange: setDraftBudgetLimitMicro,
    onLoadTemplates,
    onCreateCampaign,
  };
}
