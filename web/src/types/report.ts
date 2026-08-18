export type DataFreshness = {
  as_of: string;
  consistency: string;
  stale: boolean;
  ch_lag_seconds?: number;
};

export type ReportEnvelope<TRow = ReportRow> = {
  rows: TRow[];
  freshness: DataFreshness;
  next_cursor?: string;
};

export type ReportCompareDeltas = {
  spend_micro_delta: number;
  revenue_micro_delta: number;
  impressions_delta: number;
  clicks_delta: number;
  conversions_delta: number;
};

export type ReportRow = {
  [key: string]: unknown;
};

export type PlacementReportRow = {
  placement_id: string;
  campaign_id: string;
  impressions: number;
  clicks: number;
  conversions: number;
  spend_micro: number;
  revenue_micro: number;
  profit_micro: number;
  roi_pct: number;
  cpa_micro: number;
  ctr?: number;
  ivt_rate?: number;
  compare?: ReportCompareDeltas;
};

export type KeywordReportRow = {
  keyword: string;
  campaign_id: string;
  impressions: number;
  clicks: number;
  conversions: number;
  spend_micro: number;
  revenue_micro: number;
  profit_micro: number;
  roi_pct: number;
  cpa_micro?: number;
  ctr?: number;
  ivt_rate?: number;
  compare?: ReportCompareDeltas;
};

export type TrueRoiRow = {
  campaign_id: string;
  ad_spend_micro: number;
  revenue_micro: number;
  true_profit_micro: number;
  true_roi_pct: number;
  true_cpa_micro: number;
  conversions: number;
};
