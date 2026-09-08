import type { DataFreshness } from '@/api/types';

export type FraudCatalogReportKey =
  | 'silent-reject-impression-funnel'
  | 'signal-effectiveness'
  | 'customer-fraud-by-dimension'
  | 'ivt-by-source'
  | 'layer-desync-summary'
  | 'filter-rejects';

export type FraudCatalogReportRow = Record<string, unknown>;

export type FraudCatalogReportResponse = {
  rows: FraudCatalogReportRow[];
  freshness?: DataFreshness;
  next_cursor?: string;
  truncated?: boolean;
};

export type FraudCatalogDimension = 'placement' | 'sub1' | 'sub2' | 'country' | 'campaign';

export type FraudCatalogReportQuery = {
  customer_id?: string;
  from?: string;
  to?: string;
  campaign_id?: string;
  limit?: number;
  offset?: number;
  dimension?: FraudCatalogDimension;
  compare?: boolean;
  slice?: boolean;
};
