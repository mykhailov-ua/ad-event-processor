import { Link } from 'react-router-dom';

import { PageChrome } from '@/shell/page_chrome';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type {
  BillingInvoiceLine,
  BillingLedgerLine,
  Invoice,
  InvoiceDelivery,
} from '@/api/types';
import { displayMicro, displayTimestamp } from '@/lib/display';

export type InvoiceDetailProps = {
  invoice: Invoice | undefined;
  deliveries: InvoiceDelivery[];
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
    return <ErrorBlock title="Could not load invoice" message={error.message} />;
  }

  if (!invoice) {
    return <ErrorBlock title="Invoice not found" message="No invoice data returned." />;
  }

  const lines = invoice.lines ?? [];
  const status = invoice.status ?? '';

  return (
    <PageChrome
      title={`Invoice ${invoice.billing_month ?? invoice.id}`}
      badge={status ? <Badge variant="outline">{status}</Badge> : undefined}
    >
      <p className="text-sm text-muted-foreground">
        <Link className="hover:underline" to="/billing">
          Billing
        </Link>
      </p>

      <div className="flex flex-wrap gap-2">
        <Button disabled={downloadingPdf} onClick={onDownloadPdf} type="button" variant="outline">
          {downloadingPdf ? 'Downloading...' : 'Download PDF'}
        </Button>
        {canMutate ? (
          <>
            <Button disabled={voiding || status === 'void'} onClick={onVoid} type="button" variant="destructive">
              {voiding ? 'Voiding...' : 'Void invoice'}
            </Button>
            <Button disabled={retryingDelivery} onClick={onRetryDelivery} type="button" variant="secondary">
              {retryingDelivery ? 'Retrying...' : 'Retry delivery'}
            </Button>
          </>
        ) : null}
      </div>

      {actionError ? <ErrorBlock title="Action failed" message={actionError.message} /> : null}
      {voidSuccess ? (
        <p className="text-sm text-muted-foreground" role="status">
          Invoice voided.
        </p>
      ) : null}
      {retrySuccess ? (
        <p className="text-sm text-muted-foreground" role="status">
          Delivery retry accepted.
        </p>
      ) : null}

      <section className="ui-filter-panel gap-2 text-sm">
        <DetailRow label="Invoice ID" value={invoice.id} mono />
        <DetailRow label="Customer ID" value={invoice.customer_id} mono />
        <DetailRow label="Billing month" value={invoice.billing_month} />
        <DetailRow
          label="Subtotal"
          value={displayMicro(invoice.subtotal_micro, invoice.subtotal_micro_display)}
        />
        <DetailRow label="Tax" value={displayMicro(invoice.tax_micro, invoice.tax_micro_display)} />
        <DetailRow label="Total" value={displayMicro(invoice.total_micro, invoice.total_micro_display)} />
        <DetailRow label="Currency" value={invoice.currency} />
        <DetailRow label="Tax scheme" value={invoice.tax_scheme} />
        <DetailRow label="Tax rate (bps)" value={invoice.tax_rate_bps} />
      </section>

      <InvoiceLinesTable caption="Invoice lines" lines={lines} />

      <section className="grid gap-4">
        <h2 className="text-base font-semibold">Ledger lines</h2>
        {ledgerError && ledgerLines.length === 0 ? (
          <ErrorBlock title="Could not load ledger lines" message={ledgerError.message} />
        ) : ledgerLines.length === 0 && !ledgerFetching ? (
          <EmptyState title="No ledger lines" description="No backing ledger rows for this invoice." />
        ) : (
          <>
            <LedgerLinesTable lines={ledgerLines} />
            {ledgerNextCursor ? (
              <Button disabled={ledgerFetching} onClick={onLoadMoreLedger} type="button" variant="outline">
                {ledgerFetching ? 'Loading...' : 'Load more ledger lines'}
              </Button>
            ) : null}
          </>
        )}
        {ledgerError && ledgerLines.length > 0 ? (
          <ErrorBlock title="Ledger refresh failed" message={ledgerError.message} />
        ) : null}
      </section>

      <section className="grid gap-4">
        <h2 className="text-base font-semibold">Deliveries</h2>
        {deliveriesFetching && deliveries.length === 0 ? <PageSkeleton /> : null}
        {deliveriesError && deliveries.length === 0 ? (
          <ErrorBlock title="Could not load deliveries" message={deliveriesError.message} />
        ) : deliveries.length === 0 && !deliveriesFetching ? (
          <EmptyState title="No deliveries" description="No delivery attempts recorded for this invoice." />
        ) : (
          <DeliveriesTable items={deliveries} />
        )}
        {deliveriesError && deliveries.length > 0 ? (
          <ErrorBlock title="Deliveries refresh failed" message={deliveriesError.message} />
        ) : null}
      </section>

      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </PageChrome>
  );
}

