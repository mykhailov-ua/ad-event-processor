import type { CloneCampaignOptions, CloneCampaignRequest } from '@/api/campaigns_api';
import { ApiError } from '@/api/client';

export const DEFAULT_CLONE_OPTIONS: Required<CloneCampaignOptions> = {
  include_flow: true,
  include_postbacks: true,
  include_fraud: true,
  include_placement_blocks: true,
  reset_spend: false,
};

export function buildCloneRequestBody(
  nameSuffix: string,
  options: CloneCampaignOptions
): CloneCampaignRequest {
  const body: CloneCampaignRequest = { options };
  const suffix = nameSuffix.trim();
  if (suffix !== '') {
    body.name_suffix = suffix;
  }
  return body;
}

export function cloneRequestError(err: unknown): Error {
  if (err instanceof ApiError) {
    return err;
  }
  if (err instanceof Error) {
    return err;
  }
  return new Error(String(err));
}

export function cloneMutationErrorMessage(error: Error): string {
  if (error instanceof ApiError && error.message.toLowerCase().includes('insufficient balance')) {
    return 'Customer balance is too low to reserve this campaign budget. Reduce budget_limit or increase customer balance, then retry clone.';
  }
  return error.message;
}
