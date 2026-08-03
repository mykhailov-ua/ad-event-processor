import { api } from './api_client.js';

/**
 * Fetch Telegram summary report JSON.
 *
 * @param {{ from: string, to: string, campaignId?: string }} params
 * @returns {Promise<any>}
 */
export async function fetchTelegramSummary(params) {
  const q = new URLSearchParams({ from: params.from, to: params.to });
  if (params.campaignId) {
    q.set('campaign_id', params.campaignId);
  }
  const res = await api(`/api/v1/reports/telegram/summary?${q.toString()}`);
  return res?.data ?? null;
}

/**
 * Fetch Telegram funnel breakdown.
 *
 * @param {{ from: string, to: string, campaignId?: string }} params
 * @returns {Promise<any>}
 */
export async function fetchTelegramFunnel(params) {
  const q = new URLSearchParams({ from: params.from, to: params.to });
  if (params.campaignId) {
    q.set('campaign_id', params.campaignId);
  }
  const res = await api(`/api/v1/reports/telegram/funnel?${q.toString()}`);
  return res?.data ?? null;
}

/**
 * Fetch Telegram fraud counters.
 *
 * @param {{ from: string, to: string, campaignId?: string }} params
 * @returns {Promise<any>}
 */
export async function fetchTelegramFraud(params) {
  const q = new URLSearchParams({ from: params.from, to: params.to });
  if (params.campaignId) {
    q.set('campaign_id', params.campaignId);
  }
  const res = await api(`/api/v1/reports/telegram/fraud?${q.toString()}`);
  return res?.data ?? null;
}
