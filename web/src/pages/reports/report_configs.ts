export type ReportColumnFormat = 'money' | 'pct' | 'rate' | 'number' | 'text';

export type SimpleReportColumn = {
  key: string;
  label: string;
  format?: ReportColumnFormat;
};

export type SimpleReportConfig = {
  path: string;
  title: string;
  endpoint: string;
  columns: SimpleReportColumn[];
};

export const SIMPLE_REPORT_CONFIGS: SimpleReportConfig[] = [
  {
    path: '/reports/spend-velocity',
    title: 'Spend velocity',
    endpoint: 'spend-velocity',
    columns: [
      { key: 'bucket', label: 'Hour' },
      { key: 'spend_micro', label: 'Spend', format: 'money' },
      { key: 'clicks', label: 'Clicks', format: 'number' },
    ],
  },
  {
    path: '/reports/daypart-heatmap',
    title: 'Daypart heatmap',
    endpoint: 'daypart-heatmap',
    columns: [
      { key: 'hour', label: 'Hour (UTC)' },
      { key: 'clicks', label: 'Clicks', format: 'number' },
    ],
  },
  {
    path: '/reports/campaign-geo-device',
    title: 'Geo & device',
    endpoint: 'campaign-geo-device',
    columns: [
      { key: 'country', label: 'Country' },
      { key: 'device', label: 'Device' },
      { key: 'clicks', label: 'Clicks', format: 'number' },
    ],
  },
  {
    path: '/reports/source-quality',
    title: 'Source quality',
    endpoint: 'source-quality',
    columns: [
      { key: 'placement_id', label: 'Placement' },
      { key: 'campaign_id', label: 'Campaign' },
      { key: 'clicks', label: 'Clicks', format: 'number' },
      { key: 'conversions', label: 'Conv.', format: 'number' },
      { key: 'ivt_rate', label: 'IVT %', format: 'rate' },
      { key: 'roi_pct', label: 'ROI %', format: 'pct' },
    ],
  },
  {
    path: '/reports/discrepancy-buy-sell',
    title: 'Buy/sell discrepancy',
    endpoint: 'discrepancy-buy-sell',
    columns: [
      { key: 'campaign_id', label: 'Campaign' },
      { key: 'buy_spend_micro', label: 'Buy spend', format: 'money' },
      { key: 'sell_rev_micro', label: 'Sell revenue', format: 'money' },
      { key: 'delta_micro', label: 'Delta', format: 'money' },
      { key: 'delta_pct', label: 'Delta %', format: 'pct' },
    ],
  },
  {
    path: '/reports/true-roi',
    title: 'True ROI',
    endpoint: 'true-roi',
    columns: [
      { key: 'campaign_id', label: 'Campaign' },
      { key: 'ad_spend_micro', label: 'Ad Spend', format: 'money' },
      { key: 'revenue_micro', label: 'Revenue', format: 'money' },
      { key: 'true_profit_micro', label: 'True Profit', format: 'money' },
      { key: 'true_roi_pct', label: 'True ROI %', format: 'pct' },
      { key: 'true_cpa_micro', label: 'True CPA', format: 'money' },
      { key: 'conversions', label: 'Conv.', format: 'number' },
    ],
  },
  {
    path: '/reports/campaign-overview',
    title: 'Campaign overview',
    endpoint: 'campaign-overview',
    columns: [
      { key: 'name', label: 'Campaign' },
      { key: 'status', label: 'Status' },
      { key: 'impressions_7d', label: 'Impr. 7d', format: 'number' },
      { key: 'clicks_7d', label: 'Clicks 7d', format: 'number' },
      { key: 'utilization_pct', label: 'Budget %', format: 'pct' },
      { key: 'pacing_drift_pct', label: 'Drift %', format: 'pct' },
      { key: 'overspend_risk', label: 'Risk' },
    ],
  },
  {
    path: '/reports/customer-portfolio',
    title: 'Customer portfolio',
    endpoint: 'customer-portfolio',
    columns: [
      { key: 'row_type', label: 'Type' },
      { key: 'campaign_id', label: 'Campaign ID' },
      { key: 'name', label: 'Name' },
      { key: 'status', label: 'Status' },
      { key: 'active', label: 'Active', format: 'number' },
      { key: 'paused', label: 'Paused', format: 'number' },
      { key: 'impressions_7d', label: 'Impr. 7d', format: 'number' },
      { key: 'clicks_7d', label: 'Clicks 7d', format: 'number' },
      { key: 'utilization_pct', label: 'Budget %', format: 'pct' },
      { key: 'pacing_drift_pct', label: 'Drift %', format: 'pct' },
      { key: 'overspend_risk', label: 'Risk' },
    ],
  },
];
