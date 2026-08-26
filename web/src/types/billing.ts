import type { components } from './generated/openapi.js';

export type LedgerEntryDTO = components['schemas']['BalanceLedgerEntry'];

export type LedgerListResponse = components['schemas']['CustomerLedgerListResponse'];

export type CustomerBalanceDTO = components['schemas']['CustomerBalance'];

export type InvoiceLineDTO = components['schemas']['BillingInvoiceLine'];

export type InvoiceDTO = components['schemas']['Invoice'];

export type InvoiceListResponse = components['schemas']['InvoiceListResponse'] & {
  invoices?: InvoiceDTO[];
};

export type WalletBalanceDTO = components['schemas']['Wallet'];

export type BillingInvariantDTO = components['schemas']['BillingInvariant'];

export type InvoiceDeliveryDTO = components['schemas']['InvoiceDelivery'];

export type InvoiceDeliveryListResponse = components['schemas']['InvoiceDeliveryListResponse'];

export type BillingExportJobDTO = components['schemas']['BillingExportJob'];

export type BillingExportCreateSpec = components['schemas']['BillingExportJobSpec'];

export type BillingForecastDTO = components['schemas']['BillingForecast'];

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

export type InvoiceLedgerLineDTO = components['schemas']['BillingLedgerLine'];

export type InvoiceLedgerLinesResponse = components['schemas']['InvoiceLedgerLinesResponse'];

export type BillingSummaryDTO = components['schemas']['BillingSummary'];

export type BillingPeriodBounds = components['schemas']['PeriodBounds'];

export type TaxBreakdownDTO = components['schemas']['TaxBreakdown'];

export type ReconciliationDTO = components['schemas']['Reconciliation'];

export type InvoiceSummaryDTO = components['schemas']['InvoiceSummary'];

export type PaymentSummaryDTO = components['schemas']['PaymentSummary'];

export type BillingStatementDTO = components['schemas']['BillingStatement'];

export type InvoicePreviewDTO = components['schemas']['InvoicePreview'];

export type PaymentHistoryRowDTO = components['schemas']['PaymentHistoryRow'];

export type PaymentHistoryListResponse = components['schemas']['PaymentHistoryListResponse'];
