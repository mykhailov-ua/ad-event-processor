import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { toast } from 'sonner';

import {
  fetchCampaignListMargins,
  fetchCampaignListMetrics,
  fetchCampaignStatusTotals,
  listCampaigns,
} from '@/api/campaigns_api';
import { listCustomers } from '@/api/customers_api';
import { createSelfServeCampaign, listSelfServeTemplates } from '@/api/selfserve_api';
import type { CampaignListQuery } from '@/api/types';
import type { CustomerComboboxOption } from '@/components/system/customer_combobox';
import type { CampaignListMiddleColumnId } from '@/domains/campaigns/campaign_list_columns';
import {
  isCampaignClientSortField,
  isCampaignServerSortField,
  sortCampaignListItemsClient,
} from '@/domains/campaigns/campaign_list_sort';
import {
  CampaignsDirectory,
  type CampaignSortField,
  type CampaignStatusFilter,
  type SortOrder,
} from '@/domains/campaigns/campaigns_directory';
import { useResource } from '@/hooks/use_resource';
import { useSession } from '@/hooks/use_session';
import { DEFAULT_LIST_LIMIT, OPTIMAL_LIST_LIMIT_MAX, parseListLimit, parseListOffset } from '@/lib/list_query';

const CLIENT_SORT_COLUMNS = new Set<CampaignListMiddleColumnId>([
  'source',
  'flows',
  'clicks',
  'conversions',
  'cr',
  'revenue',
  'cost',
  'profit',
  'roi',
  'group',
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

function parseQ(raw: string | null): string {
  return raw ?? '';
}

function buildListQuery(
  params: URLSearchParams,
  defaultCustomerId: string | undefined,
): CampaignListQuery {
  const customerId = params.get('customer_id') ?? defaultCustomerId;
  const status = params.get('status');
  const q = params.get('q');
  const parsedSort = parseSort(params.get('sort'));
  const parsedOrder = parseOrder(params.get('order'));

  return {
    customer_id: customerId ?? undefined,
    status: status ?? undefined,
    q: q ?? undefined,
    limit: parseListLimit(params.get('limit')),
    offset: parseListOffset(params.get('offset')),
    sort: isCampaignServerSortField(parsedSort) ? parsedSort : 'name',
    order: isCampaignServerSortField(parsedSort) ? parsedOrder : 'asc',
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
  const appliedQ = parseQ(searchParams.get('q'));
  const appliedSort = parseSort(searchParams.get('sort'));
  const appliedOrder = parseOrder(searchParams.get('order'));
  const filtersActive = Boolean(
    appliedCustomerId ||
      appliedStatus ||
      appliedQ.trim() ||
      appliedSort !== 'name' ||
      appliedOrder !== 'asc',
  );

  const [draftCustomerId, setDraftCustomerId] = useState(appliedCustomerId);
  const [draftStatus, setDraftStatus] = useState<CampaignStatusFilter>(appliedStatus);
  const [draftQ, setDraftQ] = useState(appliedQ);

  useEffect(() => {
    setDraftCustomerId(appliedCustomerId);
    setDraftStatus(appliedStatus);
    setDraftQ(appliedQ);
  }, [appliedCustomerId, appliedQ, appliedStatus]);

  const refreshList = useCallback(() => {
    setRefreshToken((value) => value + 1);
  }, []);

  const { data, error, fetching } = useResource(
    (signal) => listCampaigns(query, signal),
    [
      query.customer_id,
      query.status,
      query.q,
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

  const { data: widthProbeMetrics } = useResource(
    (signal) => fetchCampaignListMetrics(widthProbeIds, signal),
    [widthProbeIds.join(','), refreshToken],
  );

  const { data: widthProbeMargins } = useResource(
    (signal) => fetchCampaignListMargins(widthProbeIds, signal),
    [widthProbeIds.join(','), refreshToken],
  );

  const columnWidthProbe = useMemo(() => {
    if (!widthProbeData?.items?.length) {
      return undefined;
    }
    return {
      items: widthProbeData.items,
      metricsById: widthProbeMetrics ?? {},
      marginsById: widthProbeMargins ?? {},
    };
  }, [widthProbeData?.items, widthProbeMargins, widthProbeMetrics]);

  const campaignIds = useMemo(
    () => (data?.items ?? []).map((campaign) => campaign.id),
    [data?.items],
  );

  const { data: metricsById } = useResource(
    (signal) => fetchCampaignListMetrics(campaignIds, signal),
    [campaignIds.join(','), refreshToken],
  );

  const { data: marginsById } = useResource(
    (signal) => fetchCampaignListMargins(campaignIds, signal),
    [campaignIds.join(','), refreshToken],
  );

  const listItems = useMemo(() => {
    const items = data?.items ?? [];
    if (isCampaignServerSortField(appliedSort)) {
      return items;
    }
    if (!isCampaignClientSortField(appliedSort)) {
      return items;
    }
    return sortCampaignListItemsClient(
      items,
      appliedSort,
      appliedOrder,
      metricsById ?? {},
      marginsById ?? {},
    );
  }, [appliedOrder, appliedSort, data?.items, marginsById, metricsById]);

  const { data: statusTotals, fetching: statusTotalsFetching } = useResource(
    (signal) =>
      fetchCampaignStatusTotals(
        { customer_id: query.customer_id, q: query.q },
        signal,
      ),
    [query.customer_id, query.q, refreshToken],
  );

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
    (patch: Partial<CampaignListQuery> & { sort?: CampaignSortField; order?: SortOrder }) => {
      const next = new URLSearchParams(searchParams);
      const merged: CampaignListQuery & { sort?: CampaignSortField; order?: SortOrder } = {
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

  const onSearchApply = useCallback(() => {
    updateQuery({
      q: draftQ.trim() || undefined,
      offset: 0,
    });
  }, [draftQ, updateQuery]);

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
      draftQ={draftQ}
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
      onDraftQChange={setDraftQ}
      onSearchApply={onSearchApply}
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