function DetailRow({
  label,
  value,
  mono,
}: {
  label: string;
  value: string | number | undefined | null;
  mono?: boolean;
}) {
  return (
    <div className="grid grid-cols-[minmax(8rem,12rem)_1fr] gap-2">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className={mono ? 'font-mono text-xs break-all' : undefined}>{value ?? ''}</dd>
    </div>
  );
}

function InvoiceLinesTable({
  caption,
  lines,
}: {
  caption: string;
  lines: BillingInvoiceLine[];
}) {
  if (lines.length === 0) {
    return (
      <section className="grid gap-2">
        <h2 className="text-base font-semibold">{caption}</h2>
        <p className="text-sm text-muted-foreground">No line items on this invoice.</p>
      </section>
    );
  }

  return (
    <section className="grid gap-2">
      <h2 className="text-base font-semibold">{caption}</h2>
      <div className="ui-table-frame">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Ledger type</TableHead>
              <TableHead className="text-right">Amount (micro)</TableHead>
              <TableHead className="text-right">Entry count</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {lines.map((line, index) => (
              <TableRow key={`${line.ledger_type ?? 'line'}-${index}`}>
                <TableCell>{line.ledger_type ?? ''}</TableCell>
                <TableCell className="text-right tabular-nums">
                  {displayMicro(line.amount_micro)}
                </TableCell>
                <TableCell className="text-right tabular-nums">{line.entry_count ?? ''}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </section>
  );
}

function LedgerLinesTable({ lines }: { lines: BillingLedgerLine[] }) {
  return (
    <div className="ui-table-frame">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>ID</TableHead>
            <TableHead>Type</TableHead>
            <TableHead className="text-right">Amount (micro)</TableHead>
            <TableHead>Created</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {lines.map((row) => (
            <TableRow key={row.id ?? `${row.created_at}-${row.ledger_type}`}>
              <TableCell className="tabular-nums">{row.id ?? ''}</TableCell>
              <TableCell>{row.ledger_type ?? ''}</TableCell>
              <TableCell className="text-right tabular-nums">
                {displayMicro(row.amount_micro)}
              </TableCell>
              <TableCell>{displayTimestamp(row.created_at)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function DeliveriesTable({ items }: { items: InvoiceDelivery[] }) {
  return (
    <div className="ui-table-frame">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Status</TableHead>
            <TableHead>Provider</TableHead>
            <TableHead>Recipient</TableHead>
            <TableHead>Retries</TableHead>
            <TableHead>Updated</TableHead>
            <TableHead>Error</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map((row) => (
            <TableRow key={row.id}>
              <TableCell>{row.status}</TableCell>
              <TableCell>{row.provider}</TableCell>
              <TableCell>{row.recipient}</TableCell>
              <TableCell className="tabular-nums">{row.retry_count}</TableCell>
              <TableCell>{displayTimestamp(row.updated_at)}</TableCell>
              <TableCell className="max-w-xs truncate">{row.error_message ?? ''}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
