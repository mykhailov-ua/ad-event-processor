import type { CampaignListFacetsResponse } from '@/api/campaigns_api';

// List-facets policy: server GET /campaigns/list-facets is the only production source.
// When the endpoint is missing, filters stay disabled (degraded banner); never synthesize from page rows.
export const CAMPAIGN_LIST_FACETS_DEGRADED_MESSAGE =
  'GET /api/v1/campaigns/list-facets is not available on this control plane build. Owner and country filters stay disabled until the endpoint is wired.';

export function isCampaignListFacetsDegraded(
  listFacetsFromApi: CampaignListFacetsResponse | undefined,
  listFacetsFetching: boolean,
  listFacetsError: Error | undefined
): boolean {
  if (listFacetsFetching || listFacetsError !== undefined) {
    return false;
  }
  return listFacetsFromApi === undefined;
}

export function resolveCampaignListFacets(
  listFacetsFromApi: CampaignListFacetsResponse | undefined,
  listFacetsFetching: boolean,
  listFacetsError: Error | undefined
): {
  facets: CampaignListFacetsResponse | undefined;
  degraded: boolean;
} {
  const degraded = isCampaignListFacetsDegraded(
    listFacetsFromApi,
    listFacetsFetching,
    listFacetsError
  );
  return {
    facets: listFacetsFromApi,
    degraded,
  };
}
