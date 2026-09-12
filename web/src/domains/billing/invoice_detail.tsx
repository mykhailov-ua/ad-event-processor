import { adminTypography } from '@/lib/admin_kit';
import { Link } from 'react-router-dom';

import { PageChrome } from '@/shell/page_chrome';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import { shouldShowDirectoryRefreshError } from '@/shell/directory_load_state';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import type { BillingInvoiceLine, BillingLedgerLine, Invoice, InvoiceDelivery } from '@/api/types';
import { CustomerDetailPanel } from '@/domains/customers/customer_detail_panel';
import { CustomerDetailRow } from '@/domains/customers/customer_detail_row';
import { billingPanelError } from '@/domains/billing/billing_nav';
import { displayMicro, displayTimestamp } from '@/lib/display';

export type InvoiceDetailProps = {
  invoice: Invoice | undefined;
  deliveries?: InvoiceDelivery[];
  ledgerLines: BillingLedgerLine[];
  ledgerNextCursor?: string;
  fetching: boolean;
  deliveriesFetching: boolean;
  ledgerFetching: boolean;
  error: Error | undefined;
  deliveriesError: Error | undefined;
  ledgerError: Error | undefined;
  actionError: Error | undefined;
  hasSnapshot: boolean;
  canMutate: boolean;
  downloadingPdf: boolean;
  voiding: boolean;
  retryingDelivery: boolean;
  voidSuccess: boolean;
  retrySuccess: boolean;
  onDownloadPdf: () => void;
  onVoid: () => void;
  onRetryDelivery: () => void;
  onLoadMoreLedger: () => void;
};

export function InvoiceDetail({
  invoice,
  deliveries,
  ledgerLines,
  ledgerNextCursor,
  fetching,
  deliveriesFetching,
  ledgerFetching,
  error,
  deliveriesError,
  ledgerError,
  actionError,
  hasSnapshot,
  canMutate,
  downloadingPdf,
  voiding,
  retryingDelivery,
  voidSuccess,
  retrySuccess,
  onDownloadPdf,
  onVoid,
  onRetryDelivery,
  onLoadMoreLedger,
}: InvoiceDetailProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Invoice">{billingPanelError(error, 'Could not load invoice')}</PageChrome>
    );
  }

  if (!invoice) {
    return (
      <PageChrome title="Invoice">
        {billingPanelError(new Error('No invoice data returned.'), 'Invoice not found')}
      </PageChrome>
    );
  }

  const lines = invoice.lines ?? [];
  const status = invoice.status ?? '';
  const invoiceFetchState = { fetching, error, hasSnapshot };
  const ledgerHasSnapshot = ledgerLines.length > 0;
  const deliveriesHasSnapshot = (deliveries ?? []).length > 0;

  return (
    <PageChrome
      title={`Invoice ${invoice.billing_month ?? invoice.id}`}
      badge={status ? <Badge variant="outline">{status}</Badge> : undefined}
    >
      <p>
        <Link to="/exports">Billing exports</Link>
      </p>

      {shouldShowDirectoryRefreshError(invoiceFetchState) && error
        ? billingPanelError(error, 'Refresh failed')
        : null}

      <div>
        <Button disabled={downloadingPdf} onClick={onDownloadPdf} type="button" variant="outline">
          {downloadingPdf ? 'Downloading...' : 'Download PDF'}
        </Button>
        {canMutate ? (
          <>
            <Button
              disabled={voiding || status === 'void'}
              onClick={onVoid}
              type="button"
              variant="destructive"
            >
              {voiding ? 'Voiding...' : 'Void invoice'}
            </Button>
            <Button
              disabled={retryingDelivery}
              onClick={onRetryDelivery}
              type="button"
              variant="secondary"
            >
              {retryingDelivery ? 'Retrying...' : 'Retry delivery'}
            </Button>
          </>
        ) : null}
      </div>

      {actionError ? billingPanelError(actionError, 'Action failed') : null}
      {voidSuccess ? <p role="status">Invoice voided.</p> : null}
      {retrySuccess ? <p role="status">Delivery retry accepted.</p> : null}

      <CustomerDetailPanel>
        <CustomerDetailRow label="Invoice ID" value={invoice.id} />
        <CustomerDetailRow label="Customer ID" value={invoice.customer_id} />
        <CustomerDetailRow label="Billing month" value={invoice.billing_month} />
        <CustomerDetailRow
          label="Subtotal"
          value={displayMicro(invoice.subtotal_micro, invoice.subtotal_micro_display)}
        />
        <CustomerDetailRow
          label="Tax"
          value={displayMicro(invoice.tax_micro, invoice.tax_micro_display)}
        />
        <CustomerDetailRow
          label="Total"
          value={displayMicro(invoice.total_micro, invoice.total_micro_display)}
        />
        <CustomerDetailRow label="Currency" value={invoice.currency} />
        <CustomerDetailRow label="Tax scheme" value={invoice.tax_scheme} />
        <CustomerDetailRow label="Tax rate (bps)" value={invoice.tax_rate_bps} />
      </CustomerDetailPanel>

      <InvoiceLinesTable caption="Invoice lines" lines={lines} />

      <section>
        <h2>Ledger lines</h2>
        {shouldShowDirectoryRefreshError({
          fetching: ledgerFetching,
          error: ledgerError,
          hasSnapshot: ledgerHasSnapshot,
        }) && ledgerError
          ? billingPanelError(ledgerError, 'Ledger refresh failed')
          : null}
        {ledgerError && !ledgerHasSnapshot
          ? billingPanelError(ledgerError, 'Could not load ledger lines')
          : null}
        {ledgerLines.length === 0 && !ledgerFetching && !ledgerError ? (
          <EmptyState
            title="No ledger lines"
            description="No backing ledger rows for this invoice."
          />
        ) : (
          <>
            <LedgerLinesTable lines={ledgerLines} />
            {ledgerNextCursor ? (
              <Button
                disabled={ledgerFetching}
                onClick={onLoadMoreLedger}
                type="button"
                variant="outline"
              >
                {ledgerFetching ? 'Loading...' : 'Load more ledger lines'}
              </Button>
            ) : null}
          </>
        )}
      </section>

      <section>
        <h2>Deliveries</h2>
        {shouldShowDirectoryRefreshError({
          fetching: deliveriesFetching,
          error: deliveriesError,
          hasSnapshot: deliveriesHasSnapshot,
        }) && deliveriesError
          ? billingPanelError(deliveriesError, 'Deliveries refresh failed')
          : null}
        {deliveriesFetching && !deliveriesHasSnapshot ? <PageSkeleton /> : null}
        {deliveriesError && !deliveriesHasSnapshot
          ? billingPanelError(deliveriesError, 'Could not load deliveries')
          : null}
        {(deliveries ?? []).length === 0 && !deliveriesFetching && !deliveriesError ? (
          <EmptyState
            title="No deliveries"
            description="No delivery attempts recorded for this invoice."
          />
        ) : (
          <DeliveriesTable items={deliveries} />
        )}
      </section>
    </PageChrome>
  );
}

