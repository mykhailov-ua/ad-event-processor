import type { CostSyncNetwork } from '../models/traffic_source_templates.js';

/** Cost Sync credential network id (matches `internal/costsync/fetch.go` and cost-sync UI). */
export type CostSyncApiNetworkId = 'facebook' | 'google' | 'tiktok';

export type CostSyncRequiredKey = {
  key: string;
  hint: string;
};

export type CostSyncUrlHints = {
  apiNetworkId: CostSyncApiNetworkId;
  label: string;
  requiredKeys: CostSyncRequiredKey[];
  tokenMappingDefault: string;
};

/** Required click query keys per Cost Sync network (from `docs/INTEGRATIONS.md` + provider join fields). */
export const COST_SYNC_URL_HINTS: Record<CostSyncNetwork, CostSyncUrlHints> = {
  meta: {
    apiNetworkId: 'facebook',
    label: 'Facebook / Meta',
    requiredKeys: [
      {
        key: 'ad_campaign_id',
        hint: 'Network campaign id (Cost Sync joins on campaign-level spend; mirror in sub2).',
      },
      {
        key: 'sub2',
        hint: 'Same external campaign id as ad_campaign_id (template: {{campaign.id}}).',
      },
      {
        key: 'sub4',
        hint: 'Ad id when token_mapping network_object=ad_id (Facebook provider default).',
      },
      { key: 'fbclid', hint: 'Platform click id; forwarded to lander and CAPI.' },
    ],
    tokenMappingDefault:
      'placement_field=placement_id or sub4, network_object=ad_id, attribution_mode=token',
  },
  google: {
    apiNetworkId: 'google',
    label: 'Google Ads',
    requiredKeys: [
      { key: 'ad_campaign_id', hint: 'Google {campaignid} macro for Cost Sync campaign join.' },
      { key: 'sub2', hint: 'Mirror ad_campaign_id ({campaignid}).' },
      { key: 'sub3', hint: 'Ad group id ({adgroupid}) when using ad-group-level token_mapping.' },
      { key: 'gclid', hint: 'Google click id for attribution.' },
    ],
    tokenMappingDefault:
      'placement_field=placement_id or sub3, network_object=ad_id, attribution_mode=token',
  },
  tiktok: {
    apiNetworkId: 'tiktok',
    label: 'TikTok Ads',
    requiredKeys: [
      { key: 'ad_campaign_id', hint: 'TikTok campaign id (__CAMPAIGN_ID__) for Cost Sync join.' },
      { key: 'sub2', hint: 'Mirror ad_campaign_id (__CAMPAIGN_ID__).' },
      { key: 'sub3', hint: 'Ad group id (__AID__) when token_mapping uses adset-level objects.' },
      { key: 'ttclid', hint: 'TikTok click id for attribution.' },
    ],
    tokenMappingDefault:
      'placement_field=placement_id or sub2, network_object=placement_id, attribution_mode=token',
  },
};

/**
 * Returns Cost Sync join hints for a traffic-source template network tag, or null when absent.
 */
export function costSyncHintsForNetwork(
  network: CostSyncNetwork | undefined
): CostSyncUrlHints | null {
  if (!network) return null;
  return COST_SYNC_URL_HINTS[network] ?? null;
}
