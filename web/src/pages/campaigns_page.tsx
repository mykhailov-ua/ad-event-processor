import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { toast } from 'sonner';

import {
  fetchCampaignListMetricsBatch,
  listCampaigns,
} from '@/api/campaigns_api';
import { listCustomers } from '@/api/customers_api';
import { createSelfServeCampaign, listSelfServeTemplates } from '@/api/selfserve_api';
import { listTeamMembers } from '@/api/team_api';
import type { CampaignListQuery } from '@/api/types';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';
import type { CampaignListMiddleColumnId } from '@/domains/campaigns/list/campaign_list_columns';
import {
  microQueryParamToUsdInput,
  usdInputToMicroQueryParam,
} from '@/domains/campaigns/list/campaign_list_format';
import { useTrackerHeaderSearchRegistration } from '@/lib/tracker_header_context';
import {
  campaignListStatsRangeFromDatetimeLocal,
  campaignStatsQueryForRange,
  defaultCampaignListStatsRange,
  isCampaignListStatsRangeWithinLimit,
  resolveCampaignListStatsRange,
} from '@/domains/campaigns/list/campaign_list_date_range';
import {
  isCampaignClientSortField,
  isCampaignServerSortField,
  sortCampaignListItemsClient,
} from '@/domains/campaigns/list/campaign_list_sort';
import {
  CampaignsDirectory,
  type CampaignPacingFilter,
  type CampaignSortField,
  type CampaignStatusFilter,
  type SortOrder,
} from '@/domains/campaigns/list/campaigns_directory';
import type { CampaignsListFilterOption } from '@/domains/campaigns/list/campaigns_list_filter_select';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';
import { DEFAULT_LIST_LIMIT, OPTIMAL_LIST_LIMIT_MAX, parseListLimit, parseListOffset } from '@/lib/list_query';
import { toDatetimeLocalValue } from '@/lib/datetime_range';

// Metric column ids sorted client-side on the current server page only (regime S).
// Server sort fields are name, updated_at, spend, budget_limit (see toServerSort).
const CLIENT_SORT_COLUMNS = new Set<CampaignListMiddleColumnId>([
  'status',
  'tags',
  'clicks',
  'impressions',
  'ctr',
  'unique_clicks',
  'lp_clicks',
  'lp_views',
  'group',
  'lp_ctr',
  'cr',
  'leads',
  'approved',
  'hold_leads',
  'rejected_leads',
  'approve_rate',
  'epc',
  'cpc',
  'cpa',
  'ecpa',
  'cpm',
  'blocks',
  'block_pct',
  'bots',
  'bot_pct',
  'revenue',
  'cost',
  'profit',
  'roi',
  'budget_pct',
  'flow',
  'owner',
  'countries',
]);

function parseSort(raw: string | null): CampaignSortField {
  if (raw === 'updated_at' || raw === 'spend' || raw === 'budget_limit') {
    return raw;
  }
  if (raw === 'name' || raw === 'id') {
    return raw === 'id' ? 'updated_at' : 'name';
  }
  if (raw && CLIENT_SORT_COLUMNS.has(raw as CampaignListMiddleColumnId)) {
    return raw as CampaignListMiddleColumnId;
  }
  return 'name';
}

function parseOrder(raw: string | null): SortOrder {
  return raw === 'desc' ? 'desc' : 'asc';
}

function parseStatus(raw: string | null): CampaignStatusFilter {
  if (raw === 'ACTIVE' || raw === 'PAUSED' || raw === 'ARCHIVED') {
    return raw;
  }
  return '';
}

function parseSearchQuery(raw: string | null): string {
  return raw ?? '';
}

function parsePacing(raw: string | null): CampaignPacingFilter {
  if (raw === 'EVEN' || raw === 'ASAP') {
    return raw;
  }
  return '';
}

function parseOptionalMicro(raw: string | null): number | undefined {
  if (!raw?.trim()) {
    return undefined;
  }
  const parsed = Number.parseInt(raw, 10);
  if (!Number.isFinite(parsed) || parsed < 0) {
    return undefined;
  }
  return parsed;
}

