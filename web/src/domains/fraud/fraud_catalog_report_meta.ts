import type { FraudCatalogReportKey } from '@/domains/fraud/fraud_catalog_report_types';

export type FraudCatalogColumnKind =
  | 'text'
  | 'mono'
  | 'count'
  | 'ratio'
  | 'percent'
  | 'badge';

export type FraudCatalogColumnDef = {
  id: string;
  label: string;
  field: string;
  align?: 'right';
  kind: FraudCatalogColumnKind;
  displayField?: string;
};

export type FraudCatalogReportMeta = {
  key: FraudCatalogReportKey;
  title: string;
  description: string;
  requiresCustomer: boolean;
  showDimensionFilter: boolean;
  showCompareFilter: boolean;
  showSliceFilter: boolean;
  showMinDesyncFilter?: boolean;
  columns: FraudCatalogColumnDef[];
  seriesColumns?: FraudCatalogColumnDef[];
};

const SILENT_REJECT_FUNNEL_META: FraudCatalogReportMeta = {
  key: 'silent-reject-impression-funnel',
  title: 'Non-blocking response funnel',
  description: 'Billable vs non-blocking fraud response vs IVT impressions.',
  requiresCustomer: true,
  showDimensionFilter: false,
  showCompareFilter: false,
  showSliceFilter: false,
  columns: [
    { id: 'campaign', label: 'Campaign', field: 'campaign_id', kind: 'mono' },
    { id: 'placement', label: 'Placement', field: 'placement_id', kind: 'mono' },
    { id: 'billable', label: 'Billable', field: 'billable_impressions', kind: 'count', align: 'right' },
    {
      id: 'silent',
      label: 'Non-blocking',
      field: 'silent_reject_impressions',
      kind: 'count',
      align: 'right',
    },
    { id: 'ivt', label: 'IVT', field: 'ivt_impressions', kind: 'count', align: 'right' },
    {
      id: 'silent_rate',
      label: 'Non-blocking rate',
      field: 'silent_reject_rate',
      kind: 'ratio',
      align: 'right',
    },
    {
      id: 'ivt_rate',
      label: 'IVT rate',
      field: 'ivt_impression_rate',
      kind: 'ratio',
      align: 'right',
    },
  ],
};

const SIGNAL_EFFECTIVENESS_META: FraudCatalogReportMeta = {
  key: 'signal-effectiveness',
  title: 'Signal effectiveness',
  description: 'Wire signal block and non-blocking response rates.',
  requiresCustomer: true,
  showDimensionFilter: false,
  showCompareFilter: false,
  showSliceFilter: false,
  columns: [
    { id: 'signal', label: 'Signal', field: 'signal_code', kind: 'mono' },
    { id: 'category', label: 'Category', field: 'fraud_category_label', kind: 'text', displayField: 'fraud_category' },
    { id: 'volume', label: 'Events', field: 'event_volume', kind: 'count', align: 'right' },
    {
      id: 'block',
      label: 'Block rate',
      field: 'block_rate',
      kind: 'percent',
      align: 'right',
      displayField: 'block_rate_display',
    },
    {
      id: 'silent',
      label: 'Non-blocking rate',
      field: 'silent_reject_rate',
      kind: 'percent',
      align: 'right',
      displayField: 'silent_reject_rate_display',
    },
    { id: 'tier', label: 'Weight tier', field: 'suggested_weight_tier', kind: 'badge' },
  ],
};

const CUSTOMER_FRAUD_BY_DIMENSION_META: FraudCatalogReportMeta = {
  key: 'customer-fraud-by-dimension',
  title: 'Fraud by dimension',
  description: 'Fraud concentration by placement, geo, or sub.',
  requiresCustomer: true,
  showDimensionFilter: true,
  showCompareFilter: true,
  showSliceFilter: false,
  columns: [
    { id: 'dimension', label: 'Dimension', field: 'dimension_value', kind: 'text' },
    { id: 'campaign', label: 'Campaign', field: 'campaign_id', kind: 'mono' },
    { id: 'impressions', label: 'Impressions', field: 'impressions', kind: 'count', align: 'right' },
    { id: 'clicks', label: 'Clicks', field: 'clicks', kind: 'count', align: 'right' },
    { id: 'ivt', label: 'IVT events', field: 'ivt_events', kind: 'count', align: 'right' },
    { id: 'blocked', label: 'Blocked', field: 'blocked_events', kind: 'count', align: 'right' },
    {
      id: 'ivt_rate',
      label: 'IVT rate',
      field: 'ivt_rate',
      kind: 'percent',
      align: 'right',
      displayField: 'ivt_rate_label',
    },
    {
      id: 'top_category',
      label: 'Top category',
      field: 'top_fraud_category_label',
      kind: 'text',
      displayField: 'top_fraud_category',
    },
    { id: 'delta', label: 'Delta', field: 'delta_label', kind: 'badge' },
  ],
};

const IVT_BY_SOURCE_META: FraudCatalogReportMeta = {
  key: 'ivt-by-source',
  title: 'IVT by source',
  description: 'Invalid traffic by sub and geo.',
  requiresCustomer: true,
  showDimensionFilter: false,
  showCompareFilter: false,
  showSliceFilter: false,
  columns: [
    { id: 'campaign', label: 'Campaign', field: 'campaign_id', kind: 'mono' },
    { id: 'sub1', label: 'Sub1', field: 'sub1', kind: 'text' },
    { id: 'sub2', label: 'Sub2', field: 'sub2', kind: 'text' },
    { id: 'country', label: 'Country', field: 'country', kind: 'text' },
    { id: 'impressions', label: 'Impressions', field: 'impressions', kind: 'count', align: 'right' },
    { id: 'clicks', label: 'Clicks', field: 'clicks', kind: 'count', align: 'right' },
    { id: 'ivt', label: 'IVT events', field: 'ivt_events', kind: 'count', align: 'right' },
    { id: 'ivt_rate', label: 'IVT rate', field: 'ivt_rate', kind: 'ratio', align: 'right' },
  ],
};

