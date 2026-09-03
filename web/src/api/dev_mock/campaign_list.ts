import type { Campaign } from '@/api/types';
import { campaignBudgetUsedPercent } from '@/lib/campaign_budget_used.ts';

import {
  buildDevMockCampaignMetrics,
  compareDevMockCampaignMetricSort,
  devMockCampaignMetricSortNeedsWindow,
  enrichDevMockCampaignMetricsDerived,
} from './campaign_metrics.ts';

type MockListResult = {
  status: number;
  body?: unknown;
};

function json(status: number, body: unknown): MockListResult {
  return { status, body };
}

function filtersAppliedFromQuery(url: URL, keys: readonly string[]): Record<string, string> {
  const applied: Record<string, string> = {};
  for (const key of keys) {
    const value = url.searchParams.get(key);
    if (value != null && value !== '') {
      applied[key] = value;
    }
  }
  return applied;
}

function countDevMockStatusTotals(
  rows: Array<{ status?: string }>,
): { active: number; paused: number; archived: number; total: number } {
  let active = 0;
  let paused = 0;
  let archived = 0;
  for (const row of rows) {
    switch (row.status) {
      case 'ACTIVE':
        active++;
        break;
      case 'PAUSED':
        paused++;
        break;
      case 'ARCHIVED':
        archived++;
        break;
      default:
        break;
    }
  }
  return { active, paused, archived, total: rows.length };
}

function campaignMoneyMicro(raw?: string | null): number {
  const value = Number.parseFloat(raw ?? '');
  if (!Number.isFinite(value)) {
    return 0;
  }
  return Math.round(value * 1_000_000);
}

function compareCampaignRows(
  left: Campaign,
  right: Campaign,
  sort: string,
  metricsById: Map<string, ReturnType<typeof buildDevMockCampaignMetrics>>,
): number {
  switch (sort) {
    case 'name':
      return left.name.localeCompare(right.name);
    case 'updated_at':
      return left.updated_at.localeCompare(right.updated_at);
    case 'spend':
      return campaignMoneyMicro(left.current_spend) - campaignMoneyMicro(right.current_spend);
    case 'budget_limit':
      return campaignMoneyMicro(left.budget_limit) - campaignMoneyMicro(right.budget_limit);
    case 'status':
      return (left.status ?? '').localeCompare(right.status ?? '');
    case 'budget_pct': {
      const leftLimit = campaignMoneyMicro(left.budget_limit);
      const rightLimit = campaignMoneyMicro(right.budget_limit);
      const leftPct = leftLimit > 0 ? campaignMoneyMicro(left.current_spend) / leftLimit : 0;
      const rightPct = rightLimit > 0 ? campaignMoneyMicro(right.current_spend) / rightLimit : 0;
      return leftPct - rightPct;
    }
    default: {
      const leftMetrics = metricsById.get(left.id);
      const rightMetrics = metricsById.get(right.id);
      if (!leftMetrics || !rightMetrics) {
        return left.name.localeCompare(right.name);
      }
      return compareDevMockCampaignMetricSort(sort, leftMetrics, rightMetrics);
    }
  }
}