function toServerSort(field: CampaignSortField): NonNullable<CampaignListQuery['sort']> {
  if (field === 'updated_at' || field === 'spend' || field === 'budget_limit') {
    return field;
  }
  return 'name';
}

function buildListQuery(
  params: URLSearchParams,
  defaultCustomerId: string | undefined,
): CampaignListQuery {
  const customerId = params.get('customer_id') ?? defaultCustomerId;
  const status = params.get('status');
  const q = params.get('q');
  const pacingMode = params.get('pacing_mode');
  const parsedSort = parseSort(params.get('sort'));
  const parsedOrder = parseOrder(params.get('order'));
  const serverSort = toServerSort(parsedSort);

  return {
    customer_id: customerId ?? undefined,
    status: status ?? undefined,
    q: q ?? undefined,
    pacing_mode: pacingMode ?? undefined,
    budget_min_micro: parseOptionalMicro(params.get('budget_min_micro')),
    budget_max_micro: parseOptionalMicro(params.get('budget_max_micro')),
    owner_user_id: params.get('owner_user_id') ?? undefined,
    country: params.get('country') ?? undefined,
    limit: parseListLimit(params.get('limit')),
    offset: parseListOffset(params.get('offset')),
    sort: serverSort,
    order: serverSort === parsedSort ? parsedOrder : 'asc',
  };
}

