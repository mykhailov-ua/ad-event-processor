import type { CampaignListQuery } from '@/api/types';

export type CampaignListSelectionScope = {
  query: CampaignListQuery;
  statsFrom?: string;
  statsTo?: string;
};

export function campaignListSelectionScopeKey(scope: CampaignListSelectionScope): string {
  const { query, statsFrom, statsTo } = scope;
  return [
    query.customer_id ?? '',
    query.status ?? '',
    query.q ?? '',
    query.pacing_mode ?? '',
    query.owner_user_id ?? '',
    query.country ?? '',
    query.budget_min_micro ?? '',
    query.budget_max_micro ?? '',
    query.limit ?? '',
    query.offset ?? '',
    query.sort ?? '',
    query.order ?? '',
    query.from ?? '',
    query.to ?? '',
    statsFrom ?? '',
    statsTo ?? '',
  ].join('|');
}
