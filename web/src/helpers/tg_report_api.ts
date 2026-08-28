import { api } from './api_client.js';

export type TelegramReportQuery = {
  from: string;
  to: string;
  customerId?: string;
  campaignId?: string;
};

function buildTelegramQuery(params: TelegramReportQuery): URLSearchParams {
  const q = new URLSearchParams({ from: params.from, to: params.to });
  if (params.customerId) q.set('customer_id', params.customerId);
  if (params.campaignId) q.set('campaign_id', params.campaignId);
  return q;
}

export async function fetchTelegramSummary(params: TelegramReportQuery): Promise<unknown> {
  const res = await api(
    `/api/v1/reports/telegram/summary?${buildTelegramQuery(params).toString()}`
  );
  return res?.data ?? null;
}

export async function fetchTelegramFunnel(params: TelegramReportQuery): Promise<unknown> {
  const res = await api(`/api/v1/reports/telegram/funnel?${buildTelegramQuery(params).toString()}`);
  return res?.data ?? null;
}

export async function fetchTelegramFraud(params: TelegramReportQuery): Promise<unknown> {
  const res = await api(`/api/v1/reports/telegram/fraud?${buildTelegramQuery(params).toString()}`);
  return res?.data ?? null;
}

export async function fetchTelegramBots(params: TelegramReportQuery): Promise<unknown> {
  const res = await api(`/api/v1/reports/telegram/bots?${buildTelegramQuery(params).toString()}`);
  return res?.data ?? null;
}

export async function fetchTelegramPremium(params: TelegramReportQuery): Promise<unknown> {
  const res = await api(
    `/api/v1/reports/telegram/premium?${buildTelegramQuery(params).toString()}`
  );
  return res?.data ?? null;
}
