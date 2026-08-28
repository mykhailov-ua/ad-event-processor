import type { components } from './generated/openapi.js';

export type ClickDeliveryMode = 'redirect' | 'proxy';

export type ReviewTrafficAction = 'safe_page' | 'block' | 'passthrough';

export type IngressCostConfigDTO = components['schemas']['IngressCostConfig'];

export type CampaignDTO = components['schemas']['Campaign'];

export type CampaignListResponse = components['schemas']['CampaignListResponse'];

export type CampaignMarginDTO = components['schemas']['CampaignMargin'];

export type CampaignPatchBody = components['schemas']['PatchCampaignRequest'];

/** CPA route audit documents PatchCampaignRequest JSON fields mirrored in OpenAPI. */
export const CAMPAIGN_PATCH_REQUEST_FIELDS = {
  status: 'status',
  budget_limit: 'budget_limit',
  budget_limit_micro: 'budget_limit_micro',
  start_at: 'start_at',
  end_at: 'end_at',
  daypart_hours: 'daypart_hours',
} as const;

export type BuyerCampaignPortfolioRow = {
  id: string;
  name?: string;
  status?: string;
  pacing_mode?: string;
  impressions_7d?: number;
  clicks_7d?: number;
  spend_micro?: number;
  budget_micro?: number;
  utilization_pct?: number;
  pacing_drift_pct?: number | null;
  overspend_risk?: boolean;
  margin_breach?: boolean;
  [key: string]: unknown;
};

export type BuyerPortfolioResponse = {
  active?: number;
  paused?: number;
  archived?: number;
  attention?: Array<{ id: string; name?: string; reason?: string }>;
  impressions_7d?: number;
  clicks_7d?: number;
  overspend_count?: number;
  kpis?: Record<string, unknown> | null;
  recommendations?: unknown[];
  alerts?: unknown[];
  campaigns?: BuyerCampaignPortfolioRow[];
};
