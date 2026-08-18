export type ClickDeliveryMode = 'redirect' | 'proxy';

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
  attestation_enabled?: boolean;
  attestation_ttl_sec?: number;
  dmr_enabled?: boolean;
  l1_cidr_block_enabled?: boolean;
  l15_proxy_vpn_block_enabled?: boolean;
  tls_fingerprint_block_enabled?: boolean;
  conn_type_policy?: string;
  link_signing_enabled?: boolean;
  link_signing_ttl_sec?: number;
  click_delivery?: ClickDeliveryMode | string;
  proxy_upstream_url?: string;
  proxy_rewrite_assets?: boolean;
  creative_payload?: unknown;
  referrer_filter?: string;
  start_at?: string;
  end_at?: string;
  daypart_hours: number[];
  flow_id?: string;
  owner_user_id?: string;
  created_at: string;
  updated_at: string;

  margin_breach?: boolean;
};

export type CampaignListResponse = {
  items: CampaignDTO[];
  total: number;
};

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

export type CampaignPatchBody = Partial<{
  name: string;
  status: string;
  budget_limit: string;
  budget_limit_micro: number;
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
  attestation_enabled: boolean;
  attestation_ttl_sec: number;
  dmr_enabled: boolean;
  l1_cidr_block_enabled: boolean;
  l15_proxy_vpn_block_enabled: boolean;
  tls_fingerprint_block_enabled: boolean;
  conn_type_policy: string;
  link_signing_enabled: boolean;
  link_signing_ttl_sec: number;
  click_delivery: ClickDeliveryMode;
  proxy_upstream_url: string;
  proxy_rewrite_assets: boolean;
  referrer_filter: string;
  start_at: string;
  end_at: string;
  daypart_hours: number[];
  flow_id?: string | null;
  brand_id?: string | null;
}>;

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
