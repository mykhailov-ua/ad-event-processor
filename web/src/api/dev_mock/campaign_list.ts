import type { Campaign } from '@/api/types';
import { campaignBudgetUsedPercent } from '@/lib/campaign_budget_used.ts';

import {
  buildDevMockCampaignMetrics,
  compareDevMockCampaignMetricSort,
  devMockCampaignMetricSortNeedsWindow,
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
