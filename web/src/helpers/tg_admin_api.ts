import { api, ApiError } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type TelegramBotDTO = {
  campaign_id: string;
  bot_id: number;
  bot_token: string;
  webhook_url: string;
  mini_app_url: string;
  secret_token: string;
  auth_date_ttl: number;
};

export type TelegramPostbackDTO = {
  id: string;
  campaign_id: string;
  postback_url: string;
};

export type TelegramDeeplinkDTO = {
  token: string;
  campaign_id: string;
  expires_at: string;
};

/**
 * Fetch Telegram bot config for a campaign (null on 404).
 */
export async function fetchTelegramBot(campaignId: string): Promise<TelegramBotDTO | null> {
  try {
    const res = await api(`/api/v1/telegram/bots/${campaignId}`);
    return (res?.data as TelegramBotDTO | null | undefined) ?? null;
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null;
    throw err;
  }
}

/**
 * Save Telegram bot config for a campaign.
 */
export async function saveTelegramBot(
  campaignId: string,
  bot: Partial<TelegramBotDTO>,
): Promise<void> {
  await apiConfirmed(`/api/v1/telegram/bots/${campaignId}`, {
    method: 'PUT',
    body: JSON.stringify({
      campaign_id: campaignId,
      bot_id: Number(bot.bot_id) || 0,
      bot_token: bot.bot_token ?? '',
      webhook_url: bot.webhook_url ?? '',
      mini_app_url: bot.mini_app_url ?? '',
      secret_token: bot.secret_token ?? '',
      auth_date_ttl: Number(bot.auth_date_ttl) || 300,
    }),
  });
}

/**
 * List Telegram postbacks for a campaign.
 */
export async function fetchTelegramPostbacks(campaignId: string): Promise<TelegramPostbackDTO[]> {
  const q = new URLSearchParams({ campaign_id: campaignId });
  const res = await api(`/api/v1/telegram/postbacks?${q.toString()}`);
  return Array.isArray(res?.data) ? (res.data as TelegramPostbackDTO[]) : [];
}

/**
 * Create a Telegram postback URL.
 */
export async function createTelegramPostback(campaignId: string, postbackUrl: string): Promise<void> {
  await apiConfirmed('/api/v1/telegram/postbacks', {
    method: 'POST',
    body: JSON.stringify({
      id: crypto.randomUUID(),
      campaign_id: campaignId,
      postback_url: postbackUrl,
    }),
  });
}

/**
 * Update a Telegram postback URL.
 */
export async function updateTelegramPostback(postbackId: string, postbackUrl: string): Promise<void> {
  await apiConfirmed(`/api/v1/telegram/postbacks/${postbackId}`, {
    method: 'PUT',
    body: JSON.stringify({ postback_url: postbackUrl }),
  });
}

/**
 * Delete a Telegram postback.
 */
export async function deleteTelegramPostback(postbackId: string): Promise<void> {
  await apiConfirmed(`/api/v1/telegram/postbacks/${postbackId}`, {
    method: 'DELETE',
  });
}

/**
 * Fire a Telegram postback test.
 */
export async function testTelegramPostback(postbackId: string): Promise<void> {
  await apiConfirmed(`/api/v1/telegram/postbacks/${postbackId}/test`, {
    method: 'POST',
  });
}

/**
 * Create a Telegram deeplink token with optional UTM attribution.
 */
export async function createTelegramDeeplink(
  campaignId: string,
  attribution: Record<string, string> = {},
): Promise<TelegramDeeplinkDTO> {
  const res = await apiConfirmed('/api/v1/telegram/deeplink-tokens', {
    method: 'POST',
    body: JSON.stringify({
      campaign_id: campaignId,
      utm_source: attribution.utm_source ?? '',
      utm_medium: attribution.utm_medium ?? '',
      utm_campaign: attribution.utm_campaign ?? '',
    }),
  });
  return (res?.data as TelegramDeeplinkDTO | null | undefined) ?? ({} as TelegramDeeplinkDTO);
}
