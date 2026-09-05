import { listCampaigns } from '@/api/campaigns_api';
import type { CampaignListQuery, CampaignListResponse } from '@/api/types';

import {
  campaignListSelectionScopeKey,
  type CampaignListSelectionScope,
} from '@/domains/campaigns/list/campaign_list_selection_scope';

const MAX_CACHE_ENTRIES = 32;

const cache = new Map<string, CampaignListResponse>();

export function campaignListResponseCacheKey(scope: CampaignListSelectionScope): string {
  return campaignListSelectionScopeKey(scope);
}

export function invalidateCampaignListResponseCache(): void {
  cache.clear();
}

function rememberCampaignListResponse(key: string, response: CampaignListResponse): void {
  if (cache.has(key)) {
    cache.delete(key);
  }
  cache.set(key, response);
  while (cache.size > MAX_CACHE_ENTRIES) {
    const oldestKey = cache.keys().next().value;
    if (oldestKey === undefined) {
      break;
    }
    cache.delete(oldestKey);
  }
}

export async function fetchCampaignListCached(
  query: CampaignListQuery,
  statsFrom: string | undefined,
  statsTo: string | undefined,
  signal?: AbortSignal,
): Promise<CampaignListResponse> {
  const key = campaignListResponseCacheKey({ query, statsFrom, statsTo });
  const cached = cache.get(key);
  if (cached) {
    return cached;
  }

  const response = await listCampaigns(query, signal);
  rememberCampaignListResponse(key, response);
  return response;
}
