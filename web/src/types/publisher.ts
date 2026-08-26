export type PublisherKPIs = {
  impressions: number;
  fill_rate: number;
  ecpm_micro: number;
  ivt_rate: number;
};

export type PublisherPlacement = {
  placement_id: string;
  impressions: number;
  clicks: number;
  fill_rate: number;
  revenue_micro: number;
  ecpm_micro: number;
};

export type PublisherDashboard = {
  seller_id: string;
  publisher_account_id?: string;
  from: string;
  to: string;
  kpis: PublisherKPIs;
  placements: PublisherPlacement[];
};

export type PublisherStatement = {
  id: number;
  amount_micro: number;
  created_at: string;
  campaign_id?: string;
  idempotency_hash?: string;
};

export type PublisherStatementList = {
  items: PublisherStatement[];
  total: number;
};

export type SupplyValidation = import('../helpers/supply_api.js').SupplyValidation;
