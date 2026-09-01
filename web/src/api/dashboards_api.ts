import { apiJson } from './client.js';
import type { DashboardQuery, DashboardRole, RoleDashboard } from './types.js';

const DASHBOARD_PATHS: Record<DashboardRole, string> = {
  buyer: '/api/v1/dashboards/buyer',
  adops: '/api/v1/dashboards/adops',
  cfo: '/api/v1/dashboards/cfo',
  accountant: '/api/v1/dashboards/accountant',
  fraud: '/api/v1/dashboards/fraud',
  operator: '/api/v1/dashboards/operator',
};

export const DASHBOARD_ROLES: DashboardRole[] = [
  'buyer',
  'adops',
  'cfo',
  'accountant',
  'fraud',
  'operator',
];

export function formatDashboardRoleLabel(role: DashboardRole): string {
  const labels: Record<DashboardRole, string> = {
    buyer: 'Buyer',
    adops: 'AdOps',
    cfo: 'CFO',
    accountant: 'Accountant',
    fraud: 'Fraud',
    operator: 'Operator',
  };
  return labels[role];
}

export function isDashboardRole(value: string): value is DashboardRole {
  return (DASHBOARD_ROLES as string[]).includes(value);
}

export function buildDashboardPath(role: DashboardRole, params: DashboardQuery): string {
  const search = new URLSearchParams({ customer_id: params.customer_id });
  if (params.campaign_id) {
    search.set('campaign_id', params.campaign_id);
  }
  if (params.from) {
    search.set('from', params.from);
  }
  if (params.to) {
    search.set('to', params.to);
  }
  return `${DASHBOARD_PATHS[role]}?${search.toString()}`;
}

export async function getRoleDashboard(
  role: DashboardRole,
  params: DashboardQuery,
  signal?: AbortSignal,
): Promise<RoleDashboard> {
  return apiJson<RoleDashboard>(buildDashboardPath(role, params), { signal });
}

export type CampaignDashboardQuery = {
  from?: string;
  to?: string;
};

export function buildCampaignDashboardPath(
  campaignId: string,
  params: CampaignDashboardQuery = {},
): string {
  const search = new URLSearchParams();
  if (params.from) {
    search.set('from', params.from);
  }
  if (params.to) {
    search.set('to', params.to);
  }
  const query = search.toString();
  const base = `/api/v1/dashboards/campaign/${encodeURIComponent(campaignId)}`;
  return query ? `${base}?${query}` : base;
}

export async function getCampaignDashboard(
  campaignId: string,
  params: CampaignDashboardQuery = {},
  signal?: AbortSignal,
): Promise<RoleDashboard> {
  return apiJson<RoleDashboard>(buildCampaignDashboardPath(campaignId, params), { signal });
}
