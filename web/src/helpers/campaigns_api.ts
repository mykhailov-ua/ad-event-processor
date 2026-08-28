import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type Campaign = {
  id?: string;
  name?: string;
  status?: string;
  budget_limit?: string;
  current_spend?: string;
  customer_id?: string;
  pacing_mode?: string;
  timezone?: string;
  target_url?: string;
  daily_budget?: string;
  freq_limit?: number;
  freq_window?: number;
  updated_at?: string;
};

export type CampaignFraud = {
  preset?: string;
  fraud_threshold_pass?: number;
  fraud_threshold_suspect?: number;
  fraud_threshold_ivt?: number;
  fraud_threshold_block?: number;
  silent_reject_enabled?: boolean;
};

export type CampaignStats = {
  impressions?: number;
  clicks?: number;
  conversions?: number;
  spend_micro?: number;
  freshness_label?: string;
  stale?: boolean;
};

export type CampaignEvent = {
  id?: string;
  event_type?: string;
  created_at?: string;
  payout_micro?: number;
};

export type CampaignEventListResponse = {
  items?: CampaignEvent[];
  total?: number;
};

export type PostbackConfig = {
  provider?: string;
  url_template?: string;
  api_token?: string;
  target_event?: string;
  test_event_code?: string;
};

export type CampaignDetailTab =
  | 'overview'
  | 'stats'
  | 'config'
  | 'fraud'
  | 'events'
  | 'postbacks';

export const CAMPAIGN_DETAIL_TABS: Array<{ id: CampaignDetailTab; label: string }> = [
  { id: 'overview', label: 'Overview' },
  { id: 'stats', label: 'Statistics' },
  { id: 'config', label: 'Configuration' },
  { id: 'fraud', label: 'Fraud' },
  { id: 'events', label: 'Events' },
  { id: 'postbacks', label: 'Postbacks' },
];

export const CAMPAIGN_MASKED_TABS: CampaignDetailTab[] = ['overview', 'stats', 'config'];

export function parseCampaignDetailTab(raw: string | null): CampaignDetailTab {
  const allowed: CampaignDetailTab[] = [
    'overview',
    'stats',
    'config',
    'fraud',
    'events',
    'postbacks',
  ];
  return allowed.includes(raw as CampaignDetailTab) ? (raw as CampaignDetailTab) : 'overview';
}

export function visibleCampaignTabs(masked: boolean): Array<{ id: CampaignDetailTab; label: string }> {
  if (!masked) return CAMPAIGN_DETAIL_TABS;
  return CAMPAIGN_DETAIL_TABS.filter((tab) => CAMPAIGN_MASKED_TABS.includes(tab.id));
}

export async function fetchCampaign(id: string, signal?: AbortSignal): Promise<Campaign> {
  const result = await api<Campaign>(`/api/v1/campaigns/${encodeURIComponent(id)}`, { signal });
  return result.data ?? {};
}

export async function patchCampaign(
  id: string,
  body: Record<string, unknown>
): Promise<Campaign> {
  const result = await apiConfirmed<Campaign>(`/api/v1/campaigns/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('patch campaign failed');
  }
  return result.data ?? {};
}

export async function fetchCampaignFraud(id: string, signal?: AbortSignal): Promise<CampaignFraud> {
  const result = await api<CampaignFraud>(
    `/api/v1/campaigns/${encodeURIComponent(id)}/fraud`,
    { signal }
  );
  return result.data ?? {};
}

export async function patchCampaignFraud(
  id: string,
  body: CampaignFraud
): Promise<CampaignFraud> {
  const result = await apiConfirmed<CampaignFraud>(
    `/api/v1/campaigns/${encodeURIComponent(id)}/fraud`,
    { method: 'PATCH', body: JSON.stringify(body) }
  );
  if (result.status < 200 || result.status >= 300) {
    throw new Error('patch fraud failed');
  }
  return result.data ?? {};
}

export async function fetchCampaignStats(id: string, signal?: AbortSignal): Promise<CampaignStats> {
  const result = await api<CampaignStats>(
    `/api/v1/campaigns/${encodeURIComponent(id)}/stats`,
    { signal }
  );
  return result.data ?? {};
}

export function buildCampaignEventsUrl(id: string, limit: number, offset: number): string {
  const qs = new URLSearchParams({ limit: String(limit), offset: String(offset) });
  return `/api/v1/campaigns/${encodeURIComponent(id)}/events?${qs.toString()}`;
}

export async function runPublishCheck(id: string): Promise<unknown> {
  const result = await apiConfirmed(
    `/api/v1/campaigns/${encodeURIComponent(id)}/publish-check`,
    { method: 'POST', body: '{}' }
  );
  return result.data;
}

export async function publishCampaign(id: string): Promise<void> {
  const result = await apiConfirmed(`/api/v1/campaigns/${encodeURIComponent(id)}/publish`, {
    method: 'POST',
    body: '{}',
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('publish failed');
  }
}

export async function fetchPostbackConfig(
  campaignId: string,
  signal?: AbortSignal
): Promise<PostbackConfig> {
  const result = await api<PostbackConfig>(
    `/api/v1/postbacks/config/${encodeURIComponent(campaignId)}`,
    { signal }
  );
  return result.data ?? {};
}

export async function putPostbackConfig(
  campaignId: string,
  body: PostbackConfig
): Promise<PostbackConfig> {
  const result = await apiConfirmed<PostbackConfig>(
    `/api/v1/postbacks/config/${encodeURIComponent(campaignId)}`,
    { method: 'PUT', body: JSON.stringify(body) }
  );
  if (result.status < 200 || result.status >= 300) {
    throw new Error('save postback failed');
  }
  return result.data ?? {};
}

export type CampaignListResponse = {
  items?: Campaign[];
  total?: number;
  limit?: number;
  offset?: number;
};

export type CampaignSortField = 'name' | 'updated_at' | 'spend';
export type CampaignSortOrder = 'asc' | 'desc';

export type CampaignListParams = {
  limit: number;
  offset: number;
  sort: CampaignSortField;
  order: CampaignSortOrder;
  customer_id?: string;
  status?: string;
  q?: string;
  pacing_mode?: string;
};

export type CampaignBulkAction = 'pause' | 'resume';

export function buildCampaignsListUrl(params: CampaignListParams): string {
  const qs = new URLSearchParams({
    limit: String(params.limit),
    offset: String(params.offset),
    sort: params.sort,
    order: params.order,
  });
  if (params.customer_id) qs.set('customer_id', params.customer_id);
  if (params.status) qs.set('status', params.status);
  if (params.q) qs.set('q', params.q);
  if (params.pacing_mode) qs.set('pacing_mode', params.pacing_mode);
  return `/api/v1/campaigns?${qs.toString()}`;
}

export async function bulkMutateCampaigns(
  action: CampaignBulkAction,
  campaignIds: string[]
): Promise<void> {
  const result = await apiConfirmed('/api/v1/campaigns/bulk', {
    method: 'POST',
    body: JSON.stringify({ action, campaign_ids: campaignIds }),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('bulk mutate failed');
  }
}