export function devMockListCampaigns(url: URL, campaigns: Campaign[]): MockListResult {
  const customerId = url.searchParams.get('customer_id') ?? '';
  const status = url.searchParams.get('status') ?? '';
  const q = (url.searchParams.get('q') ?? '').trim().toLowerCase();
  const pacing = url.searchParams.get('pacing_mode') ?? '';
  const ownerUserId = url.searchParams.get('owner_user_id') ?? '';
  const country = url.searchParams.get('country') ?? '';
  const budgetMin = url.searchParams.get('budget_min_micro');
  const budgetMax = url.searchParams.get('budget_max_micro');
  const sort = url.searchParams.get('sort') ?? 'name';
  const order = url.searchParams.get('order') === 'desc' ? 'desc' : 'asc';
  const from = url.searchParams.get('from') ?? '';
  const to = url.searchParams.get('to') ?? '';
  const limit = Math.min(200, Math.max(1, Number.parseInt(url.searchParams.get('limit') ?? '50', 10) || 50));
  const offset = Math.max(0, Number.parseInt(url.searchParams.get('offset') ?? '0', 10) || 0);

  if (devMockCampaignMetricSortNeedsWindow(sort) && (!from.trim() || !to.trim())) {
    return json(400, {
      error: {
        code: 'INVALID_QUERY',
        message: 'from and to required for metric sort',
      },
    });
  }

  let rows = [...campaigns];

  if (customerId) {
    rows = rows.filter((row) => row.customer_id === customerId);
  }
  if (pacing) {
    rows = rows.filter((row) => row.pacing_mode === pacing);
  }
  if (ownerUserId) {
    rows = rows.filter((row) => row.owner_user_id === ownerUserId);
  }
  if (country) {
    rows = rows.filter((row) => row.target_countries?.includes(country));
  }
  if (q) {
    rows = rows.filter((row) => row.name.toLowerCase().includes(q) || row.id.includes(q));
  }
  if (budgetMin) {
    const min = Number.parseInt(budgetMin, 10);
    if (Number.isFinite(min)) {
      rows = rows.filter((row) => campaignMoneyMicro(row.budget_limit) >= min);
    }
  }
  if (budgetMax) {
    const max = Number.parseInt(budgetMax, 10);
    if (Number.isFinite(max)) {
      rows = rows.filter((row) => campaignMoneyMicro(row.budget_limit) <= max);
    }
  }

  const statusTotals = countDevMockStatusTotals(rows);

  if (status) {
    rows = rows.filter((row) => row.status === status);
  }

  const metricsById = new Map<string, ReturnType<typeof buildDevMockCampaignMetrics>>();
  if (devMockCampaignMetricSortNeedsWindow(sort)) {
    const rangeFrom =
      from.trim() ||
      new Date(Date.now() - 7 * 86_400_000).toISOString();
    const rangeTo = to.trim() || new Date().toISOString();
    rows.forEach((row, index) => {
      metricsById.set(row.id, buildDevMockCampaignMetrics(row.id, index + 1, rangeFrom, rangeTo));
    });
  }

  rows.sort((left, right) => {
    let cmp = compareCampaignRows(left, right, sort, metricsById);
    if (cmp === 0) {
      cmp = left.name.localeCompare(right.name);
    }
    return order === 'desc' ? -cmp : cmp;
  });

  const total = rows.length;
  const items = rows.slice(offset, offset + limit).map((row) => {
    const budgetUsedPct = campaignBudgetUsedPercent(row.budget_limit, row.current_spend);
    return budgetUsedPct == null ? row : { ...row, budget_used_pct: budgetUsedPct };
  });
  const filtersApplied = filtersAppliedFromQuery(url, [
    'customer_id',
    'status',
    'q',
    'pacing_mode',
    'owner_user_id',
    'country',
    'budget_min_micro',
    'budget_max_micro',
    'from',
    'to',
  ]);

  return json(200, {
    items,
    total,
    limit,
    offset,
    sort: { field: sort, order },
    status_totals: statusTotals,
    ...(Object.keys(filtersApplied).length > 0 ? { filters_applied: filtersApplied } : {}),
  });
}

export function devMockListCampaignFacets(
  url: URL,
  campaigns: Campaign[],
  userEmailById: Readonly<Record<string, string>>,
): MockListResult {
  const customerId = url.searchParams.get('customer_id') ?? '';
  let rows = [...campaigns];
  if (customerId) {
    rows = rows.filter((row) => row.customer_id === customerId);
  }

  const countries = new Set<string>();
  const owners = new Map<string, { user_id: string; email?: string }>();

  for (const row of rows) {
    for (const code of row.target_countries ?? []) {
      const trimmed = code?.trim();
      if (trimmed) {
        countries.add(trimmed);
      }
    }
    const ownerId = row.owner_user_id?.trim();
    if (!ownerId || owners.has(ownerId)) {
      continue;
    }
    const email = userEmailById[ownerId];
    owners.set(ownerId, email ? { user_id: ownerId, email } : { user_id: ownerId });
  }

  return json(200, {
    countries: [...countries].sort(),
    owners: [...owners.values()].sort((left, right) => {
      const leftLabel = left.email ?? left.user_id;
      const rightLabel = right.email ?? right.user_id;
      return leftLabel.localeCompare(rightLabel);
    }),
  });
}

