export type DashboardRangePreset =
  | 'today'
  | 'yesterday'
  | '7d'
  | '30d'
  | 'this_month'
  | 'custom';

export type DashboardPeriod = {
  from?: string;
  to?: string;
  timezone?: string;
};

export type DashboardFreshness = {
  stale?: boolean;
  label?: string;
};

export type DashboardKpis = {
  spend_micro?: number;
  cost_micro?: number;
  revenue_micro?: number;
  profit_micro?: number;
  conversions?: number;
  unique_clicks?: number;
  cpa_micro?: number;
  cpc_micro?: number;
  epc_micro?: number;
  cr_pct?: number;
  roi_pct?: number;
  freshness?: DashboardFreshness;
};

export type DashboardSeriesPoint = {
  label?: string;
  impressions?: number;
  clicks?: number;
  conversions?: number;
  blocks?: number;
  spend_micros?: number;
  spend_micro?: number;
  revenue_micro?: number;
  profit_micro?: number;
};

export type DashboardBreakdownRow = {
  id?: string;
  name?: string;
  clicks?: number;
  unique_clicks?: number;
  impressions?: number;
  conversions?: number;
  cost_micro?: number;
  revenue_micro?: number;
  profit_micro?: number;
  cpc_micro?: number;
  cpa_micro?: number;
  epc_micro?: number;
  cr_pct?: number;
  roi_pct?: number;
};

export type DashboardBreakdownTotals = {
  clicks?: number;
  unique_clicks?: number;
  impressions?: number;
  conversions?: number;
  cost_micro?: number;
  revenue_micro?: number;
  profit_micro?: number;
  cpc_micro?: number;
  cpa_micro?: number;
  epc_micro?: number;
  cr_pct?: number;
  roi_pct?: number;
};

export type DashboardBreakdownTable = {
  rows?: DashboardBreakdownRow[];
  totals?: DashboardBreakdownTotals;
  truncated?: boolean;
  total?: number;
};

export type DashboardBreakdowns = {
  campaigns?: DashboardBreakdownTable;
  sources?: DashboardBreakdownTable;
  landers?: DashboardBreakdownTable;
  offers?: DashboardBreakdownTable;
};

export type ClickLogEvent = {
  event_type?: string;
  click_id?: string;
  campaign_id?: string;
  placement_id?: string;
  created_at?: string;
  attributed_cost_micro?: number;
  cost_source?: string;
  revenue_micro?: number;
  inbound_status?: string;
  goal_name?: string;
  sub1?: string;
  country?: string;
};

export type BuyerCampaignPortfolioRow = {
  id?: string;
  name?: string;
  status?: string;
  impressions_7d?: number;
  clicks_7d?: number;
  spend_micro?: number;
  budget_micro?: number;
  utilization_pct?: number;
  overspend_risk?: boolean;
  margin_breach?: boolean;
};

export type BuyerAttentionRow = {
  id?: string;
  name?: string;
  reason?: string;
};

export type DashboardAlertCard = {
  id?: string;
  level?: string;
  title?: string;
  detail?: string;
  route?: string;
};

export type DashboardRecommendationCard = {
  id?: string;
  title?: string;
  detail?: string;
  confidence?: number;
  campaign_id?: string;
};

export type CustomerFraudOverview = {
  total_events?: number;
  block_rate_display?: string;
  freshness?: DashboardFreshness;
};

export type BuyerPortfolio = {
  customer_id?: string;
  period?: DashboardPeriod;
  active?: number;
  paused?: number;
  archived?: number;
  impressions_7d?: number;
  clicks_7d?: number;
  unique_clicks_7d?: number;
  overspend_count?: number;
  kpis?: DashboardKpis;
  series?: DashboardSeriesPoint[];
  breakdowns?: DashboardBreakdowns;
  recent_clicks?: ClickLogEvent[];
  campaigns?: BuyerCampaignPortfolioRow[];
  attention?: BuyerAttentionRow[];
  alerts?: DashboardAlertCard[];
  recommendations?: DashboardRecommendationCard[];
  fraud?: CustomerFraudOverview;
};

export function parseBuyerPortfolio(payload: Record<string, unknown> | undefined): BuyerPortfolio | undefined {
  if (!payload || typeof payload !== 'object') {
    return undefined;
  }
  if (
    !('customer_id' in payload) &&
    !('campaigns' in payload) &&
    !('impressions_7d' in payload) &&
    !('breakdowns' in payload)
  ) {
    return undefined;
  }
  return payload as BuyerPortfolio;
}
