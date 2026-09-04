import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { CampaignStats, CampaignStatsQuery } from '@/api/types';

export function buildCampaignStatsCacheKey(
  campaignId: string,
  statsQuery: CampaignStatsQuery,
  cacheRevision: string,
): string {
  return [
    cacheRevision,
    campaignId,
    statsQuery.from ?? '',
    statsQuery.to ?? '',
    statsQuery.granularity ?? '',
  ].join('\0');
}

export function campaignStatsFromListMetrics(
  campaignId: string,
  metrics: CampaignListMetrics,
  statsQuery: CampaignStatsQuery = {},
): CampaignStats {
  return {
    campaign_id: campaignId,
    current_spend: '0',
    metrics: {
      impressions: metrics.impressions ?? 0,
      clicks: metrics.clicks ?? 0,
      conversions: metrics.conversions ?? 0,
    },
    hourly: [],
    granularity: statsQuery.granularity ?? 'hour',
    from: statsQuery.from ?? '',
    to: statsQuery.to ?? '',
    stale: metrics.stale === true,
    source: 'list_batch',
    consistency: 'list_batch',
  };
}

const statsCache = new Map<string, CampaignStats>();

export function readCachedCampaignStats(cacheKey: string): CampaignStats | undefined {
  return statsCache.get(cacheKey);
}

function statsQueryKeyParts(statsQuery: CampaignStatsQuery): [string, string, string] {
  return [statsQuery.from ?? '', statsQuery.to ?? '', statsQuery.granularity ?? ''];
}

/** Match any session revision (e.g. list scope) when editor does not know cacheRevision. */
export function readCachedCampaignStatsForCampaign(
  campaignId: string,
  statsQuery: CampaignStatsQuery = {},
): CampaignStats | undefined {
  const [from, to, granularity] = statsQueryKeyParts(statsQuery);
  for (const [key, stats] of statsCache) {
    const parts = key.split('\0');
    if (parts.length < 5) {
      continue;
    }
    if (parts[1] !== campaignId) {
      continue;
    }
    if (parts[2] !== from || parts[3] !== to || parts[4] !== granularity) {
      continue;
    }
    return stats;
  }
  return undefined;
}

export function writeCachedCampaignStats(cacheKey: string, stats: CampaignStats): void {
  statsCache.set(cacheKey, stats);
}

export function clearCampaignStatsCache(): void {
  statsCache.clear();
}
