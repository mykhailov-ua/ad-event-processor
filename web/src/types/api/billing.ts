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

/** GET /api/v1/customers/{id}/billing/forecast */
export type BillingForecastDTO = {
  customer_id?: string;
  month?: string;
  ledger_mtd_micro?: number;
  ledger_run_rate_micro_per_day?: number;
  projected_month_end_micro?: number;
  days_remaining?: number;
  low_confidence?: boolean;
  ch_unavailable?: boolean;
};

/** GET /api/v1/disputes */
export type DisputeRowDTO = {
  intent_id?: string;
  customer_id?: string;
  amount_micro?: number;
  currency?: string;
  provider_dispute_id?: string;
  updated_at?: string;
  chargeback_ledger_entry_ids?: number[];
};

export type DisputeListResponse = {
  disputes?: DisputeRowDTO[];
  total?: number;
};

/** GET /api/v1/billing/invoices/{id}/ledger-lines */
export type InvoiceLedgerLineDTO = {
  id?: number;
  amount_micro?: number;
  ledger_type?: string;
  created_at?: string;
};

export type InvoiceLedgerLinesResponse = {
  items?: InvoiceLedgerLineDTO[];
  total?: number;
  next_cursor?: string;
  limit?: number;
};

/** GET /api/v1/billing/summary — fleet MTD billing ops (admin, shards:read) */
export type BillingSummaryDTO = {
  invoiced_mtd_micro?: number;
  invoice_count_mtd?: number;
  undelivered_invoice_notifications?: number;
  customers_with_spend_in_month?: number;
};

export type BillingPeriodBounds = {
  from?: string;
  to?: string;
};

export type TaxBreakdownDTO = {
  scheme?: string;
  rate_bps?: number;
  tax_micro?: number;
};

export type ReconciliationDTO = {
  invoice_total_micro?: number;
  ledger_sum_micro?: number;
  delta_micro?: number;
};

export type InvoiceSummaryDTO = {
  id?: string;
  customer_id?: string;
  billing_month?: string;
  subtotal_micro?: number;
  tax_micro?: number;
  total_micro?: number;
  status?: string;
  currency?: string;
};

export type PaymentSummaryDTO = {
  ledger_id?: number;
  amount_micro?: number;
  payment_intent_id?: string;
  created_at?: string;
};

/** GET /api/v1/customers/{id}/billing/statement */
export type BillingStatementDTO = {
  customer_id?: string;
  period?: BillingPeriodBounds;
  opening_balance_micro?: number;
  closing_balance_micro?: number;
  lines?: InvoiceLineDTO[];
  invoices?: InvoiceSummaryDTO[];
  payments?: PaymentSummaryDTO[];
  tax_breakdown?: TaxBreakdownDTO;
  reconciliation?: ReconciliationDTO;
  currency?: string;
};

/** POST /api/v1/billing/invoices/preview */
export type InvoicePreviewDTO = {
  customer_id?: string;
  billing_month?: string;
  currency?: string;
  subtotal_micro?: number;
  tax_micro?: number;
  total_micro?: number;
  tax_scheme?: string;
  tax_rate_bps?: number;
  lines?: InvoiceLineDTO[];
  would_skip?: boolean;
  ledger_sum_micro?: number;
};

/** GET /api/v1/customers/{id}/payments */
export type PaymentHistoryRowDTO = {
  intent_id?: string;
  customer_id?: string;
  amount_micro?: number;
  currency?: string;
  status?: string;
  provider?: string;
  provider_ref?: string;
  idempotency_key?: string;
  ledger_entry_id?: string;
  created_at?: string;
  updated_at?: string;
};

export type PaymentHistoryListResponse = {
  items?: PaymentHistoryRowDTO[];
  total?: number;
  limit?: number;
  offset?: number;
};
