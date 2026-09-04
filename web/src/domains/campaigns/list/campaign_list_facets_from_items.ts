import type { CampaignListFacetsResponse, CampaignListFacetOwner } from '@/api/campaigns_api';
import type { Campaign } from '@/api/types';

export function campaignListFacetsFromItems(items: readonly Campaign[]): CampaignListFacetsResponse {
  const countries = new Set<string>();
  const owners = new Map<string, CampaignListFacetOwner>();

  for (const row of items) {
    for (const code of row.target_countries ?? []) {
      const trimmed = code?.trim();
      if (trimmed) {
        countries.add(trimmed);
      }
    }
    const ownerId = row.owner_user_id?.trim();
    if (!ownerId || owners.has(ownerId)) {
      continue;
    }
    owners.set(ownerId, { user_id: ownerId });
  }

  return {
    countries: [...countries].sort(),
    owners: [...owners.values()].sort((left, right) => left.user_id.localeCompare(right.user_id)),
  };
}
