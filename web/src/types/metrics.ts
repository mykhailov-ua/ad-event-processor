export type MetricsFreshness = {
  stale?: boolean;
  ch_lag_seconds?: number;
  lagSeconds?: number;
};

export type MetricsBlockDTO = {
  spend_micro?: number;
  revenue_micro?: number;
  profit_micro?: number;
  conversions?: number;
  cpa_micro?: number;
  roi_pct?: number;
  freshness?: MetricsFreshness;
};