export function CampaignsPage() {
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
    () => buildListQuery(searchParams, session?.default_customer_id),
    [searchParams, session?.default_customer_id],
  );

  const widthProbeQuery = useMemo(
    (): CampaignListQuery => ({
      customer_id: query.customer_id,
      status: query.status,
      q: query.q,
      limit: OPTIMAL_LIST_LIMIT_MAX,
      offset: 0,
      sort: 'name',
      order: 'asc',
    }),
    [query.customer_id, query.q, query.status],
  );
  const customerId = query.customer_id;
  const appliedCustomerId = searchParams.get('customer_id') ?? '';
  const appliedStatus = parseStatus(searchParams.get('status'));
  const appliedQ = parseSearchQuery(searchParams.get('q'));
  const appliedSort = parseSort(searchParams.get('sort'));
  const appliedOrder = parseOrder(searchParams.get('order'));
  const appliedPacing = parsePacing(searchParams.get('pacing_mode'));
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
  const [draftStatsFrom, setDraftStatsFrom] = useState(
    toDatetimeLocalValue(appliedStatsRange.from),
  );
  const [draftStatsTo, setDraftStatsTo] = useState(toDatetimeLocalValue(appliedStatsRange.to));
  const filtersActive = Boolean(
    appliedCustomerId ||
      appliedStatus ||
      appliedPacing ||
      appliedOwnerUserId ||
      appliedCountry ||
      appliedBudgetMinMicro ||
      appliedBudgetMaxMicro ||
      appliedQ.trim() ||
      appliedSort !== 'name' ||
      appliedOrder !== 'asc' ||
      searchParams.has('stats_from') ||
      searchParams.has('stats_to'),
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
      refreshToken,
    ],
  );

  const { data: widthProbeData } = useResource(
    (signal) => listCampaigns(widthProbeQuery, signal),
    [widthProbeQuery.customer_id, widthProbeQuery.q, widthProbeQuery.status, refreshToken],
  );

  const widthProbeIds = useMemo(
    () => (widthProbeData?.items ?? []).map((campaign) => campaign.id),
    [widthProbeData?.items],
  );

  const { data: widthProbeMetricsBatch } = useResource(
    (signal) => fetchCampaignListMetricsBatch(widthProbeIds, statsQuery, signal),
    [widthProbeIds.join(','), refreshToken, statsQuery.from, statsQuery.to],
  );

  const columnWidthProbe = useMemo(() => {
    if (!widthProbeData?.items?.length) {
      return undefined;
    }
    return {
      items: widthProbeData.items,
      metricsById: widthProbeMetricsBatch?.metricsById ?? {},
      marginsById: widthProbeMetricsBatch?.marginsById ?? {},
    };
  }, [widthProbeData?.items, widthProbeMetricsBatch]);

  const campaignIds = useMemo(
    () => (data?.items ?? []).map((campaign) => campaign.id),
    [data?.items],
  );

  const { data: metricsBatch } = useResource(
    (signal) => fetchCampaignListMetricsBatch(campaignIds, statsQuery, signal),
    [campaignIds.join(','), refreshToken, statsQuery.from, statsQuery.to],
  );

  const metricsById = metricsBatch?.metricsById;
  const marginsById = metricsBatch?.marginsById;

  const listItems = useMemo(() => {
    let items = data?.items ?? [];
    // Hybrid sort: API ordered the page; metric columns reorder rows in-place using batch metrics.
    if (isCampaignClientSortField(appliedSort) && !isCampaignServerSortField(appliedSort)) {
      items = sortCampaignListItemsClient(
        items,
        appliedSort,
        appliedOrder,
        metricsById ?? {},
        marginsById ?? {},
      );
    }
    return items;
  }, [
    appliedOrder,
    appliedSort,
    data?.items,
    marginsById,
    metricsById,
  ]);

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

  const templates = templatesData?.items ?? [];

  useEffect(() => {
    if (templates.length === 0) {
      setDraftTemplateId('');
      return;
    }
    if (!templates.some((template) => template.id === draftTemplateId)) {
      setDraftTemplateId(templates[0]?.id ?? '');
    }
  }, [draftTemplateId, templates]);

  const updateQuery = useCallback(
    (
      patch: Partial<Omit<CampaignListQuery, 'sort'>> & {
        sort?: CampaignSortField;
        order?: SortOrder;
        stats_from?: string;
        stats_to?: string;
        owner_user_id?: string;
        country?: string;
      },
    ) => {
      const next = new URLSearchParams(searchParams);
      const merged = {
        ...query,
        ...patch,
      };

      if (merged.customer_id) {
        next.set('customer_id', merged.customer_id);
      } else {
        next.delete('customer_id');
      }
      if (merged.status) {
        next.set('status', merged.status);
      } else {
        next.delete('status');
      }
      if (merged.q) {
        next.set('q', merged.q);
      } else {
        next.delete('q');
      }
      if (merged.pacing_mode) {
        next.set('pacing_mode', merged.pacing_mode);
      } else {
        next.delete('pacing_mode');
      }
      if (merged.budget_min_micro != null) {
        next.set('budget_min_micro', String(merged.budget_min_micro));
      } else {
        next.delete('budget_min_micro');
      }
      if (merged.budget_max_micro != null) {
        next.set('budget_max_micro', String(merged.budget_max_micro));
      } else {
        next.delete('budget_max_micro');
      }
      if (merged.owner_user_id) {
        next.set('owner_user_id', merged.owner_user_id);
      } else {
        next.delete('owner_user_id');
      }
      if (merged.country) {
        next.set('country', merged.country);
      } else {
        next.delete('country');
      }
      if ('stats_from' in patch || 'stats_to' in patch) {
        next.delete('stats_range');
        if (merged.stats_from?.trim()) {
          next.set('stats_from', merged.stats_from);
        } else {
          next.delete('stats_from');
        }
        if (merged.stats_to?.trim()) {
          next.set('stats_to', merged.stats_to);
        } else {
          next.delete('stats_to');
        }
      }
      next.set('limit', String(merged.limit ?? DEFAULT_LIST_LIMIT));
      next.set('offset', String(merged.offset ?? 0));
      next.set('sort', merged.sort ?? 'name');
      next.set('order', merged.order ?? 'asc');

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
    (customerId: string) => {
      setDraftCustomerId(customerId);
      updateQuery({
        customer_id: customerId.trim() || undefined,
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
      if (!from.trim() || !to.trim()) {
        setDraftStatsFrom(toDatetimeLocalValue(defaultStatsRange.from));
        setDraftStatsTo(toDatetimeLocalValue(defaultStatsRange.to));
        setActionError(undefined);
        updateQuery({ stats_from: undefined, stats_to: undefined, offset: 0 });
        return;
      }

      const range = campaignListStatsRangeFromDatetimeLocal(from, to);
      if (!range) {
        setActionError(new Error('Invalid stats period.'));
        return;
      }
      if (!isCampaignListStatsRangeWithinLimit(range)) {
        setActionError(new Error('Stats period cannot exceed 90 days.'));
        return;
      }

      setActionError(undefined);
      setDraftStatsFrom(from);
      setDraftStatsTo(to);
      updateQuery({ stats_from: range.from, stats_to: range.to, offset: 0 });
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
      updateQuery({
        sort: field,
        order: nextOrder,
        offset: 0,
      });
    },
    [appliedOrder, appliedSort, updateQuery],
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

  return (
    <CampaignsDirectory
      items={listItems}
      total={data?.total ?? 0}
      limit={data?.limit ?? query.limit ?? DEFAULT_LIST_LIMIT}
      offset={data?.offset ?? query.offset ?? 0}
      statusTotals={statusTotals}
      statusTotalsLoading={statusTotalsFetching && statusTotals == null}
      customerOptions={customerOptions}
      customersLoading={customersFetching && customersData == null}
      customerNameById={customerNameById}
      metricsById={metricsById ?? {}}
      marginsById={marginsById ?? {}}
      columnWidthProbe={columnWidthProbe}
      appliedCustomerId={appliedCustomerId}
      appliedStatus={appliedStatus}
      appliedSort={appliedSort}
      appliedOrder={appliedOrder}
      draftCustomerId={draftCustomerId}
      draftStatus={draftStatus}
      draftPacing={draftPacing}
      draftOwnerUserId={draftOwnerUserId}
      draftCountry={draftCountry}
      draftBudgetMinUsd={draftBudgetMinUsd}
      draftBudgetMaxUsd={draftBudgetMaxUsd}
      draftStatsFrom={draftStatsFrom}
      draftStatsTo={draftStatsTo}
      ownerOptions={ownerOptions}
      ownerEmailById={ownerEmailById}
      countryOptions={countryOptions}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
      filtersActive={filtersActive}
      customerId={customerId}
      createSectionOpen={createSectionOpen}
      onCreateSectionOpenChange={setCreateSectionOpen}
      templates={templates}
      templatesLoading={templatesFetching}
      templatesError={templatesError}
      draftTemplateId={draftTemplateId}
      draftCreateName={draftCreateName}
      draftBudgetLimitMicro={draftBudgetLimitMicro}
      creating={creating}
      actionError={actionError}
      onDraftCustomerIdChange={onDraftCustomerIdChange}
      onDraftStatusChange={onDraftStatusChange}
      onDraftPacingChange={onDraftPacingChange}
      onDraftOwnerUserIdChange={onDraftOwnerUserIdChange}
      onDraftCountryChange={onDraftCountryChange}
      onDraftBudgetMinUsdChange={setDraftBudgetMinUsd}
      onDraftBudgetMaxUsdChange={setDraftBudgetMaxUsd}
      onBudgetFiltersApply={onBudgetFiltersApply}
      onStatsRangeChange={onStatsRangeChange}
      onRefreshList={refreshList}
      onColumnSort={onColumnSort}
      onPageChange={onPageChange}
      onPageSizeChange={onPageSizeChange}
      onDraftTemplateIdChange={setDraftTemplateId}
      onDraftCreateNameChange={setDraftCreateName}
      onDraftBudgetLimitMicroChange={setDraftBudgetLimitMicro}
      onLoadTemplates={onLoadTemplates}
      onCreateCampaign={onCreateCampaign}
    />
  );
}