function filterDevMockCampaignRows(url: URL, campaigns: Campaign[]): Campaign[] {
  const customerId = url.searchParams.get('customer_id') ?? '';
  const status = url.searchParams.get('status') ?? '';
  const q = (url.searchParams.get('q') ?? '').trim().toLowerCase();
  const pacing = url.searchParams.get('pacing_mode') ?? '';
  const ownerUserId = url.searchParams.get('owner_user_id') ?? '';
  const country = url.searchParams.get('country') ?? '';
  const budgetMin = url.searchParams.get('budget_min_micro');
  const budgetMax = url.searchParams.get('budget_max_micro');

  let rows = [...campaigns];
  if (customerId) {
    rows = rows.filter((row) => row.customer_id === customerId);
  }
  if (pacing) {
    rows = rows.filter((row) => row.pacing_mode === pacing);
  }
  if (ownerUserId) {
    rows = rows.filter((row) => row.owner_user_id === ownerUserId);
  }
  if (country) {
    rows = rows.filter((row) => row.target_countries?.includes(country));
  }
  if (q) {
    rows = rows.filter((row) => row.name.toLowerCase().includes(q) || row.id.includes(q));
  }
  if (budgetMin) {
    const min = Number.parseInt(budgetMin, 10);
    if (Number.isFinite(min)) {
      rows = rows.filter((row) => campaignMoneyMicro(row.budget_limit) >= min);
    }
  }
  if (budgetMax) {
    const max = Number.parseInt(budgetMax, 10);
    if (Number.isFinite(max)) {
      rows = rows.filter((row) => campaignMoneyMicro(row.budget_limit) <= max);
    }
  }
  if (status) {
    rows = rows.filter((row) => row.status === status);
  }
  return rows;
}

export function devMockListCampaignMetricsTotals(
  url: URL,
  campaigns: Campaign[],
): MockListResult {
  const from = url.searchParams.get('from') ?? new Date(Date.now() - 7 * 86_400_000).toISOString();
  const to = url.searchParams.get('to') ?? new Date().toISOString();
  const rows = filterDevMockCampaignRows(url, campaigns);
  const flowCount = rows.filter((row) => row.flow_id).length;

  const totals: Record<string, number> = {
    impressions: 0,
    clicks: 0,
    conversions: 0,
    unique_clicks: 0,
    blocks: 0,
    leads_raw: 0,
    hold_leads: 0,
    rejected_leads: 0,
    lp_clicks: 0,
    lp_views: 0,
    bots: 0,
    advertiser_spend_micro: 0,
    rtb_cost_micro: 0,
    operator_margin_micro: 0,
    publisher_payout_micro: 0,
  };

  let marginBreachCount = 0;

  rows.forEach((row, index) => {
    if ((index + 1) % 13 === 0) {
      marginBreachCount += 1;
    }
    const metrics = buildDevMockCampaignMetrics(row.id, index + 1, from, to);
    totals.impressions += metrics.impressions;
    totals.clicks += metrics.clicks;
    totals.conversions += metrics.conversions;
    totals.unique_clicks += metrics.unique_clicks;
    totals.blocks += metrics.blocks;
    totals.leads_raw += metrics.leads_raw;
    totals.hold_leads += metrics.hold_leads;
    totals.rejected_leads += metrics.rejected_leads;
    totals.lp_clicks += metrics.lp_clicks;
    totals.lp_views += metrics.lp_views;
    totals.bots += metrics.bots;
    totals.advertiser_spend_micro += metrics.rtb_cost_micro + metrics.profit_micro;
    totals.rtb_cost_micro += metrics.rtb_cost_micro;
    totals.operator_margin_micro += metrics.profit_micro;
    totals.publisher_payout_micro += Math.floor(metrics.rtb_cost_micro * 0.72);
  });

  const totalsRow: Record<string, unknown> = { ...totals };
  enrichDevMockCampaignMetricsDerived(totalsRow, {
    impressions: totals.impressions,
    clicks: totals.clicks,
    conversions: totals.conversions,
    leads_raw: totals.leads_raw,
    lp_clicks: totals.lp_clicks,
    lp_views: totals.lp_views,
    blocks: totals.blocks,
    bots: totals.bots,
    rtb_cost_micro: totals.rtb_cost_micro,
    profit_micro: totals.operator_margin_micro,
  });

  return json(200, {
    campaign_count: rows.length,
    flow_count: flowCount,
    margin_breach_count: marginBreachCount,
    totals: totalsRow,
    from,
    to,
    stale: false,
  });
}
