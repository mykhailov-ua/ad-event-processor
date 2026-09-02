// Offline /api/v1 stub when admin dev mode is on (api/client.ts). Unhandled paths return undefined.
import type { Campaign } from '@/api/types';

import {
  devMockDashboardSummary,
  devMockDoctorSummary,
  devMockIncidentSnapshot,
  devMockOpsHomeSnapshot,
  devMockOpsList,
  devMockOpsObject,
  devMockOpsShardsResponse,
  devMockStackHealthSnapshot,
} from './ops_fixtures.ts';
import {
  devMockOnboardingTemplates,
  devMockWizardSessionGet,
  devMockWizardSessionPost,
} from './wizard_fixtures.ts';
import { DEV_MOCK_CUSTOMERS, DEV_MOCK_USERS, devMockStore } from './store.ts';

type MockResult = {
  status: number;
  body?: unknown;
  contentType?: string;
};

function json(status: number, body: unknown): MockResult {
  return { status, body, contentType: 'application/json' };
}

function emptyList(limit = 50, offset = 0): MockResult {
  return json(200, { items: [], total: 0, limit, offset });
}

function parseJsonBody(init?: RequestInit): unknown {
  if (!init?.body || typeof init.body !== 'string') {
    return undefined;
  }
  try {
    return JSON.parse(init.body) as unknown;
  } catch {
    return undefined;
  }
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

function listCampaigns(url: URL): MockResult {
  const { campaigns } = devMockStore();
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
  const limit = Math.min(200, Math.max(1, Number.parseInt(url.searchParams.get('limit') ?? '50', 10) || 50));
  const offset = Math.max(0, Number.parseInt(url.searchParams.get('offset') ?? '0', 10) || 0);

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
      rows = rows.filter((row) => Math.round(Number.parseFloat(row.budget_limit) * 1_000_000) >= min);
    }
  }
  if (budgetMax) {
    const max = Number.parseInt(budgetMax, 10);
    if (Number.isFinite(max)) {
      rows = rows.filter((row) => Math.round(Number.parseFloat(row.budget_limit) * 1_000_000) <= max);
    }
  }

  const statusTotals = countDevMockStatusTotals(rows);

  if (status) {
    rows = rows.filter((row) => row.status === status);
  }

  rows.sort((left, right) => {
    let cmp = 0;
    if (sort === 'name') {
      cmp = left.name.localeCompare(right.name);
    } else if (sort === 'updated_at') {
      cmp = left.updated_at.localeCompare(right.updated_at);
    } else if (sort === 'spend') {
      cmp = Number.parseFloat(left.current_spend) - Number.parseFloat(right.current_spend);
    } else if (sort === 'budget_limit') {
      cmp = Number.parseFloat(left.budget_limit) - Number.parseFloat(right.budget_limit);
    } else {
      cmp = left.name.localeCompare(right.name);
    }
    return order === 'desc' ? -cmp : cmp;
  });

  const total = rows.length;
  const items = rows.slice(offset, offset + limit);
  const filtersApplied = filtersAppliedFromQuery(url, [
    'customer_id',
    'status',
    'q',
    'pacing_mode',
    'owner_user_id',
    'country',
    'budget_min_micro',
    'budget_max_micro',
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

function campaignMetricsRangeScale(fromIso: string, toIso: string): number {
  const fromMs = Date.parse(fromIso);
  const toMs = Date.parse(toIso);
  if (!Number.isFinite(fromMs) || !Number.isFinite(toMs) || toMs <= fromMs) {
    return 1;
  }
  const days = Math.max(1, (toMs - fromMs) / 86_400_000);
  return days / 7;
}

function campaignMetrics(url: URL): MockResult {
  const ids = (url.searchParams.get('ids') ?? '')
    .split(',')
    .map((id) => id.trim())
    .filter(Boolean);
  const from = url.searchParams.get('from') ?? new Date(Date.now() - 7 * 86400_000).toISOString();
  const to = url.searchParams.get('to') ?? new Date().toISOString();
  const scale = campaignMetricsRangeScale(from, to);
  const scaleCount = (value: number) => Math.max(0, Math.round(value * scale));
  const items: Record<string, Record<string, unknown>> = {};

  for (const [index, campaignId] of ids.entries()) {
    const seq = index + 1;
    const impressions = scaleCount(12_000 + seq * 1_340);
    const clicks = scaleCount(420 + seq * 37);
    const conversions = scaleCount(18 + (seq % 11));
    const holdLeads = scaleCount(3 + (seq % 5));
    const rejectedLeads = scaleCount(2 + (seq % 4));
    const leadsRaw = conversions + holdLeads + rejectedLeads;
    const lpClicks = scaleCount(Math.floor((420 + seq * 37) * (0.42 + (seq % 7) * 0.04)));
    const lpViews = Math.max(lpClicks, scaleCount(Math.floor((420 + seq * 37) * 0.88)));
    const bots = scaleCount(4 + (seq % 9));
    const rtbCost = scaleCount(1_200_000 + seq * 88_000);
    const margin = scaleCount(180_000 + seq * 12_000 - (seq % 5) * 40_000);
    items[campaignId] = {
      campaign_id: campaignId,
      impressions,
      clicks,
      conversions,
      leads_raw: leadsRaw,
      hold_leads: holdLeads,
      rejected_leads: rejectedLeads,
      lp_clicks: lpClicks,
      lp_views: lpViews,
      unique_clicks: scaleCount(Math.floor((420 + seq * 37) * 0.82)),
      blocks: seq % 7,
      bots,
      advertiser_spend_micro: rtbCost + margin,
      rtb_cost_micro: rtbCost,
      operator_margin_micro: margin,
      publisher_payout_micro: Math.floor(rtbCost * 0.72),
      margin_breach: seq % 13 === 0,
    };
  }

  return json(200, { items, from, to, stale: false });
}

function patchCampaign(campaignId: string, body: unknown): MockResult {
  const record = body && typeof body === 'object' ? (body as Record<string, unknown>) : {};
  const { campaigns } = devMockStore();
  const index = campaigns.findIndex((row) => row.id === campaignId);
  if (index < 0) {
    return json(404, { error: { code: 'NOT_FOUND', message: 'Campaign not found' } });
  }
  const current = campaigns[index];
  const next: Campaign = {
    ...current,
    ...record,
    id: current.id,
    updated_at: new Date().toISOString(),
  } as Campaign;
  campaigns[index] = next;
  return json(200, next);
}

function bulkCampaignAction(body: unknown): MockResult {
  const record = body && typeof body === 'object' ? (body as Record<string, unknown>) : {};
  const action = record.action === 'resume' ? 'resume' : 'pause';
  const ids = Array.isArray(record.campaign_ids) ? (record.campaign_ids as string[]) : [];
  const { campaigns } = devMockStore();
  const results = ids.map((id) => {
    const row = campaigns.find((campaign) => campaign.id === id);
    if (!row) {
      return { id, ok: false, error_code: 'NOT_FOUND' };
    }
    row.status = action === 'pause' ? 'PAUSED' : 'ACTIVE';
    row.updated_at = new Date().toISOString();
    return { id, ok: true };
  });
  return json(200, { results });
}

function sessionBootstrap(): MockResult {
  const customerId = DEV_MOCK_CUSTOMERS[0].id;
  return json(200, {
    user: {
      id: DEV_MOCK_USERS[0].id,
      email: DEV_MOCK_USERS[0].email,
      role: 'admin',
      customer_id: customerId,
      permissions: [
        'campaigns:read',
        'campaigns:write',
        'customers:read',
        'customers:write',
        'audit:read',
        'settings:read',
        'rtb:read',
        'billing:read',
      ],
    },
    session: {
      role: 'admin',
      mask_level: 'full',
      default_customer_id: customerId,
      timezone: 'UTC',
    },
    eula_required: false,
    eula_accepted: true,
    eula_version: 'dev',
  });
}

function metaResponse(): MockResult {
  return json(200, {
    product_name: 'ad-event-processor',
    vendor_name: 'dev',
    version: 'dev-mock',
    bootstrap_complete: true,
    eula_required: false,
    eula_accepted: true,
    license: { state: 'ACTIVE', tier: 'dev' },
  });
}

function customersList(url: URL): MockResult {
  const limit = Number.parseInt(url.searchParams.get('limit') ?? '50', 10) || 50;
  const offset = Number.parseInt(url.searchParams.get('offset') ?? '0', 10) || 0;
  const items = DEV_MOCK_CUSTOMERS.map((customer) => ({
    id: customer.id,
    name: customer.name,
    status: 'ACTIVE',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  }));
  return json(200, {
    items: items.slice(offset, offset + limit),
    total: items.length,
    limit,
    offset,
  });
}

function teamMembers(): MockResult {
  return json(200, {
    items: DEV_MOCK_USERS.map((user) => ({
      user_id: user.id,
      email: user.email,
      role: 'buyer',
      campaigns_owned: 0,
    })),
    total: DEV_MOCK_USERS.length,
  });
}

function selfServeTemplates(): MockResult {
  return json(200, {
    items: [
      { id: 'tpl-search', name: 'Search traffic', description: 'Default search template' },
      { id: 'tpl-social', name: 'Social traffic', description: 'Social placements template' },
      { id: 'tpl-native', name: 'Native display', description: 'Native display template' },
    ],
  });
}

function campaignById(campaignId: string): MockResult {
  const row = devMockStore().campaigns.find((campaign) => campaign.id === campaignId);
  if (!row) {
    return json(404, { error: { code: 'NOT_FOUND', message: 'Campaign not found' } });
  }
  return json(200, row);
}

function campaignEditorShell(campaignId: string): MockResult {
  return json(200, {
    campaign_id: campaignId,
    sections: [
      {
        id: 'general',
        title: 'General',
        order: 1,
        visible: true,
        complete: true,
        issue_count: 0,
      },
      {
        id: 'targeting',
        title: 'Targeting',
        order: 2,
        visible: true,
        complete: false,
        issue_count: 1,
        issue_tone: 'warn',
      },
      {
        id: 'tracking',
        title: 'Tracking',
        order: 3,
        visible: true,
        complete: true,
        issue_count: 0,
      },
    ],
    completion_pct: 67,
    allowed_actions: ['publish', 'pause'],
  });
}

function putCampaignOwner(campaignId: string, body: unknown): MockResult {
  const record = body && typeof body === 'object' ? (body as Record<string, unknown>) : {};
  const userId = record.user_id;
  if (typeof userId !== 'string' || userId.length === 0) {
    return json(400, { error: { code: 'BAD_REQUEST', message: 'user_id required' } });
  }

  const { campaigns } = devMockStore();
  const index = campaigns.findIndex((row) => row.id === campaignId);
  if (index < 0) {
    return json(404, { error: { code: 'NOT_FOUND', message: 'Campaign not found' } });
  }

  campaigns[index] = {
    ...campaigns[index],
    owner_user_id: userId,
    updated_at: new Date().toISOString(),
  };
  return json(200, { status: 'ok' });
}

function previewCampaignClone(campaignId: string, body: unknown): MockResult {
  const row = devMockStore().campaigns.find((campaign) => campaign.id === campaignId);
  if (!row) {
    return json(404, { error: { code: 'NOT_FOUND', message: 'Campaign not found' } });
  }

  const record = body && typeof body === 'object' ? (body as Record<string, unknown>) : {};
  const options =
    record.options && typeof record.options === 'object'
      ? (record.options as Record<string, unknown>)
      : {};
  const namePrefix = typeof record.name_prefix === 'string' ? record.name_prefix : '';
  const nameSuffix = typeof record.name_suffix === 'string' ? record.name_suffix : ' (copy)';

  return json(200, {
    source_id: campaignId,
    name: `${namePrefix}${row.name}${nameSuffix}`,
    would_create: {
      include_flow: options.include_flow !== false,
      include_postbacks: options.include_postbacks !== false,
      include_fraud: options.include_fraud !== false,
      include_placement_blocks: options.include_placement_blocks !== false,
      reset_spend: options.reset_spend === true,
    },
  });
}

function genericOk(): MockResult {
  return json(200, { status: 'ok' });
}

function reportCatalog(): MockResult {
  return json(200, {
    items: [
      { key: 'campaign-performance', title: 'Campaign performance', category: 'campaigns' },
      { key: 'click-log', title: 'Click log', category: 'traffic' },
    ],
  });
}

function settingsView(): MockResult {
  return json(200, {
    config: {},
    secrets: {},
    restart_required: false,
    templates: [],
  });
}

export function resolveDevMockRequest(path: string, init?: RequestInit): MockResult | undefined {
  if (!path.startsWith('/api/')) {
    return undefined;
  }

  const url = new URL(path, 'http://dev.local');
  const method = (init?.method ?? 'GET').toUpperCase();
  const pathname = url.pathname;

  if (method === 'GET' && pathname === '/api/v1/meta') {
    return metaResponse();
  }
  if (method === 'GET' && pathname === '/api/v1/session/bootstrap') {
    return sessionBootstrap();
  }
  if (method === 'GET' && pathname === '/api/v1/auth/me') {
    const boot = sessionBootstrap();
    const body = boot.body as { user: unknown } | undefined;
    return json(200, body?.user ?? {});
  }
  if (method === 'GET' && pathname === '/api/v1/session') {
    const boot = sessionBootstrap();
    const body = boot.body as { session: unknown } | undefined;
    return json(200, body?.session ?? {});
  }
  if (method === 'POST' && (pathname === '/api/v1/auth/login' || pathname === '/api/v1/auth/refresh')) {
    return genericOk();
  }
  if (method === 'POST' && pathname === '/api/v1/auth/logout') {
    return { status: 204 };
  }
  if (method === 'GET' && pathname === '/api/v1/customers') {
    return customersList(url);
  }
  if (method === 'GET' && pathname.startsWith('/api/v1/customers/')) {
    const customerId = decodeURIComponent(pathname.slice('/api/v1/customers/'.length));
    const customer = DEV_MOCK_CUSTOMERS.find((row) => row.id === customerId);
    if (!customer) {
      return json(404, { error: { code: 'NOT_FOUND', message: 'Customer not found' } });
    }
    return json(200, {
      id: customer.id,
      name: customer.name,
      status: 'ACTIVE',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    });
  }
  if (method === 'GET' && pathname === '/api/v1/campaigns') {
    return listCampaigns(url);
  }
  if (method === 'GET' && pathname === '/api/v1/campaigns/metrics') {
    return campaignMetrics(url);
  }
  if (method === 'GET' && pathname === '/api/v1/campaigns/onboarding-templates') {
    return json(200, devMockOnboardingTemplates());
  }
  if (method === 'GET' && pathname === '/api/v1/campaigns/wizard/session') {
    const sessionId = url.searchParams.get('session_id') ?? '';
    const result = devMockWizardSessionGet(sessionId);
    return json(result.status, result.body);
  }
  if (method === 'POST' && pathname === '/api/v1/campaigns/wizard/session') {
    const body = (parseJsonBody(init) ?? {}) as Record<string, unknown>;
    const result = devMockWizardSessionPost(body);
    return json(result.status, result.body);
  }
  if (
    method === 'POST' &&
    (pathname === '/api/v1/campaigns/bulk' || pathname === '/api/v1/campaigns/bulk-action')
  ) {
    return bulkCampaignAction(parseJsonBody(init));
  }
  if (method === 'PUT' && pathname.startsWith('/api/v1/campaigns/')) {
    const rest = pathname.slice('/api/v1/campaigns/'.length);
    const [campaignId, ...segments] = rest.split('/');
    if (segments.length === 1 && segments[0] === 'owner') {
      return putCampaignOwner(decodeURIComponent(campaignId), parseJsonBody(init));
    }
  }
  if (method === 'POST' && pathname.startsWith('/api/v1/campaigns/')) {
    const rest = pathname.slice('/api/v1/campaigns/'.length);
    const [campaignId, ...segments] = rest.split('/');
    if (segments.length === 1 && segments[0] === 'clone-preview') {
      return previewCampaignClone(decodeURIComponent(campaignId), parseJsonBody(init));
    }
  }
  if (method === 'PATCH' && pathname.startsWith('/api/v1/campaigns/')) {
    const rest = pathname.slice('/api/v1/campaigns/'.length);
    if (!rest.includes('/')) {
      return patchCampaign(decodeURIComponent(rest), parseJsonBody(init));
    }
  }
  if (method === 'GET' && pathname.startsWith('/api/v1/campaigns/')) {
    const rest = pathname.slice('/api/v1/campaigns/'.length);
    const [campaignId, ...segments] = rest.split('/');
    if (segments.length === 0) {
      return campaignById(decodeURIComponent(campaignId));
    }
    if (segments[0] === 'stats') {
      return json(200, {
        impressions: 42_000,
        clicks: 1_240,
        conversions: 86,
        spend_micro: 3_400_000,
      });
    }
    if (segments[0] === 'margin') {
      return json(200, {
        campaign_id: decodeURIComponent(campaignId),
        window_start: new Date().toISOString(),
        window_hours: 24,
        advertiser_spend_micro: 4_200_000,
        rtb_cost_micro: 3_100_000,
        operator_margin_micro: 1_100_000,
        publisher_payout_micro: 2_200_000,
      });
    }
    if (segments[0] === 'events') {
      return emptyList();
    }
    if (segments[0] === 'integration-panel') {
      return json(200, { sections: [] });
    }
    if (segments[0] === 'integration-health') {
      return json(200, { campaign_id: decodeURIComponent(campaignId), summary: 'ok', rows: [] });
    }
    if (segments[0] === 'fraud' || segments[0] === 'fraud-editor') {
      return json(200, { campaign_id: decodeURIComponent(campaignId), enabled: false });
    }
    if (segments[0] === 'geo-summary') {
      return json(200, { countries: [] });
    }
    if (segments[0] === 'conversion-mappings') {
      return json(200, { items: [], total: 0 });
    }
    if (segments[0] === 'editor') {
      return campaignEditorShell(decodeURIComponent(campaignId));
    }
  }
  if (method === 'GET' && pathname === '/api/v1/selfserve/templates') {
    return selfServeTemplates();
  }
  if (method === 'GET' && pathname === '/api/v1/team/members') {
    return teamMembers();
  }
  if (method === 'GET' && pathname === '/api/v1/reports/catalog') {
    return reportCatalog();
  }
  if (method === 'GET' && pathname === '/api/v1/settings/platform') {
    return settingsView();
  }
  if (method === 'GET' && pathname === '/api/v1/license/status') {
    return json(200, { state: 'ACTIVE', tier: 'dev' });
  }
  if (method === 'GET' && pathname === '/api/v1/eula') {
    return json(200, { accepted: true, version: 'dev' });
  }
  if (method === 'GET' && pathname.startsWith('/api/v1/dashboards/')) {
    return json(200, { series: [], totals: {} });
  }
  if (method === 'GET' && pathname.startsWith('/api/v1/reports/')) {
    return json(200, { columns: [], rows: [], total: 0 });
  }
  if (method === 'GET' && pathname.startsWith('/api/v1/audit')) {
    return emptyList();
  }
  if (method === 'GET' && pathname === '/api/v1/ops/home') {
    return devMockOpsHomeSnapshot();
  }
  if (method === 'GET' && pathname === '/api/v1/ops/doctor') {
    return json(200, devMockDoctorSummary());
  }
  if (method === 'GET' && pathname === '/api/v1/ops/health/snapshot') {
    return json(200, devMockStackHealthSnapshot());
  }
  if (method === 'GET' && pathname === '/api/v1/ops/dashboard/summary') {
    return json(200, devMockDashboardSummary());
  }
  if (method === 'GET' && pathname === '/api/v1/ops/incidents') {
    return devMockIncidentSnapshot();
  }
  if (method === 'GET' && pathname === '/api/v1/ops/shards') {
    return devMockOpsShardsResponse();
  }
  if (method === 'GET' && pathname === '/api/v1/ops/dlq') {
    return json(200, { items: [], partial: false });
  }
  if (method === 'POST' && pathname.startsWith('/api/v1/ops/dlq/') && pathname.endsWith('/retry')) {
    const id = decodeURIComponent(
      pathname.slice('/api/v1/ops/dlq/'.length, pathname.length - '/retry'.length),
    );
    if (id && !id.includes('/')) {
      return { status: 202 };
    }
  }
  if (
    method === 'GET' &&
    (pathname === '/api/v1/ops/dlq/inbox' ||
      pathname === '/api/v1/ops/blacklist' ||
      pathname === '/api/v1/ops/outbox' ||
      pathname.startsWith('/api/v1/ops/recon'))
  ) {
    const url = new URL(path, 'http://dev.local');
    const limit = Number.parseInt(url.searchParams.get('limit') ?? '50', 10) || 50;
    const offset = Number.parseInt(url.searchParams.get('offset') ?? '0', 10) || 0;
    return devMockOpsList(limit, offset);
  }
  if (method === 'GET' && pathname.startsWith('/api/v1/ops/')) {
    return devMockOpsObject();
  }
  if (method === 'GET' && pathname.startsWith('/api/v1/billing')) {
    return emptyList();
  }
  if (method === 'GET' && pathname.startsWith('/api/v1/rtb')) {
    return emptyList();
  }
  if (method === 'GET' && pathname.startsWith('/api/v1/fraud')) {
    return emptyList();
  }
  if (method === 'GET' && pathname.startsWith('/api/v1/integrations')) {
    return emptyList();
  }
  if (method === 'GET' && pathname.startsWith('/api/v1/automation')) {
    return emptyList();
  }
  if (method === 'POST' && pathname === '/api/v1/consent') {
    return { status: 204 };
  }
  if (method === 'GET' && pathname.startsWith('/api/v1/')) {
    return emptyList();
  }
  if (method === 'POST' || method === 'PATCH' || method === 'PUT' || method === 'DELETE') {
    return genericOk();
  }

  return undefined;
}

export function devMockResponse(path: string, init?: RequestInit): Response | undefined {
  const resolved = resolveDevMockRequest(path, init);
  if (!resolved) {
    return undefined;
  }
  if (resolved.status === 204 || resolved.status === 202) {
    return new Response(null, { status: resolved.status });
  }
  return new Response(JSON.stringify(resolved.body ?? {}), {
    status: resolved.status,
    headers: {
      'Content-Type': resolved.contentType ?? 'application/json',
    },
  });
}