const LAYER_DESYNC_SUMMARY_META: FraudCatalogReportMeta = {
  key: 'layer-desync-summary',
  title: 'Layer desync summary',
  description: 'Cross-layer desync fraud counts by campaign.',
  requiresCustomer: true,
  showDimensionFilter: false,
  showCompareFilter: false,
  showSliceFilter: false,
  columns: [
    { id: 'campaign', label: 'Campaign', field: 'campaign_id', kind: 'mono' },
    { id: 'desync', label: 'Desync layers', field: 'layer_desync_count', kind: 'count', align: 'right' },
    { id: 'events', label: 'Events', field: 'event_count', kind: 'count', align: 'right' },
    {
      id: 'silent',
      label: 'Non-blocking',
      field: 'silent_reject_count',
      kind: 'count',
      align: 'right',
    },
  ],
};

const LAYER_DESYNC_DRILLDOWN_META: FraudCatalogReportMeta = {
  key: 'layer-desync-drilldown',
  title: 'Layer desync drilldown',
  description: 'Fraud reasons and hourly trend for cross-layer desync events.',
  requiresCustomer: true,
  showDimensionFilter: false,
  showCompareFilter: false,
  showSliceFilter: false,
  showMinDesyncFilter: true,
  columns: [
    { id: 'reason', label: 'Fraud reason', field: 'fraud_reason', kind: 'mono' },
    {
      id: 'category',
      label: 'Category',
      field: 'fraud_category_label',
      kind: 'text',
      displayField: 'fraud_category',
    },
    { id: 'events', label: 'Events', field: 'event_count', kind: 'count', align: 'right' },
    {
      id: 'silent',
      label: 'Non-blocking',
      field: 'silent_reject_count',
      kind: 'count',
      align: 'right',
    },
  ],
  seriesColumns: [
    { id: 'hour', label: 'Hour', field: 'label', kind: 'mono' },
    { id: 'events', label: 'Events', field: 'event_count', kind: 'count', align: 'right' },
    {
      id: 'silent',
      label: 'Non-blocking',
      field: 'silent_reject_count',
      kind: 'count',
      align: 'right',
    },
  ],
};

const RTT_SPLIT_TUNNEL_META: FraudCatalogReportMeta = {
  key: 'rtt-split-tunnel',
  title: 'RTT split tunnel',
  description: 'RTT split-tunnel distribution by campaign and country.',
  requiresCustomer: true,
  showDimensionFilter: false,
  showCompareFilter: false,
  showSliceFilter: false,
  columns: [
    { id: 'campaign', label: 'Campaign', field: 'campaign_id', kind: 'mono' },
    { id: 'country', label: 'Country', field: 'country', kind: 'text' },
    { id: 'events', label: 'Events', field: 'event_count', kind: 'count', align: 'right' },
    {
      id: 'split',
      label: 'Split tunnel',
      field: 'split_tunnel_count',
      kind: 'count',
      align: 'right',
    },
    {
      id: 'share',
      label: 'Share',
      field: 'share_label',
      kind: 'percent',
      align: 'right',
      displayField: 'split_tunnel_share',
    },
    {
      id: 'coverage',
      label: 'Coverage',
      field: 'coverage_label',
      kind: 'percent',
      align: 'right',
      displayField: 'coverage_pct',
    },
  ],
};

const FILTER_REJECTS_META: FraudCatalogReportMeta = {
  key: 'filter-rejects',
  title: 'Filter rejects',
  description: 'Unified filter reject kinds.',
  requiresCustomer: false,
  showDimensionFilter: false,
  showCompareFilter: false,
  showSliceFilter: true,
  columns: [
    { id: 'kind', label: 'Reject kind', field: 'reject_kind', kind: 'mono' },
    { id: 'count', label: 'Count', field: 'reject_count', kind: 'count', align: 'right' },
    { id: 'country', label: 'Country', field: 'country', kind: 'text' },
    { id: 'placement', label: 'Placement', field: 'placement_id', kind: 'mono' },
  ],
};

const FRAUD_CATALOG_REPORT_META: Record<FraudCatalogReportKey, FraudCatalogReportMeta> = {
  'silent-reject-impression-funnel': SILENT_REJECT_FUNNEL_META,
  'signal-effectiveness': SIGNAL_EFFECTIVENESS_META,
  'customer-fraud-by-dimension': CUSTOMER_FRAUD_BY_DIMENSION_META,
  'ivt-by-source': IVT_BY_SOURCE_META,
  'layer-desync-summary': LAYER_DESYNC_SUMMARY_META,
  'layer-desync-drilldown': LAYER_DESYNC_DRILLDOWN_META,
  'rtt-split-tunnel': RTT_SPLIT_TUNNEL_META,
  'filter-rejects': FILTER_REJECTS_META,
};

export function getFraudCatalogReportMeta(key: FraudCatalogReportKey): FraudCatalogReportMeta {
  return FRAUD_CATALOG_REPORT_META[key];
}

export function isFraudCatalogReportKey(key: string): key is FraudCatalogReportKey {
  return key in FRAUD_CATALOG_REPORT_META;
}
