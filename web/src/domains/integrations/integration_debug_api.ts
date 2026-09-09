import { runCampaignSmoke, validateCampaignFlow } from '@/api/campaigns_api';
import { testPostbackConfig } from '@/api/integrations_api';
import type {
  CampaignFlowValidateResponse,
  CampaignSmokeResult,
  PostbackDryRunResult,
} from '@/api/types';

export function buildIntegrationsDebuggerHref(campaignId: string): string {
  return `/integrations/debugger?campaign_id=${encodeURIComponent(campaignId)}`;
}

export async function runIntegrationSmokeTest(
  campaignId: string,
  signal?: AbortSignal
): Promise<CampaignSmokeResult> {
  return runCampaignSmoke(campaignId, signal);
}

export async function validateIntegrationFlow(
  campaignId: string,
  signal?: AbortSignal
): Promise<CampaignFlowValidateResponse> {
  return validateCampaignFlow(campaignId, {}, signal);
}

export async function runPostbackDryRun(
  campaignId: string,
  signal?: AbortSignal
): Promise<PostbackDryRunResult> {
  return testPostbackConfig(campaignId, signal);
}
