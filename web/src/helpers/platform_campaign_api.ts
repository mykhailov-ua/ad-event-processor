import { api, ApiError } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import { getOrCreate } from './idempotency.js';

/** Linked Meta or Google Ads campaign row from `/api/v1/platform-campaigns/links`. */
export type PlatformCampaignLink = {
  campaign_id: string;
  customer_id: string;
  network: string;
  external_campaign_id: string;
  account_id?: string;
  external_status?: string;
  external_daily_budget_micro?: number;
  last_synced_at?: string;
  sync_error?: string;
  updated_at: string;
};

/** Upsert body for `PUT /api/v1/platform-campaigns/links/{campaign_id}/{network}`. */
export type UpsertPlatformCampaignLinkRequest = {
  customer_id: string;
  external_campaign_id: string;
  account_id?: string;
};

/** Mutation result from pause, resume, or budget endpoints. */
export type PlatformCampaignMutation = {
  idempotency_key: string;
  status: string;
  action: string;
  network: string;
  campaign_id: string;
  error_message?: string;
  preview?: unknown;
  response?: unknown;
};

/** Supported ad platform networks for platform campaign sync. */
export const PLATFORM_CAMPAIGN_NETWORKS = ['facebook', 'google'] as const;

export type PlatformCampaignNetwork = (typeof PLATFORM_CAMPAIGN_NETWORKS)[number];

/**
 * Returns true when the deployment JWT lacks the enterprise platform campaign API flag.
 */
export function isPlatformCampaignLicenseError(err: unknown): boolean {
  return err instanceof ApiError && err.status === 403 && err.code === 'LICENSE_FORBIDDEN';
}

/**
 * List platform campaign links, optionally filtered by campaign id.
 */
export async function fetchPlatformCampaignLinks(
  campaignId: string
): Promise<PlatformCampaignLink[]> {
  const res = await api<PlatformCampaignLink[]>(
    `/api/v1/platform-campaigns/links?campaign_id=${encodeURIComponent(campaignId)}`
  );
  return Array.isArray(res.data) ? res.data : [];
}

/**
 * Create or update a link between an ad-event-processor campaign and a Meta/Google campaign.
 */
export async function upsertPlatformCampaignLink(
  campaignId: string,
  network: PlatformCampaignNetwork,
  body: UpsertPlatformCampaignLinkRequest
): Promise<PlatformCampaignLink> {
  const res = await apiConfirmed(
    `/api/v1/platform-campaigns/links/${encodeURIComponent(campaignId)}/${encodeURIComponent(network)}`,
    {
      method: 'PUT',
      body: JSON.stringify(body),
    }
  );
  return res.data as PlatformCampaignLink;
}

/**
 * Remove a platform campaign link.
 */
export async function deletePlatformCampaignLink(
  campaignId: string,
  network: PlatformCampaignNetwork
): Promise<void> {
  await apiConfirmed(
    `/api/v1/platform-campaigns/links/${encodeURIComponent(campaignId)}/${encodeURIComponent(network)}`,
    { method: 'DELETE' }
  );
}

/**
 * Pull latest status and budget from the ad platform for one link.
 */
export async function refreshPlatformCampaignLink(
  campaignId: string,
  network: PlatformCampaignNetwork
): Promise<PlatformCampaignLink> {
  const res = await apiConfirmed(
    `/api/v1/platform-campaigns/links/${encodeURIComponent(campaignId)}/${encodeURIComponent(network)}/refresh`,
    { method: 'POST' }
  );
  return res.data as PlatformCampaignLink;
}

/**
 * Enqueue a manual sync run for all links on a campaign.
 */
export async function runPlatformCampaignSync(campaignId: string): Promise<void> {
  await apiConfirmed('/api/v1/platform-campaigns/sync-run', {
    method: 'POST',
    body: JSON.stringify({ campaign_id: campaignId }),
  });
}

/**
 * Pause the linked campaign on Meta or Google Ads.
 */
export async function pausePlatformCampaign(
  campaignId: string,
  network: PlatformCampaignNetwork
): Promise<PlatformCampaignMutation> {
  const scope = `platform-pause:${campaignId}:${network}`;
  const res = await apiConfirmed(
    `/api/v1/platform-campaigns/${encodeURIComponent(campaignId)}/pause`,
    {
      method: 'POST',
      body: JSON.stringify({
        network,
        idempotency_key: getOrCreate(scope),
      }),
      idempotencyScope: scope,
    }
  );
  return res.data as PlatformCampaignMutation;
}

/**
 * Resume the linked campaign on Meta or Google Ads.
 */
export async function resumePlatformCampaign(
  campaignId: string,
  network: PlatformCampaignNetwork
): Promise<PlatformCampaignMutation> {
  const scope = `platform-resume:${campaignId}:${network}`;
  const res = await apiConfirmed(
    `/api/v1/platform-campaigns/${encodeURIComponent(campaignId)}/resume`,
    {
      method: 'POST',
      body: JSON.stringify({
        network,
        idempotency_key: getOrCreate(scope),
      }),
      idempotencyScope: scope,
    }
  );
  return res.data as PlatformCampaignMutation;
}

/**
 * Set daily budget cap on the linked Meta or Google campaign.
 */
export async function setPlatformCampaignDailyBudget(
  campaignId: string,
  network: PlatformCampaignNetwork,
  dailyBudgetMicro: number
): Promise<PlatformCampaignMutation> {
  const scope = `platform-budget:${campaignId}:${network}:${dailyBudgetMicro}`;
  const res = await apiConfirmed(
    `/api/v1/platform-campaigns/${encodeURIComponent(campaignId)}/budget`,
    {
      method: 'POST',
      body: JSON.stringify({
        network,
        idempotency_key: getOrCreate(scope),
        daily_budget_micro: dailyBudgetMicro,
      }),
      idempotencyScope: scope,
    }
  );
  return res.data as PlatformCampaignMutation;
}
