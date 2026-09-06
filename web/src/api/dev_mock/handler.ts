// Non-prod tier: offline API stub when admin dev mode is on (api/client.ts). See web/DESIGN.md.
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
import {
  devMockListCampaignFacets,
  devMockListCampaignMetricsTotals,
  devMockListCampaigns,
  devMockCampaignStats,
  devMockExportCampaignsBatch,
} from './campaign_list.ts';
import {
  buildDevMockCampaignMetrics,
  enrichDevMockCampaignMetricsDerived,
} from './campaign_metrics.ts';
import {
  devMockBillingInvariant,
  devMockBillingSummary,
  devMockCustomerBalance,
  devMockCustomerLedger,
  devMockCustomerStatement,
  devMockCustomerWallet,
  devMockInvoiceById,
  devMockInvoicesList,
} from './billing_fixtures.ts';
import {
  devMockBareArrayForPath,
  devMockCostSyncSnapshot,
  devMockIntegrationSnapshot,
  devMockPostbacksSnapshot,
} from './catalog_fixtures.ts';
import { devMockAuditList } from './audit_fixtures.ts';
import { devMockRoleDashboard } from './dashboard_fixtures.ts';
import {
  devMockClickLogReport,
  devMockReportCatalog,
  devMockReportEnvelope,
} from './reports_fixtures.ts';
import {
  devMockTeamBudgetApprovals,
  devMockTeamMembersList,
  devMockTeamOverview,
} from './team_fixtures.ts';
import type { MockResult } from './handler_types.ts';
import {
  devMockAuthorizeRequest,
  devMockCurrentPermissions,
  devMockSessionRoleLabel,
  getDevMockRole,
} from './rbac.ts';
import {
  devMockApplyPlatformSettings,
  devMockPatchPlatformSettings,
  devMockPlatformSettingsView,
} from './settings_fixtures.ts';
import { DEV_MOCK_CUSTOMERS, DEV_MOCK_USERS, devMockStore } from './store.ts';

function json(status: number, body: unknown): MockResult {
  return { status, body, contentType: 'application/json' };
}

function emptyList(limit = 50, offset = 0): MockResult {
  return json(200, { items: [], total: 0, limit, offset });
}

