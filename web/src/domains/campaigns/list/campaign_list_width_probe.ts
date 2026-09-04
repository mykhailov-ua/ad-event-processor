import type { CampaignListQuery } from '@/api/types';
import { OPTIMAL_LIST_LIMIT_MAX } from '@/lib/list_query';
import { isUuidLike } from '@/lib/customer_label';

export type CampaignListWidthProbeListSnapshot = {
  items: readonly { id: string }[];
  total: number;
};

// Width probe list: same filters, name sort, first OPTIMAL_LIST_LIMIT_MAX rows.
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

// True when the page list already includes every filtered row (unique ids, total <= cap).
export function listResponseCoversWidthProbeDataset(
  data: CampaignListWidthProbeListSnapshot | undefined,
): boolean {
  if (!data || data.total === 0) {
    return false;
  }
  if (data.total > OPTIMAL_LIST_LIMIT_MAX) {
    return false;
  }
  if (data.items.length !== data.total) {
    return false;
  }
  const ids = new Set<string>();
  for (const item of data.items) {
    if (!item.id || ids.has(item.id)) {
      return false;
    }
    ids.add(item.id);
  }
  return true;
}

// Deduped ids for one metrics batch (page rows plus optional width-probe rows).
export function mergeCampaignIdsForMetricsBatch(
  ...idLists: readonly (readonly string[])[]
): string[] {
  const seen = new Set<string>();
  const merged: string[] = [];
  for (const ids of idLists) {
    for (const id of ids) {
      if (!id || !isUuidLike(id) || seen.has(id)) {
        continue;
      }
      seen.add(id);
      merged.push(id);
    }
  }
  return merged;
}
