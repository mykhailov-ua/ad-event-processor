import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { toast } from 'sonner';

import { createSelfServeCampaign } from '@/api/selfserve_api';
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
import {
  campaignListSortNeedsMetricWindow,
  campaignListSortToApi,
} from '@/domains/campaigns/list/campaign_list_sort';
import type { CampaignsDirectoryProps } from '@/domains/campaigns/list/campaigns_directory_types';
import type {
  CampaignPacingFilter,
  CampaignSortField,
  CampaignStatusFilter,
} from '@/domains/campaigns/list/campaigns_list_types';
import { useCampaignsPageList } from '@/pages/use_campaigns_page_list';
import { useSession } from '@/hooks/use_session';
import { useTrackerHeaderSearchRegistration } from '@/lib/tracker_header_context';
import { userErrorMessage } from '@/lib/admin_error';
import { DEFAULT_LIST_LIMIT } from '@/lib/list_query';
import { toDatetimeLocalValue } from '@/lib/datetime_range';

export function useCampaignsPage(): CampaignsDirectoryProps {
  const [searchParams, setSearchParams] = useSearchParams();
  const { session } = useSession();
  const [refreshToken, setRefreshToken] = useState(0);
  const [createSectionOpen, setCreateSectionOpen] = useState(false);
  const [templatesRefreshToken, setTemplatesRefreshToken] = useState(0);
  const [draftTemplateId, setDraftTemplateId] = useState('');
  const [draftCreateName, setDraftCreateName] = useState('');
  const [draftBudgetLimitMicro, setDraftBudgetLimitMicro] = useState('');
  const [creating, setCreating] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();

  const query = useMemo(
    () => buildCampaignListQuery(searchParams, session?.default_customer_id),
    [searchParams, session?.default_customer_id],
  );
  const exportFilterQuery = useMemo(
    () => campaignListFilterQueryFromListQuery(query),
    [query],
  );

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
        searchParams.get('stats_range'),
      ),
    [searchParams],
  );
  const statsQuery = useMemo(
    () => campaignStatsQueryForRange(appliedStatsRange),
    [appliedStatsRange],
  );
  const listScopeKey = useMemo(
    () =>
      campaignListSelectionScopeKey({
        query,
        statsFrom: statsQuery.from,
        statsTo: statsQuery.to,
      }),
    [query, statsQuery.from, statsQuery.to],
  );
  const [draftStatsFrom, setDraftStatsFrom] = useState(
    toDatetimeLocalValue(appliedStatsRange.from),
  );
  const [draftStatsTo, setDraftStatsTo] = useState(toDatetimeLocalValue(appliedStatsRange.to));
  const filtersActive = campaignListFiltersActive(
    searchParams,
    appliedQ,
    appliedSort,
    appliedOrder,
  );

  const [draftCustomerId, setDraftCustomerId] = useState(appliedCustomerId);
  const [draftStatus, setDraftStatus] = useState<CampaignStatusFilter>(appliedStatus);
  const [draftPacing, setDraftPacing] = useState<CampaignPacingFilter>(appliedPacing);
  const [draftQ, setDraftQ] = useState(appliedQ);
  const [draftOwnerUserId, setDraftOwnerUserId] = useState(appliedOwnerUserId);
  const [draftCountry, setDraftCountry] = useState(appliedCountry);
  const [draftBudgetMinUsd, setDraftBudgetMinUsd] = useState(
    microQueryParamToUsdInput(appliedBudgetMinMicro),
  );
  const [draftBudgetMaxUsd, setDraftBudgetMaxUsd] = useState(
    microQueryParamToUsdInput(appliedBudgetMaxMicro),
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

  const refreshList = useCallback(() => {
    setRefreshToken((value) => value + 1);
  }, []);

  const {
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
    statusTotalsLoading,
    customerOptions,
    customersLoading,
    customerNameById,
    templates,
    templatesError,
    templatesLoading,
    listFacetsFetching,
    filterTotals,
    filterTotalsCapped,
    filterTotalsError,
    metricsError,
    metricsStale,
  } = useCampaignsPageList({
    query,
    statsQuery,
    refreshToken,
    customerId,
    appliedOwnerUserId,
    createSectionOpen,
    templatesRefreshToken,
  });

  useEffect(() => {
    if (templates.length === 0) {
      setDraftTemplateId('');
      return;
    }
    if (!templates.some((template) => template.id === draftTemplateId)) {
      setDraftTemplateId(templates[0]?.id ?? '');
    }
  }, [draftTemplateId, templates]);

  useEffect(() => {
    if (error && data != null) {
      toast.error(userErrorMessage(error, 'Could not refresh campaign list'));
    }
  }, [data, error]);

  useEffect(() => {
    if (metricsError) {
      toast.error(userErrorMessage(metricsError, 'Could not load campaign metrics'));
    }
  }, [metricsError]);

  useEffect(() => {
    if (filterTotalsError) {
      toast.error(userErrorMessage(filterTotalsError, 'Could not load filter totals'));
    }
  }, [filterTotalsError]);

  const updateQuery = useCallback(
    (patch: CampaignListQueryPatch) => {
      const next = applyCampaignListQueryPatch(searchParams, query, patch);
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

  const onPageSizeChange = useCallback(
    (size: number) => {
      updateQuery({ limit: size, offset: 0 });
    },
    [updateQuery],
  );

  const onDraftCustomerIdChange = useCallback(
    (nextCustomerId: string) => {
      setDraftCustomerId(nextCustomerId);
      updateQuery({
        customer_id: nextCustomerId.trim() || undefined,
        offset: 0,
      });
    },
    [updateQuery],
  );

  const onDraftStatusChange = useCallback(
    (status: CampaignStatusFilter) => {
      setDraftStatus(status);
      updateQuery({
        status: status || undefined,
        offset: 0,
      });
    },
    [updateQuery],
  );

  const onDraftPacingChange = useCallback(
    (pacing: CampaignPacingFilter) => {
      setDraftPacing(pacing);
      updateQuery({
        pacing_mode: pacing || undefined,
        offset: 0,
      });
    },
    [updateQuery],
  );

  const onDraftOwnerUserIdChange = useCallback(
    (ownerUserId: string) => {
      setDraftOwnerUserId(ownerUserId);
      updateQuery({
        owner_user_id: ownerUserId || undefined,
        offset: 0,
      });
    },
    [updateQuery],
  );

  const onDraftCountryChange = useCallback(
    (country: string) => {
      setDraftCountry(country);
      updateQuery({
        country: country || undefined,
        offset: 0,
      });
    },
    [updateQuery],
  );

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
    [defaultStatsRange.from, defaultStatsRange.to, updateQuery],
  );

  const onBudgetFiltersApply = useCallback(() => {
    updateQuery({
      budget_min_micro: usdInputToMicroQueryParam(draftBudgetMinUsd),
      budget_max_micro: usdInputToMicroQueryParam(draftBudgetMaxUsd),
      offset: 0,
    });
  }, [draftBudgetMaxUsd, draftBudgetMinUsd, updateQuery]);

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
    [draftQ, fetching, onSearchApply],
  );
  useTrackerHeaderSearchRegistration(headerSearchConfig);

  const onColumnSort = useCallback(
    (field: CampaignSortField) => {
      const nextOrder =
        appliedSort === field && appliedOrder === 'asc' ? 'desc' : 'asc';
      const patch: CampaignListQueryPatch = {
        sort: field,
        order: nextOrder,
        offset: 0,
      };
      if (campaignListSortNeedsMetricWindow(campaignListSortToApi(field))) {
        patch.stats_from = appliedStatsRange.from;
        patch.stats_to = appliedStatsRange.to;
      }
      updateQuery(patch);
    },
    [appliedOrder, appliedSort, appliedStatsRange.from, appliedStatsRange.to, updateQuery],
  );

  const onLoadTemplates = useCallback(() => {
    setTemplatesRefreshToken((value) => value + 1);
  }, []);

  const onCreateCampaign = useCallback(async () => {
    if (!customerId || !draftTemplateId) {
      return;
    }

    const budgetRaw = draftBudgetLimitMicro.trim();
    let budgetLimitMicro: number | undefined;
    if (budgetRaw) {
      const parsed = Number.parseInt(budgetRaw, 10);
      if (!Number.isFinite(parsed) || parsed <= 0) {
        setActionError(new Error('Budget must be a positive integer (micro units)'));
        return;
      }
      budgetLimitMicro = parsed;
    }

    setCreating(true);
    setActionError(undefined);
    try {
      await createSelfServeCampaign({
        customer_id: customerId,
        template_id: draftTemplateId,
        name: draftCreateName.trim() || undefined,
        budget_limit_micro: budgetLimitMicro,
      });
      setDraftCreateName('');
      setDraftBudgetLimitMicro('');
      setCreateSectionOpen(false);
      toast.success('Campaign created');
      refreshList();
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [
    customerId,
    draftBudgetLimitMicro,
    draftCreateName,
    draftTemplateId,
    refreshList,
  ]);

  return {
    items: data?.items ?? [],
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
    columnWidthProbe,
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
    filterTotals,
    filterTotalsCapped,
    filteredTotal: data?.total ?? 0,
    metricsStale,
    listScopeKey,
    statsQuery,
    exportFilterQuery,
    fetching,
    error,
    hasSnapshot: data != null,
    filtersActive,
    customerId,
    createSectionOpen,
    onCreateSectionOpenChange: setCreateSectionOpen,
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
    onDraftBudgetMinUsdChange: setDraftBudgetMinUsd,
    onDraftBudgetMaxUsdChange: setDraftBudgetMaxUsd,
    onBudgetFiltersApply,
    onStatsRangeChange,
    onRefreshList: refreshList,
    onColumnSort,
    onPageChange,
    onPageSizeChange,
    onDraftTemplateIdChange: setDraftTemplateId,
    onDraftCreateNameChange: setDraftCreateName,
    onDraftBudgetLimitMicroChange: setDraftBudgetLimitMicro,
    onLoadTemplates,
    onCreateCampaign,
  };
}
