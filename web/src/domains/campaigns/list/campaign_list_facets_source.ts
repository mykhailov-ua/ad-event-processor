import type { CampaignListFacetsResponse } from '@/api/campaigns_api';

export const CAMPAIGN_LIST_FACETS_DEGRADED_MESSAGE =
  'GET /api/v1/campaigns/list-facets is not available on this control plane build. Owner and country filters stay disabled until the endpoint is wired.';

export function isCampaignListFacetsDegraded(
  listFacetsFromApi: CampaignListFacetsResponse | undefined,
  listFacetsFetching: boolean,
): boolean {
  return !listFacetsFetching && listFacetsFromApi === undefined;
}

export function resolveCampaignListFacets(
  listFacetsFromApi: CampaignListFacetsResponse | undefined,
  listFacetsFetching: boolean,
): {
  facets: CampaignListFacetsResponse | undefined;
  degraded: boolean;
} {
  const degraded = isCampaignListFacetsDegraded(listFacetsFromApi, listFacetsFetching);
  return {
    facets: listFacetsFromApi,
    degraded,
  };
}
