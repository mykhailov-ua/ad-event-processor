import { api } from './api_client.js';
import { coalesce } from './request_multiplex.js';
import { probeStart, probeEnd } from './perf_probe.js';
import { mapBuyerDashboard } from '../models/buyer.js';

const DASHBOARD_TTL_MS = 60_000;

/** @type {Map<string, { at: number, data: import('../models/buyer.js').BuyerPortfolioVM }>} */
const dashboardCache = new Map();

/**
 * Drop cached buyer dashboard payload (all customers when id omitted).
 *
 * @param {string} [customerId]
 * @returns {void}
 */
export function invalidateBuyerDashboard(customerId = '') {
  if (customerId) {
    dashboardCache.delete(cacheKey(customerId));
    return;
  }
  dashboardCache.clear();
}

/**
 * @param {string} customerId
 * @returns {string}
 */
function cacheKey(customerId) {
  return customerId || '_session';
}

/**
 * Fetch buyer dashboard from network and update cache.
 *
 * @param {string} customerId
 * @returns {Promise<import('../models/buyer.js').BuyerPortfolioVM>}
 */
async function fetchBuyerDashboardNetwork(customerId) {
  const probe = probeStart('buyer.dashboard');
  const params = new URLSearchParams();
  if (customerId) params.set('customer_id', customerId);
  const qs = params.toString();
  const path = qs ? `/api/v1/dashboards/buyer?${qs}` : '/api/v1/dashboards/buyer';
  const { data } = await api(path);
  const mapped = mapBuyerDashboard(data);
  probeEnd(probe, { allocs: 1, bytes: mapped.campaigns.length * 160 });
  dashboardCache.set(cacheKey(customerId), { at: Date.now(), data: mapped });
  return mapped;
}

/**
 * Fetch buyer portfolio from the dashboards API (coalesced, TTL-cached).
 *
 * @param {string} [customerId]
 * @returns {Promise<import('../models/buyer.js').BuyerPortfolioVM>}
 */
export async function fetchBuyerDashboard(customerId = '') {
  const key = cacheKey(customerId);
  const hit = dashboardCache.get(key);
  if (hit && Date.now() - hit.at < DASHBOARD_TTL_MS) {
    return hit.data;
  }
  return coalesce(`buyer-dashboard:${key}`, () => fetchBuyerDashboardNetwork(customerId));
}

/**
 * Fetch buyer dashboard for table stat lookups (cached/coalesced).
 *
 * @param {string} [customerId]
 * @returns {Promise<import('../models/buyer.js').BuyerPortfolioVM>}
 */
export function loadBuyerDashboard(customerId = '') {
  return fetchBuyerDashboard(customerId);
}
