import { Link } from 'react-router-dom';

import type { BillingInvariant, BillingSummary, Invoice, InvoicePreview } from '@/api/types';
import { BillingInvoices } from '@/domains/billing/billing_invoices';
import type { InvoiceStatusFilter } from '@/domains/billing/billing_invoices';
import { BillingSummarySection } from '@/domains/billing/billing_summary';
import { BillingInvariantPanel, BillingPreviewPanel } from '@/domains/billing/billing_tools';
import { PageChrome } from '@/shell/page_chrome';

export type BillingOverviewProps = {
  summary: BillingSummary | undefined;
  summaryFetching: boolean;
  summaryError: Error | undefined;
  hasSummarySnapshot: boolean;
  invoices: Invoice[] | undefined;
  invoicesTotal: number;
  invoicesLimit: number;
  invoicesOffset: number;
  invoicesFetching: boolean;
  invoicesError: Error | undefined;
  hasInvoicesSnapshot: boolean;
  draftMonth: string;
  draftStatus: InvoiceStatusFilter;
  toolsCustomerId: string;
  previewMonth: string;
  invariant: BillingInvariant | undefined;
  invariantFetching: boolean;
  invariantError: Error | undefined;
  hasInvariantSnapshot: boolean;
  preview: InvoicePreview | undefined;
  previewFetching: boolean;
  previewError: Error | undefined;
  hasPreviewSnapshot: boolean;
  onDraftMonthChange: (value: string) => void;
  onDraftStatusChange: (value: InvoiceStatusFilter) => void;
  onToolsCustomerIdChange: (value: string) => void;
  onPreviewMonthChange: (value: string) => void;
  onApplyFilters: () => void;
  onPageChange: (nextOffset: number) => void;
  onCheckInvariant: () => void;
  onPreviewInvoice: () => void;
};

export function BillingOverview({
  summary,
  summaryFetching,
  summaryError,
  hasSummarySnapshot,
  invoices,
  invoicesTotal,
  invoicesLimit,
  invoicesOffset,
  invoicesFetching,
  invoicesError,
  hasInvoicesSnapshot,
  draftMonth,
  draftStatus,
  toolsCustomerId,
  previewMonth,
  invariant,
  invariantFetching,
  invariantError,
  hasInvariantSnapshot,
  preview,
  previewFetching,
  previewError,
  hasPreviewSnapshot,
  onDraftMonthChange,
  onDraftStatusChange,
  onToolsCustomerIdChange,
  onPreviewMonthChange,
  onApplyFilters,
  onPageChange,
  onCheckInvariant,
  onPreviewInvoice,
}: BillingOverviewProps) {
  return (
    <PageChrome
      title="Billing"
      controlPanel={
        <div className="flex flex-wrap gap-4 text-sm">
          <Link className="text-muted-foreground hover:underline" to="/billing/exports">
            Ledger exports
          </Link>
        </div>
      }
    >
      <BillingSummarySection
        summary={summary}
        fetching={summaryFetching}
        error={summaryError}
        hasSnapshot={hasSummarySnapshot}
      />

      <BillingInvariantPanel
        draftCustomerId={toolsCustomerId}
        invariant={invariant}
        fetching={invariantFetching}
        error={invariantError}
        hasSnapshot={hasInvariantSnapshot}
        onDraftCustomerIdChange={onToolsCustomerIdChange}
        onCheck={onCheckInvariant}
      />

      <BillingPreviewPanel
        draftCustomerId={toolsCustomerId}
        draftMonth={previewMonth}
        preview={preview}
        fetching={previewFetching}
        error={previewError}
        hasSnapshot={hasPreviewSnapshot}
        onDraftCustomerIdChange={onToolsCustomerIdChange}
        onDraftMonthChange={onPreviewMonthChange}
        onPreview={onPreviewInvoice}
      />

      <BillingInvoices
        items={invoices}
        total={invoicesTotal}
        limit={invoicesLimit}
        offset={invoicesOffset}
        draftMonth={draftMonth}
        draftStatus={draftStatus}
        fetching={invoicesFetching}
        error={invoicesError}
        hasSnapshot={hasInvoicesSnapshot}
        onDraftMonthChange={onDraftMonthChange}
        onDraftStatusChange={onDraftStatusChange}
        onApplyFilters={onApplyFilters}
        onPageChange={onPageChange}
      />
    </PageChrome>
  );
}
