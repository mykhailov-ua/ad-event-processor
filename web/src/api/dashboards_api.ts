import { apiJson } from './client';
import type { DashboardQuery } from './types';
import type {
  AdopsDashboardPayload,
  BuyerDashboardPayload,
} from '@/domains/dashboards/dashboard_types';

function buildDashboardQuery(params: DashboardQuery): string {
  const search = new URLSearchParams();
  if (params.customer_id) {
    search.set('customer_id', params.customer_id);
  }
  if (params.from) {
    search.set('from', params.from);
  }
  if (params.to) {
    search.set('to', params.to);
  }
  if ('campaign_id' in params && params.campaign_id) {
    search.set('campaign_id', params.campaign_id);
  }
  const encoded = search.toString();
  return encoded ? `?${encoded}` : '';
}

export async function getBuyerDashboard(
  params: DashboardQuery,
  signal?: AbortSignal
): Promise<BuyerDashboardPayload> {
  return apiJson<BuyerDashboardPayload>(`/api/v1/dashboards/buyer${buildDashboardQuery(params)}`, {
    signal,
  });
}

export async function getAdopsDashboard(
  params: Omit<DashboardQuery, 'campaign_id'>,
  signal?: AbortSignal
): Promise<AdopsDashboardPayload> {
  return apiJson<AdopsDashboardPayload>(`/api/v1/dashboards/adops${buildDashboardQuery(params)}`, {
    signal,
  });
}
