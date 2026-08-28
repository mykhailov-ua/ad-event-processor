import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type TelegramBot = {
  id?: string;
  campaign_id?: string;
  bot_id?: string;
  webhook_url?: string;
  mini_app_url?: string;
};

export type TelegramPostback = {
  id?: string;
  campaign_id?: string;
  postback_url?: string;
};

export type TelegramDetailTab = 'bots' | 'postbacks' | 'deeplink' | 'reports';

export const TELEGRAM_DETAIL_TABS: Array<{ id: TelegramDetailTab; label: string }> = [
  { id: 'bots', label: 'Bots' },
  { id: 'postbacks', label: 'Postbacks' },
  { id: 'deeplink', label: 'Deeplink' },
  { id: 'reports', label: 'Reports' },
];

export function parseTelegramDetailTab(raw: string | null): TelegramDetailTab {
  const allowed: TelegramDetailTab[] = ['bots', 'postbacks', 'deeplink', 'reports'];
  return allowed.includes(raw as TelegramDetailTab) ? (raw as TelegramDetailTab) : 'bots';
}

export async function fetchTelegramBots(signal?: AbortSignal): Promise<TelegramBot[]> {
  const result = await api<TelegramBot[]>('/api/v1/telegram/bots', { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function configureTelegramBot(
  botId: string,
  body: Record<string, unknown>
): Promise<void> {
  const result = await apiConfirmed(`/api/v1/telegram/bots/${encodeURIComponent(botId)}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('configure bot failed');
  }
}

export async function fetchTelegramPostbacks(
  campaignId: string,
  signal?: AbortSignal
): Promise<TelegramPostback[]> {
  const qs = new URLSearchParams({ campaign_id: campaignId });
  const result = await api<TelegramPostback[]>(`/api/v1/telegram/postbacks?${qs.toString()}`, {
    signal,
  });
  return Array.isArray(result.data) ? result.data : [];
}

export async function createTelegramPostback(body: {
  campaign_id: string;
  postback_url: string;
}): Promise<void> {
  const result = await apiConfirmed('/api/v1/telegram/postbacks', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('create postback failed');
  }
}

export async function createDeeplinkToken(body: Record<string, string>): Promise<unknown> {
  const result = await apiConfirmed('/api/v1/telegram/deeplink-tokens', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  return result.data;
}
