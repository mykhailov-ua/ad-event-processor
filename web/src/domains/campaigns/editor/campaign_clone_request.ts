import type { CloneCampaignOptions, CloneCampaignRequest } from '@/api/campaigns_api';

export const DEFAULT_CLONE_OPTIONS: Required<CloneCampaignOptions> = {
  include_flow: true,
  include_postbacks: true,
  include_fraud: true,
  include_placement_blocks: true,
  reset_spend: false,
};

export function buildCloneRequestBody(
  nameSuffix: string,
  options: CloneCampaignOptions,
): CloneCampaignRequest {
  const body: CloneCampaignRequest = { options };
  const suffix = nameSuffix.trim();
  if (suffix !== '') {
    body.name_suffix = suffix;
  }
  return body;
}
