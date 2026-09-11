import type { AdOpsDashboard, DataFreshness, RoleDashboard } from '@/api/types';

export type DashboardFreshness = DataFreshness & {
  freshness_label?: string;
};

export type DashboardMetricsBlock = {
  spend_micro?: number;
  revenue_micro?: number;
  profit_micro?: number;
  conversions?: number;
  unique_clicks?: number;
  roi_pct?: number;
  freshness?: DashboardFreshness;
};

export type DashboardAttentionRow = {
  id: string;
  name: string;
  reason: string;
};

export type DashboardBreakdownRow = {
  id?: string;
  name?: string;
  clicks?: number;
  conversions?: number;
  revenue_micro?: number;
  profit_micro?: number;
  roi_pct?: number;
};

export type DashboardBreakdownTable = {
  rows?: DashboardBreakdownRow[];
  truncated?: boolean;
  total?: number;
};

export type BuyerDashboardPayload = RoleDashboard & {
  customer_id?: string;
  period?: { from?: string; to?: string };
  kpis?: DashboardMetricsBlock;
  active?: number;
  paused?: number;
  archived?: number;
  impressions_7d?: number;
  clicks_7d?: number;
  unique_clicks_7d?: number;
  overspend_count?: number;
  attention?: DashboardAttentionRow[];
  breakdowns?: {
    campaigns?: DashboardBreakdownTable;
  };
};

export type AdopsDashboardCampaignRow = NonNullable<AdOpsDashboard['campaigns']>[number];

export type AdopsDashboardTableMeta = NonNullable<
  NonNullable<AdOpsDashboard['table_sections_meta']>[string]
>;

export type AdopsDashboardPayload = AdOpsDashboard;

export const DASHBOARD_BREAKDOWN_UI_CAP = 10;
export const DASHBOARD_ATTENTION_UI_CAP = 8;
export const DASHBOARD_WORST_SOURCES_UI_CAP = 5;
