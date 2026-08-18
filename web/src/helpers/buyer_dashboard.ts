import { api } from './api_client.js';
import { coalesce } from './request_multiplex.js';
import { probeStart, probeEnd } from './perf_probe.js';
import { mapBuyerDashboard } from '../models/buyer.js';
import type { BuyerPortfolioResponse } from '../types/campaign.js';

const DASHBOARD_TTL_MS = 60_000;

export type BuyerPortfolioVM = ReturnType<typeof mapBuyerDashboard>;

type DashboardCacheEntry = {
  at: number;
  data: BuyerPortfolioVM;
};

const dashboardCache = new Map<string, DashboardCacheEntry>();

export function invalidateBuyerDashboard(customerId = ''): void {
  if (customerId) {
    dashboardCache.delete(cacheKey(customerId));
    return;
  }
  dashboardCache.clear();
}

function cacheKey(customerId: string): string {
  return customerId || '_session';
}

async function fetchBuyerDashboardNetwork(customerId: string): Promise<BuyerPortfolioVM> {
  const probe = probeStart('buyer.dashboard');
  const params = new URLSearchParams();
  if (customerId) params.set('customer_id', customerId);
  const qs = params.toString();
  const path = qs ? `/api/v1/dashboards/buyer?${qs}` : '/api/v1/dashboards/buyer';
  const { data } = await api<BuyerPortfolioResponse>(path);
  const mapped = mapBuyerDashboard(data);
  probeEnd(probe, { allocs: 1, bytes: mapped.campaigns.length * 160 });
  dashboardCache.set(cacheKey(customerId), { at: Date.now(), data: mapped });
  return mapped;
}

export async function fetchBuyerDashboard(customerId = ''): Promise<BuyerPortfolioVM> {
  const key = cacheKey(customerId);
  const hit = dashboardCache.get(key);
  if (hit && Date.now() - hit.at < DASHBOARD_TTL_MS) {
    return hit.data;
  }
  return coalesce(`buyer-dashboard:${key}`, () => fetchBuyerDashboardNetwork(customerId));
}

export function loadBuyerDashboard(customerId = ''): Promise<BuyerPortfolioVM> {
  return fetchBuyerDashboard(customerId);
}
