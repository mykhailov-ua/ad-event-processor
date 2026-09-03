import type { CampaignListQuery } from '@/api/types';
import { OPTIMAL_LIST_LIMIT_MAX } from '@/lib/list_query';

export type CampaignListWidthProbeListSnapshot = {
  items: readonly { id: string }[];
  total: number;
};

/**
 * List query for column-width sampling: same filters as the directory list,
 * fixed name sort, first OPTIMAL_LIST_LIMIT_MAX rows.
 */
export function buildCampaignListWidthProbeQuery(
  query: CampaignListQuery,
): CampaignListQuery {
  return {
    customer_id: query.customer_id,
    status: query.status,
    q: query.q,
    pacing_mode: query.pacing_mode,
    budget_min_micro: query.budget_min_micro,
    budget_max_micro: query.budget_max_micro,
    owner_user_id: query.owner_user_id,
    country: query.country,
    limit: OPTIMAL_LIST_LIMIT_MAX,
    offset: 0,
    sort: 'name',
    order: 'asc',
  };
}

/**
 * True when the current list response already contains every filtered row
 * (total within probe cap), so a separate width-probe list fetch is redundant.
 */
export function listResponseCoversWidthProbeDataset(
  data: CampaignListWidthProbeListSnapshot | undefined,
): boolean {
  if (!data || data.total === 0) {
    return false;
  }
  if (data.total > OPTIMAL_LIST_LIMIT_MAX) {
    return false;
  }
  return data.items.length === data.total;
}

/** Deduped campaign ids for a single metrics batch (page rows + optional probe rows). */
export function mergeCampaignIdsForMetricsBatch(
  ...idLists: readonly (readonly string[])[]
): string[] {
  const seen = new Set<string>();
  const merged: string[] = [];
  for (const ids of idLists) {
    for (const id of ids) {
      if (!id || seen.has(id)) {
        continue;
      }
      seen.add(id);
      merged.push(id);
    }
  }
  return merged;
}