function InvoiceLinesTable({ caption, lines }: { caption: string; lines: BillingInvoiceLine[] }) {
  if (lines.length === 0) {
    return (
      <section className="grid gap-4">
        <h2 className={adminTypography.sectionTitle}>{caption}</h2>
        <p>No line items on this invoice.</p>
      </section>
    );
  }

  return (
    <section>
      <h2>{caption}</h2>
      <DirectoryTable horizontalScroll>
        <TableHeader>
          <TableRow>
            <DirectoryTableHead>Ledger type</DirectoryTableHead>
            <DirectoryTableHead>Amount (micro)</DirectoryTableHead>
            <DirectoryTableHead>Entry count</DirectoryTableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {lines.map((line, index) => (
            <TableRow key={`${line.ledger_type ?? 'line'}-${index}`}>
              <TableCell>{line.ledger_type ?? ''}</TableCell>
              <TableCell>{displayMicro(line.amount_micro)}</TableCell>
              <TableCell>{line.entry_count ?? ''}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </DirectoryTable>
    </section>
  );
}

function LedgerLinesTable({ lines }: { lines: BillingLedgerLine[] }) {
  return (
    <DirectoryTable horizontalScroll>
      <TableHeader>
        <TableRow>
          <DirectoryTableHead>ID</DirectoryTableHead>
          <DirectoryTableHead>Type</DirectoryTableHead>
          <DirectoryTableHead>Amount (micro)</DirectoryTableHead>
          <DirectoryTableHead>Created</DirectoryTableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {lines.map((row) => (
          <TableRow key={row.id ?? `${row.created_at}-${row.ledger_type}`}>
            <TableCell>{row.id ?? ''}</TableCell>
            <TableCell>{row.ledger_type ?? ''}</TableCell>
            <TableCell>{displayMicro(row.amount_micro)}</TableCell>
            <TableCell>{displayTimestamp(row.created_at)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </DirectoryTable>
  );
}

function DeliveriesTable({ items }: { items?: InvoiceDelivery[] }) {
  return (
    <DirectoryTable horizontalScroll>
      <TableHeader>
        <TableRow>
          <DirectoryTableHead>Status</DirectoryTableHead>
          <DirectoryTableHead>Provider</DirectoryTableHead>
          <DirectoryTableHead>Recipient</DirectoryTableHead>
          <DirectoryTableHead>Retries</DirectoryTableHead>
          <DirectoryTableHead>Updated</DirectoryTableHead>
          <DirectoryTableHead>Error</DirectoryTableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {(items ?? []).map((row) => (
          <TableRow key={row.id}>
            <TableCell>{row.status}</TableCell>
            <TableCell>{row.provider}</TableCell>
            <TableCell>{row.recipient}</TableCell>
            <TableCell>{row.retry_count}</TableCell>
            <TableCell>{displayTimestamp(row.updated_at)}</TableCell>
            <TableCell>{row.error_message ?? ''}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </DirectoryTable>
  );
}
