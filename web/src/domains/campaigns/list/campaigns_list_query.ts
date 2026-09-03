import type { CampaignListQuery } from '@/api/types';
import type { CampaignListMiddleColumnId } from '@/domains/campaigns/list/campaign_list_columns';
import {
  campaignListStatsRangeFromDatetimeLocal,
  isCampaignListStatsRangeWithinLimit,
  resolveCampaignListStatsRange,
} from '@/domains/campaigns/list/campaign_list_date_range';
import {
  campaignListSortNeedsMetricWindow,
  campaignListSortToApi,
  sortFieldForCampaignColumn,
} from '@/domains/campaigns/list/campaign_list_sort';
import type {
  CampaignPacingFilter,
  CampaignSortField,
  CampaignStatusFilter,
  SortOrder,
} from '@/domains/campaigns/list/campaigns_list_types';
import { DEFAULT_LIST_LIMIT, parseListLimit, parseListOffset } from '@/lib/list_query';

export function parseCampaignListSort(raw: string | null): CampaignSortField {
  if (!raw?.trim()) {
    return 'name';
  }
  const trimmed = raw.trim();
  if (trimmed === 'id') {
    return 'updated_at';
  }
  if (
    trimmed === 'name' ||
    trimmed === 'updated_at' ||
    trimmed === 'spend' ||
    trimmed === 'budget_limit'
  ) {
    return trimmed;
  }
  if (sortFieldForCampaignColumn(trimmed)) {
    return trimmed as CampaignListMiddleColumnId;
  }
  return 'name';
}

export function parseCampaignListOrder(raw: string | null): SortOrder {
  return raw === 'desc' ? 'desc' : 'asc';
}

export function parseCampaignListStatus(raw: string | null): CampaignStatusFilter {
  if (raw === 'ACTIVE' || raw === 'PAUSED' || raw === 'ARCHIVED') {
    return raw;
  }
  return '';
}

export function parseCampaignListSearchQuery(raw: string | null): string {
  return raw ?? '';
}

export function parseCampaignListPacing(raw: string | null): CampaignPacingFilter {
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

export function buildCampaignListQuery(
  params: URLSearchParams,
  defaultCustomerId: string | undefined,
): CampaignListQuery {
  const customerId = params.get('customer_id') ?? defaultCustomerId;
  const status = params.get('status');
  const q = params.get('q');
  const pacingMode = params.get('pacing_mode');
  const parsedSort = parseCampaignListSort(params.get('sort'));
  const parsedOrder = parseCampaignListOrder(params.get('order'));
  const apiSort = campaignListSortToApi(parsedSort);
  const statsRange = resolveCampaignListStatsRange(
    params.get('stats_from'),
    params.get('stats_to'),
    params.get('stats_range'),
  );

  const query: CampaignListQuery = {
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
    sort: apiSort,
    order: parsedOrder,
  };

  if (campaignListSortNeedsMetricWindow(apiSort)) {
    query.from = statsRange.from;
    query.to = statsRange.to;
  }

  return query;
}

export type CampaignListQueryPatch = Partial<Omit<CampaignListQuery, 'sort'>> & {
  sort?: CampaignSortField;
  order?: SortOrder;
  stats_from?: string;
  stats_to?: string;
  owner_user_id?: string;
  country?: string;
};

export function applyCampaignListQueryPatch(
  searchParams: URLSearchParams,
  currentQuery: CampaignListQuery,
  patch: CampaignListQueryPatch,
): URLSearchParams {
  const next = new URLSearchParams(searchParams);
  const merged = {
    ...currentQuery,
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

  return next;
}

export function campaignListFiltersActive(
  searchParams: URLSearchParams,
  appliedQ: string,
  appliedSort: CampaignSortField,
  appliedOrder: SortOrder,
): boolean {
  return Boolean(
    searchParams.get('customer_id') ||
      searchParams.get('status') ||
      searchParams.get('pacing_mode') ||
      searchParams.get('owner_user_id') ||
      searchParams.get('country') ||
      searchParams.get('budget_min_micro') ||
      searchParams.get('budget_max_micro') ||
      appliedQ.trim() ||
      appliedSort !== 'name' ||
      appliedOrder !== 'asc' ||
      searchParams.has('stats_from') ||
      searchParams.has('stats_to'),
  );
}

export function validateCampaignListStatsDraft(
  from: string,
  to: string,
): { ok: true; from: string; to: string } | { ok: false; error: string } {
  if (!from.trim() || !to.trim()) {
    return { ok: false, error: 'empty' };
  }
  const range = campaignListStatsRangeFromDatetimeLocal(from, to);
  if (!range) {
    return { ok: false, error: 'Invalid stats period.' };
  }
  if (!isCampaignListStatsRangeWithinLimit(range)) {
    return { ok: false, error: 'Stats period cannot exceed 90 days.' };
  }
  return { ok: true, from: range.from, to: range.to };
}