function paginatedList(items: unknown[], url: URL): MockResult {
  const limit = Number.parseInt(url.searchParams.get('limit') ?? '50', 10) || 50;
  const offset = Number.parseInt(url.searchParams.get('offset') ?? '0', 10) || 0;
  return json(200, {
    items: items.slice(offset, offset + limit),
    total: items.length,
    limit,
    offset,
  });
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

function listCampaigns(url: URL): MockResult {
  return devMockListCampaigns(url, devMockStore().campaigns);
}

function listCampaignFacets(url: URL): MockResult {
  const userEmailById = Object.fromEntries(DEV_MOCK_USERS.map((user) => [user.id, user.email]));
  return devMockListCampaignFacets(url, devMockStore().campaigns, userEmailById);
}

function campaignMetrics(url: URL): MockResult {
  const ids = (url.searchParams.get('ids') ?? '')
    .split(',')
    .map((id) => id.trim())
    .filter(Boolean);
  const from = url.searchParams.get('from') ?? new Date(Date.now() - 7 * 86400_000).toISOString();
  const to = url.searchParams.get('to') ?? new Date().toISOString();
  const items: Record<string, Record<string, unknown>> = {};

  for (const [index, campaignId] of ids.entries()) {
    const metrics = buildDevMockCampaignMetrics(campaignId, index + 1, from, to);
    const item: Record<string, unknown> = {
      campaign_id: campaignId,
      impressions: metrics.impressions,
      clicks: metrics.clicks,
      conversions: metrics.conversions,
      leads_raw: metrics.leads_raw,
      hold_leads: metrics.hold_leads,
      rejected_leads: metrics.rejected_leads,
      lp_clicks: metrics.lp_clicks,
      lp_views: metrics.lp_views,
      unique_clicks: metrics.unique_clicks,
      blocks: metrics.blocks,
      bots: metrics.bots,
      advertiser_spend_micro: metrics.rtb_cost_micro + metrics.profit_micro,
      rtb_cost_micro: metrics.rtb_cost_micro,
      operator_margin_micro: metrics.profit_micro,
      publisher_payout_micro: Math.floor(metrics.rtb_cost_micro * 0.72),
      margin_breach: (index + 1) % 13 === 0,
    };
    enrichDevMockCampaignMetricsDerived(item, metrics);
    items[campaignId] = item;
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
  const actionRaw = typeof record.action === 'string' ? record.action : 'pause';
  const action = actionRaw === 'resume' ? 'resume' : actionRaw === 'archive' ? 'archive' : 'pause';
  const ids = Array.isArray(record.campaign_ids) ? (record.campaign_ids as string[]) : [];
  const { campaigns } = devMockStore();
  const results = ids.map((id) => {
    const row = campaigns.find((campaign) => campaign.id === id);
    if (!row) {
      return { id, ok: false, error_code: 'NOT_FOUND' };
    }
    if (action === 'archive') {
      row.status = 'ARCHIVED';
    } else {
      row.status = action === 'pause' ? 'PAUSED' : 'ACTIVE';
    }
    row.updated_at = new Date().toISOString();
    return { id, ok: true };
  });
  return json(200, { results });
}

function sessionBootstrap(): MockResult {
  const customerId = DEV_MOCK_CUSTOMERS[0].id;
  const role = getDevMockRole();
  const permissions = devMockCurrentPermissions();
  const sessionRole = devMockSessionRoleLabel(role);
  return json(200, {
    user: {
      id: DEV_MOCK_USERS[0].id,
      email: DEV_MOCK_USERS[0].email,
      role: sessionRole,
      customer_id: customerId,
      permissions,
    },
    session: {
      role: sessionRole,
      mask_level: role === 'B' || role === 'S' ? 'masked' : 'full',
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
    payment_enabled: true,
    license: { state: 'ACTIVE', tier: 'dev' },
  });
}

function customersList(url: URL): MockResult {
  const limit = Number.parseInt(url.searchParams.get('limit') ?? '50', 10) || 50;
  const offset = Number.parseInt(url.searchParams.get('offset') ?? '0', 10) || 0;
  const items = DEV_MOCK_CUSTOMERS.map((customer, index) => ({
    id: customer.id,
    name: customer.name,
    status: 'ACTIVE',
    currency: 'USD',
    balance: (12_000 + index * 2450.75).toFixed(6),
    cost_center: `CC-${customer.name.slice(0, 3).toUpperCase()}`,
    active_campaigns: devMockStore().campaigns.filter((row) => row.customer_id === customer.id)
      .length,
    total_spend: (84_200 + index * 12_400).toFixed(6),
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
  return json(200, devMockTeamMembersList());
}

function selfServeTemplates(): MockResult {
  return json(200, {
    items: [
      {
        id: 'meta_social_funnel',
        name: 'Meta social funnel',
        description: 'Facebook and Instagram click-to-lander flow with conversion mapping.',
      },
      {
        id: 'push_house_funnel',
        name: 'Push house funnel',
        description: 'Push notification source with hold-friendly postback mapping.',
      },
      {
        id: 'native_mgid_funnel',
        name: 'Native MGID funnel',
        description: 'Native placements with teaser URL macros and geo targeting.',
      },
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

function brandById(brandId: string): MockResult {
  const now = new Date().toISOString();
  return json(200, {
    id: brandId,
    customer_id: DEV_MOCK_CUSTOMERS[0].id,
    name: `Brand ${brandId.slice(0, 8)}`,
    created_at: now,
    updated_at: now,
    freq_limit: 0,
    freq_window: 0,
  });
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

function mockNotImplemented(): MockResult {
  return json(501, {
    error: { code: 'NOT_IMPLEMENTED', message: 'Route not implemented in dev mock' },
  });
}

function authLoginResponse(): MockResult {
  const boot = sessionBootstrap();
  const body = boot.body as { user: unknown };
  return json(200, { user: body.user });
}

function authRefreshResponse(): MockResult {
  return json(200, { status: 'ok' });
}

function reportCatalog(): MockResult {
  return json(200, devMockReportCatalog());
}

function settingsView(): MockResult {
  return json(200, devMockPlatformSettingsView());
}

function parseMockJsonBody(init?: RequestInit): Record<string, unknown> | undefined {
  const raw = init?.body;
  if (typeof raw !== 'string' || !raw.trim()) {
    return undefined;
  }
  try {
    const parsed: unknown = JSON.parse(raw);
    return isRecord(parsed) ? parsed : undefined;
  } catch {
    return undefined;
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value != null && typeof value === 'object' && !Array.isArray(value);
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
  if (
    method === 'POST' &&
    (pathname === '/api/v1/auth/login' || pathname === '/api/v1/auth/refresh')
  ) {
    return pathname.endsWith('/login') ? authLoginResponse() : authRefreshResponse();
  }
  if (method === 'POST' && pathname === '/api/v1/auth/logout') {
    return { status: 204 };
  }

  const denied = devMockAuthorizeRequest(pathname, method);
  if (denied) {
    return denied;
  }

  if (method === 'GET' && pathname === '/api/v1/customers') {
    return customersList(url);
  }
  if (method === 'GET' && pathname.startsWith('/api/v1/customers/')) {
    const rest = pathname.slice('/api/v1/customers/'.length);
    const [customerId, ...segments] = rest.split('/');
    const decodedId = decodeURIComponent(customerId);
    const customer = DEV_MOCK_CUSTOMERS.find((row) => row.id === decodedId);
    if (!customer) {
      return json(404, { error: { code: 'NOT_FOUND', message: 'Customer not found' } });
    }
    if (segments.length === 0) {
      const index = DEV_MOCK_CUSTOMERS.findIndex((row) => row.id === decodedId);
      return json(200, {
        id: customer.id,
        name: customer.name,
        status: 'ACTIVE',
        currency: 'USD',
        balance: (12_000 + index * 2450.75).toFixed(6),
        cost_center: `CC-${customer.name.slice(0, 3).toUpperCase()}`,
        active_campaigns: devMockStore().campaigns.filter((row) => row.customer_id === customer.id)
          .length,
        total_spend: (84_200 + index * 12_400).toFixed(6),
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      });
    }
    if (segments[0] === 'balance') {
      return json(200, devMockCustomerBalance(decodedId));
    }
    if (segments[0] === 'wallet') {
      return json(200, devMockCustomerWallet(decodedId));
    }
    if (segments[0] === 'ledger') {
      return json(200, devMockCustomerLedger(decodedId, url));
    }
    if (segments[0] === 'billing' && segments[1] === 'statement') {
      return json(200, devMockCustomerStatement(decodedId, url));
    }
  }
  if (method === 'GET' && pathname === '/api/v1/campaigns') {
    return listCampaigns(url);
  }
  if (method === 'GET' && pathname === '/api/v1/campaigns/list-facets') {
    return listCampaignFacets(url);
  }
  if (method === 'GET' && pathname === '/api/v1/campaigns/metrics') {
    return campaignMetrics(url);
  }
  if (method === 'GET' && pathname === '/api/v1/campaigns/metrics-totals') {
    return devMockListCampaignMetricsTotals(url, devMockStore().campaigns);
  }
  if (method === 'GET' && pathname === '/api/v1/campaigns/export') {
    return devMockExportCampaignsBatch(url);
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
      return devMockCampaignStats(decodeURIComponent(campaignId), url);
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
  if (method === 'GET') {
    const brandMatch = /^\/api\/v1\/brands\/([^/]+)$/.exec(pathname);
    if (brandMatch) {
      return brandById(decodeURIComponent(brandMatch[1]));
    }
  }
  if (method === 'GET' && pathname === '/api/v1/team/overview') {
    const customerId = url.searchParams.get('customer_id')?.trim();
    return json(200, devMockTeamOverview(customerId));
  }
  if (method === 'GET' && pathname === '/api/v1/team/budget-approvals') {
    return json(200, devMockTeamBudgetApprovals(url));
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
  if (method === 'PATCH' && pathname === '/api/v1/settings/platform') {
    const patch = parseMockJsonBody(init);
    if (!patch) {
      return json(400, { error: { code: 'BAD_REQUEST', message: 'Patch must be a JSON object' } });
    }
    return json(200, devMockPatchPlatformSettings(patch));
  }
  if (method === 'POST' && pathname === '/api/v1/settings/platform/apply') {
    const body = parseMockJsonBody(init);
    return json(
      200,
      devMockApplyPlatformSettings(
        typeof body?.install_root === 'string' ? body.install_root : undefined
      )
    );
  }
  if (method === 'GET' && pathname === '/api/v1/license/status') {
    return json(200, { state: 'ACTIVE', tier: 'dev' });
  }
  if (method === 'GET' && pathname === '/api/v1/eula') {
    return json(200, { accepted: true, version: 'dev' });
  }
  if (method === 'GET' && pathname.startsWith('/api/v1/dashboards/')) {
    return json(200, devMockRoleDashboard(url, pathname));
  }
  if (method === 'GET' && pathname === '/api/v1/reports/click-log') {
    return json(200, devMockClickLogReport(url));
  }
  if (method === 'GET' && pathname.startsWith('/api/v1/reports/')) {
    const reportKey = decodeURIComponent(pathname.slice('/api/v1/reports/'.length));
    if (reportKey === 'catalog' || reportKey.startsWith('jobs')) {
      return emptyList();
    }
    return json(200, devMockReportEnvelope(reportKey));
  }
  if (method === 'GET' && pathname.startsWith('/api/v1/audit')) {
    return json(200, devMockAuditList(url));
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
      pathname.slice('/api/v1/ops/dlq/'.length, pathname.length - '/retry'.length)
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
    return devMockOpsList(pathname, limit, offset);
  }
  if (method === 'GET' && pathname.startsWith('/api/v1/ops/')) {
    return devMockOpsObject();
  }
  if (method === 'GET' && pathname === '/api/v1/billing/summary') {
    return json(200, devMockBillingSummary());
  }
  if (method === 'GET' && pathname === '/api/v1/billing/invariant') {
    const customerId = url.searchParams.get('customer_id')?.trim();
    return json(200, devMockBillingInvariant(customerId));
  }
  if (method === 'GET' && pathname === '/api/v1/billing/invoices') {
    return json(200, devMockInvoicesList(url));
  }
  if (method === 'GET' && pathname.startsWith('/api/v1/billing/invoices/')) {
    const invoiceRest = pathname.slice('/api/v1/billing/invoices/'.length);
    if (!invoiceRest.includes('/')) {
      const invoice = devMockInvoiceById(decodeURIComponent(invoiceRest));
      if (!invoice) {
        return json(404, { error: { code: 'NOT_FOUND', message: 'Invoice not found' } });
      }
      return json(200, invoice);
    }
  }
  if (method === 'GET' && pathname.startsWith('/api/v1/billing')) {
    return emptyList();
  }
  if (method === 'POST' && pathname === '/api/v1/consent') {
    return { status: 204 };
  }
  if (method === 'GET' && pathname === '/api/v1/cost-sync/snapshot') {
    return json(200, devMockCostSyncSnapshot());
  }
  if (method === 'GET' && pathname === '/api/v1/postbacks/snapshot') {
    return json(200, devMockPostbacksSnapshot());
  }
  if (method === 'GET' && pathname === '/api/v1/integration/snapshot') {
    return json(200, devMockIntegrationSnapshot());
  }
  if (method === 'GET') {
    const catalogItems = devMockBareArrayForPath(pathname);
    if (catalogItems) {
      return json(200, catalogItems);
    }
  }
  if (method === 'GET' && pathname.startsWith('/api/v1/')) {
    const listItems = devMockBareArrayForPath(pathname);
    if (listItems) {
      return paginatedList(listItems, url);
    }
    return emptyList();
  }
  if (method === 'POST' || method === 'PATCH' || method === 'PUT' || method === 'DELETE') {
    return mockNotImplemented();
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
