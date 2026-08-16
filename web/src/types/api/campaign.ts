/**
 * Go: adminapi.CampaignDTO — list item and GET /campaigns/{id}.
 * Money fields are decimal strings (not micro-int).
 */
export type CampaignDTO = {
  id: string;
  name: string;
  status: string;
  budget_limit: string;
  current_spend: string;
  customer_id: string;
  pacing_mode: string;
  daily_budget: string;
  timezone: string;
  freq_limit: number;
  freq_window: number;
  target_countries: string[];
  target_url?: string;
  brand_id?: string;
  safe_page_url?: string;
  safe_page_enabled?: boolean;
  creative_payload?: unknown;
  referrer_filter?: string;
  start_at?: string;
  end_at?: string;
  daypart_hours: number[];
  flow_id?: string;
  created_at: string;
  updated_at: string;
  /** Present when list API embeds margin guard summary. */
  margin_breach?: boolean;
};

/** GET /api/v1/campaigns list envelope */
export type CampaignListResponse = {
  items: CampaignDTO[];
  total: number;
};

/** Go: adminapi.CampaignMarginDTO — GET /campaigns/{id}/margin */
export type CampaignMarginDTO = {
  campaign_id: string;
  window_start: string;
  window_hours: number;
  advertiser_spend_micro: number;
  rtb_cost_micro: number;
  operator_margin_micro: number;
  publisher_payout_micro: number;
  cost_over_revenue_limit: number;
  threshold_bps: number;
  margin_breach: boolean;
};

/** PATCH /api/v1/campaigns/{id} body (partial). */
export type CampaignPatchBody = Partial<{
  name: string;
  status: string;
  budget_limit: string;
  daily_budget: string;
  daily_budget_micro: number;
  pacing_mode: string;
  timezone: string;
  freq_limit: number;
  freq_window: number;
  target_countries: string[];
  target_url: string;
  safe_page_url: string;
  safe_page_enabled: boolean;
  referrer_filter: string;
  start_at: string;
  end_at: string;
  daypart_hours: number[];
  flow_id?: string | null;
}>;

/**
 * Buyer portfolio campaign row from dashboards (not CampaignDTO).
 * Includes impressions_7d / pacing_drift_pct for health UI.
 */
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

/** Buyer dashboard / portfolio payload (subset used by UI). */
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
