import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type PostbackConfigRow = {
  campaign_id?: string;
  [key: string]: unknown;
};

export type PostbackDlqRow = {
  id?: number | string;
  campaign_id?: string;
  event_type?: string;
  failures_count?: number;
  status?: string;
  last_error?: string;
  provider?: string;
  [key: string]: unknown;
};

export type PostbackCampaignStatusRow = {
  campaign_id: string;
  provider: string;
  last_success_at?: string;
  dlq_pending_count: number;
};

export type PostbackDryRunResult = {
  ok: boolean;
  provider: string;
  http_status?: number;
  error?: string;
  rendered_url?: string;
  target_event?: string;
  test_event?: boolean;
};

export async function fetchPostbackConfig(campaignId: string): Promise<PostbackConfigRow | null> {
  const res = await api('/api/v1/postbacks/config');
  const rows = Array.isArray(res.data) ? (res.data as PostbackConfigRow[]) : [];
  return rows.find((row) => row.campaign_id === campaignId) ?? null;
}

export async function savePostbackConfig(
  campaignId: string,
  body: Record<string, unknown>
): Promise<void> {
  await apiConfirmed(`/api/v1/postbacks/config/${campaignId}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
}

export async function fetchPostbackDlq(campaignId?: string): Promise<PostbackDlqRow[]> {
  const res = await api('/api/v1/postbacks/dlq');
  const rows = Array.isArray(res.data) ? (res.data as PostbackDlqRow[]) : [];
  if (!campaignId) return rows;
  return rows.filter((row) => row.campaign_id === campaignId);
}

export async function fetchPostbackCampaignStatus(): Promise<PostbackCampaignStatusRow[]> {
  const res = await api<PostbackCampaignStatusRow[]>('/api/v1/postbacks/campaign-status');
  return res.data ?? [];
}

export async function retryPostbackDlq(id: number | string): Promise<void> {
  await apiConfirmed(`/api/v1/postbacks/dlq/${id}/retry`, { method: 'POST', body: '{}' });
}

export async function testPostbackConfig(campaignId: string): Promise<PostbackDryRunResult> {
  const res = await apiConfirmed<PostbackDryRunResult>(
    `/api/v1/postbacks/config/${encodeURIComponent(campaignId)}/test`,
    { method: 'POST', body: '{}' }
  );
  return res.data ?? { ok: false, provider: 'unknown', error: 'empty response' };
}
