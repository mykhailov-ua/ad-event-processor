import { api } from './api_client.js';

/**
 * @typedef {{ from: string, to: string, customerId?: string, campaignId?: string }} TelegramReportQuery
 */

/**
 * @param {TelegramReportQuery} params
 * @returns {URLSearchParams}
 */
function buildTelegramQuery(params) {
  const q = new URLSearchParams({ from: params.from, to: params.to });
  if (params.customerId) q.set('customer_id', params.customerId);
  if (params.campaignId) q.set('campaign_id', params.campaignId);
  return q;
}

/**
 * Fetch Telegram summary report JSON.
 *
 * @param {TelegramReportQuery} params
 * @returns {Promise<any>}
 */
export async function fetchTelegramSummary(params) {
  const res = await api(`/api/v1/reports/telegram/summary?${buildTelegramQuery(params).toString()}`);
  return res?.data ?? null;
}

/**
 * Fetch Telegram funnel breakdown.
 *
 * @param {TelegramReportQuery} params
 * @returns {Promise<any>}
 */
export async function fetchTelegramFunnel(params) {
  const res = await api(`/api/v1/reports/telegram/funnel?${buildTelegramQuery(params).toString()}`);
  return res?.data ?? null;
}

/**
 * Fetch Telegram fraud counters.
 *
 * @param {TelegramReportQuery} params
 * @returns {Promise<any>}
 */
export async function fetchTelegramFraud(params) {
  const res = await api(`/api/v1/reports/telegram/fraud?${buildTelegramQuery(params).toString()}`);
  return res?.data ?? null;
}

/**
 * Fetch Telegram bot breakdown report.
 *
 * @param {TelegramReportQuery} params
 * @returns {Promise<any>}
 */
export async function fetchTelegramBots(params) {
  const res = await api(`/api/v1/reports/telegram/bots?${buildTelegramQuery(params).toString()}`);
  return res?.data ?? null;
}

/**
 * Fetch Telegram premium vs non-premium metrics.
 *
 * @param {TelegramReportQuery} params
 * @returns {Promise<any>}
 */
export async function fetchTelegramPremium(params) {
  const res = await api(`/api/v1/reports/telegram/premium?${buildTelegramQuery(params).toString()}`);
  return res?.data ?? null;
}
