/** Billing / ledger / invoice DTOs (admin API JSON tags). */

export type LedgerEntryDTO = {
  id?: string;
  type?: string;
  amount?: number | string;
  campaign_id?: string;
  created_at?: string;
  currency?: string;
  [key: string]: unknown;
};

export type LedgerListResponse = {
  items?: LedgerEntryDTO[];
  total?: number;
};

export type InvoiceLineDTO = {
  ledger_type?: string;
  amount_micro?: number;
  entry_count?: number;
  description?: string;
  quantity?: number;
  unit_micro?: number;
  total_micro?: number;
  [key: string]: unknown;
};

export type InvoiceDTO = {
  id?: string;
  customer_id?: string;
  billing_month?: string;
  status?: string;
  total_micro?: number;
  tax_micro?: number;
  currency?: string;
  lines?: InvoiceLineDTO[];
  [key: string]: unknown;
};

export type InvoiceListResponse = {
  items?: InvoiceDTO[];
  invoices?: InvoiceDTO[];
  total?: number;
};

export type WalletBalanceDTO = {
  balance_micro?: number;
  balance?: number | string;
  currency?: string;
  allowed_overdraft_micro?: number;
  low_balance_threshold_micro?: number;
  payment_provider?: string;
  payment_provider_configured?: boolean;
  [key: string]: unknown;
};

/** GET /api/v1/billing/invariant — maps to adminapi.InvariantDTO */
export type BillingInvariantDTO = {
  ok: boolean;
  customer_id?: string;
  /** Wallet balance (wallet_balance_micro in ops copy). */
  balance_micro?: number;
  /** Ledger sum (ledger_balance_micro in ops copy). */
  ledger_sum_micro?: number;
  diff_micro?: number;
  /** Present when fleet scan runs without customer_id (admin only). */
  fleet_scan_limit?: number;
};

/** GET /api/v1/billing/invoices/{id}/deliveries */
export type InvoiceDeliveryDTO = {
  id: string;
  status: string;
  provider: string;
  recipient: string;
  template_id: string;
  error_message?: string;
  retry_count: number;
  created_at: string;
  updated_at: string;
};

export type InvoiceDeliveryListResponse = {
  items?: InvoiceDeliveryDTO[];
};

/** Billing ledger export job — adminapi.JobStatusDTO */
export type BillingExportJobDTO = {
  id: string;
  customer_id: string;
  format: string;
  status: string;
  bytes?: number;
  download_url?: string;
  error?: string;
  created_at: string;
  completed_at?: string;
};

export type BillingExportCreateSpec = {
  customer_id: string;
  from: string;
  to: string;
  format: 'csv' | 'ndjson';
};
