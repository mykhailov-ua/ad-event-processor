import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import type { components } from '../types/generated/openapi.js';

export type PostbackConfigRow = components['schemas']['PostbackConfig'];
export type PostbackDlqRow = components['schemas']['PostbackDlqEntry'];
export type PostbackCampaignStatusRow = components['schemas']['PostbackCampaignStatus'];
export type PostbackDryRunResult = components['schemas']['PostbackDryRunResult'];
export type UpdatePostbackConfigRequest = components['schemas']['UpdatePostbackConfigRequest'];

/**
 * Fetch all postback configs and return the row for one campaign.
 */
export async function fetchPostbackConfig(campaignId: string): Promise<PostbackConfigRow | null> {
  const res = await api('/api/v1/postbacks/config');
  const rows = Array.isArray(res.data) ? (res.data as PostbackConfigRow[]) : [];
  return rows.find((row) => row.campaign_id === campaignId) ?? null;
}

/**
 * Upsert outbound postback/CAPI config for a campaign.
 */
export async function savePostbackConfig(
  campaignId: string,
  body: UpdatePostbackConfigRequest
): Promise<void> {
  await apiConfirmed(`/api/v1/postbacks/config/${campaignId}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
}

/**
 * List postback DLQ rows, optionally filtered client-side by campaign.
 */
export async function fetchPostbackDlq(campaignId?: string): Promise<PostbackDlqRow[]> {
  const res = await api('/api/v1/postbacks/dlq');
  const rows = Array.isArray(res.data) ? (res.data as PostbackDlqRow[]) : [];
  if (!campaignId) return rows;
  return rows.filter((row) => row.campaign_id === campaignId);
}

/**
 * List per-campaign postback delivery health.
 */
export async function fetchPostbackCampaignStatus(): Promise<PostbackCampaignStatusRow[]> {
  const res = await api<PostbackCampaignStatusRow[]>('/api/v1/postbacks/campaign-status');
  return res.data ?? [];
}

/**
 * Re-enqueue one DLQ postback by id.
 */
export async function retryPostbackDlq(id: number | string): Promise<void> {
  await apiConfirmed(`/api/v1/postbacks/dlq/${id}/retry`, { method: 'POST', body: '{}' });
}

/**
 * Dry-run postback dispatch for a campaign config.
 */
export async function testPostbackConfig(campaignId: string): Promise<PostbackDryRunResult> {
  const res = await apiConfirmed<PostbackDryRunResult>(
    `/api/v1/postbacks/config/${encodeURIComponent(campaignId)}/test`,
    { method: 'POST', body: '{}' }
  );
  return res.data ?? { ok: false, provider: 'unknown', error: 'empty response', test_event: false };
}
