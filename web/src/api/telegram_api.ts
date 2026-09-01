import { apiFetch, apiJson } from './client.js';
import type {
  TelegramBot,
  TelegramDeeplink,
  TelegramListPostbacksQuery,
  TelegramPostback,
  TelegramUpdatePostbackRequest,
  TelegramValidateRequest,
  TelegramValidateResult,
} from './types.js';

export async function listTelegramBots(signal?: AbortSignal): Promise<TelegramBot[]> {
  return apiJson<TelegramBot[]>('/api/v1/telegram/bots', { signal });
}

export async function getTelegramBot(campaignId: string, signal?: AbortSignal): Promise<TelegramBot> {
  return apiJson<TelegramBot>(`/api/v1/telegram/bots/${encodeURIComponent(campaignId)}`, { signal });
}

export async function configureTelegramBot(
  campaignId: string,
  body: TelegramBot,
  signal?: AbortSignal,
): Promise<void> {
  const response = await apiFetch(`/api/v1/telegram/bots/${encodeURIComponent(campaignId)}`, {
    method: 'PUT',
    body: JSON.stringify(body),
    signal,
  });
  if (!response.ok) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
}

export async function listTelegramPostbacks(
  params: TelegramListPostbacksQuery,
  signal?: AbortSignal,
): Promise<TelegramPostback[]> {
  const search = new URLSearchParams();
  search.set('campaign_id', params.campaign_id);
  return apiJson<TelegramPostback[]>(`/api/v1/telegram/postbacks?${search.toString()}`, { signal });
}

export async function createTelegramPostback(
  body: TelegramPostback,
  signal?: AbortSignal,
): Promise<void> {
  const response = await apiFetch('/api/v1/telegram/postbacks', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
  if (!response.ok) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
}

export async function updateTelegramPostback(
  id: string,
  body: TelegramUpdatePostbackRequest,
  signal?: AbortSignal,
): Promise<void> {
  const response = await apiFetch(`/api/v1/telegram/postbacks/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(body),
    signal,
  });
  if (!response.ok) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
}

export async function deleteTelegramPostback(id: string, signal?: AbortSignal): Promise<void> {
  const response = await apiFetch(`/api/v1/telegram/postbacks/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    signal,
  });
  if (!response.ok) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
}

export async function testTelegramPostback(id: string, signal?: AbortSignal): Promise<void> {
  const response = await apiFetch(
    `/api/v1/telegram/postbacks/${encodeURIComponent(id)}/test`,
    { method: 'POST', signal },
  );
  if (!response.ok) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
}

export async function createTelegramDeeplink(
  body: TelegramDeeplink,
  signal?: AbortSignal,
): Promise<TelegramDeeplink> {
  return apiJson<TelegramDeeplink>('/api/v1/telegram/deeplink-tokens', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function getTelegramDeeplink(token: string, signal?: AbortSignal): Promise<TelegramDeeplink> {
  return apiJson<TelegramDeeplink>(
    `/api/v1/telegram/deeplink-tokens/${encodeURIComponent(token)}`,
    { signal },
  );
}

export async function validateTelegramInitData(
  body: TelegramValidateRequest,
  signal?: AbortSignal,
): Promise<TelegramValidateResult> {
  return apiJson<TelegramValidateResult>('/api/v1/telegram/validate', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}
