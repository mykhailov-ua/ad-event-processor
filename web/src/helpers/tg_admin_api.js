import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

/**
 * @typedef {Object} TelegramBotDTO
 * @property {string} campaign_id
 * @property {number} bot_id
 * @property {string} bot_token
 * @property {string} webhook_url
 * @property {string} mini_app_url
 * @property {string} secret_token
 * @property {number} auth_date_ttl
 */

/**
 * @typedef {Object} TelegramPostbackDTO
 * @property {string} id
 * @property {string} campaign_id
 * @property {string} postback_url
 */

/**
 * @typedef {Object} TelegramDeeplinkDTO
 * @property {string} token
 * @property {string} campaign_id
 * @property {string} expires_at
 */

/**
 * @param {string} campaignId
 * @returns {Promise<TelegramBotDTO|null>}
 */
export async function fetchTelegramBot(campaignId) {
  try {
    const res = await api(`/api/v1/telegram/bots/${campaignId}`);
    return res?.data ?? null;
  } catch (err) {
    if (err?.status === 404) return null;
    throw err;
  }
}

/**
 * @param {string} campaignId
 * @param {Partial<TelegramBotDTO>} bot
 * @returns {Promise<void>}
 */
export async function saveTelegramBot(campaignId, bot) {
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
 * @param {string} campaignId
 * @returns {Promise<TelegramPostbackDTO[]>}
 */
export async function fetchTelegramPostbacks(campaignId) {
  const q = new URLSearchParams({ campaign_id: campaignId });
  const res = await api(`/api/v1/telegram/postbacks?${q.toString()}`);
  return Array.isArray(res?.data) ? res.data : [];
}

/**
 * @param {string} campaignId
 * @param {string} postbackUrl
 * @returns {Promise<void>}
 */
export async function createTelegramPostback(campaignId, postbackUrl) {
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
 * @param {string} postbackId
 * @param {string} postbackUrl
 * @returns {Promise<void>}
 */
export async function updateTelegramPostback(postbackId, postbackUrl) {
  await apiConfirmed(`/api/v1/telegram/postbacks/${postbackId}`, {
    method: 'PUT',
    body: JSON.stringify({ postback_url: postbackUrl }),
  });
}

/**
 * @param {string} postbackId
 * @returns {Promise<void>}
 */
export async function deleteTelegramPostback(postbackId) {
  await apiConfirmed(`/api/v1/telegram/postbacks/${postbackId}`, {
    method: 'DELETE',
  });
}

/**
 * @param {string} postbackId
 * @returns {Promise<void>}
 */
export async function testTelegramPostback(postbackId) {
  await apiConfirmed(`/api/v1/telegram/postbacks/${postbackId}/test`, {
    method: 'POST',
  });
}

/**
 * @param {string} campaignId
 * @param {Record<string, string>} [attribution]
 * @returns {Promise<TelegramDeeplinkDTO>}
 */
export async function createTelegramDeeplink(campaignId, attribution = {}) {
  const res = await apiConfirmed('/api/v1/telegram/deeplink-tokens', {
    method: 'POST',
    body: JSON.stringify({
      campaign_id: campaignId,
      utm_source: attribution.utm_source ?? '',
      utm_medium: attribution.utm_medium ?? '',
      utm_campaign: attribution.utm_campaign ?? '',
    }),
  });
  return res?.data ?? {};
}
