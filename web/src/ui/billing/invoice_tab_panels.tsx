import { useEffect, useState } from 'react';
import {
  fetchInvoiceLedgerLines,
  invoicePdfUrl,
  retryInvoiceDelivery,
  voidInvoice,
  type InvoiceDetail,
  type InvoiceDelivery,
  type InvoiceLedgerLine,
} from '../../helpers/billing_api.js';
import { ConfirmCancelledError } from '../../helpers/confirmed_api.js';
import { formatLocaleDateTime as formatDate } from '../../helpers/format_display.js';
import { formatAmountMicro } from '../../helpers/money.js';
import { pushToastMessage } from '../../helpers/toast_ui.js';
import { useResource } from '../../helpers/use_resource.js';
import { Button } from '../system/button.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import styles from './invoice_detail.module.css';

export function InvoiceToolbar({
  invoiceId,
  onReload,
}: {
  invoiceId: string;
  onReload: () => void;
}) {
  const [voiding, setVoiding] = useState(false);
  const [retrying, setRetrying] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const onVoid = async () => {
    setVoiding(true);
    setError(null);
    try {
      await voidInvoice(invoiceId);
      pushToastMessage({ title: 'Voided', message: 'Invoice void requested' });
      onReload();
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setError(err);
    } finally {
      setVoiding(false);
    }
  };

  const onRetryDelivery = async () => {
    setRetrying(true);
    setError(null);
    try {
      await retryInvoiceDelivery(invoiceId);
      pushToastMessage({ title: 'Retry queued', message: 'Delivery retry requested' });
      onReload();
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setError(err);
    } finally {
      setRetrying(false);
    }
  };

  return (
    <div className={styles.panel}>
      <div className={styles.toolbar}>
        <Button variant="secondary" size="sm" type="button" disabled={voiding} onClick={() => void onVoid()}>
          {voiding ? 'Voiding...' : 'Void invoice'}
        </Button>
        <Button
          variant="secondary"
          size="sm"
          type="button"
          disabled={retrying}
          onClick={() => void onRetryDelivery()}
        >
          {retrying ? 'Retrying...' : 'Retry delivery'}
        </Button>
      </div>
      {error ? <ErrorBlock error={error} fallbackTitle="Action failed" /> : null}
    </div>
  );
}

export function InvoiceHeaderPanel({ invoice }: { invoice: InvoiceDetail }) {
  return (
    <div className={styles.panel}>
      <dl className={styles.dl}>
        <dt>ID</dt>
        <dd className={styles.mono}>{invoice.id ?? '-'}</dd>
        <dt>Customer</dt>
        <dd className={styles.mono}>{invoice.customer_id ?? '-'}</dd>
        <dt>Billing month</dt>
        <dd>{invoice.billing_month ?? '-'}</dd>
        <dt>Status</dt>
        <dd>{invoice.status ?? '-'}</dd>
        <dt>Currency</dt>
        <dd>{invoice.currency ?? 'USD'}</dd>
        <dt>Subtotal</dt>
        <dd>{formatAmountMicro(invoice.subtotal_micro, invoice.currency)}</dd>
        <dt>Tax</dt>
        <dd>{formatAmountMicro(invoice.tax_micro, invoice.currency)}</dd>
        <dt>Total</dt>
        <dd>{formatAmountMicro(invoice.total_micro, invoice.currency)}</dd>
        <dt>Tax scheme</dt>
        <dd>{invoice.tax_scheme ?? '-'}</dd>
        <dt>Tax rate (bps)</dt>
        <dd>{invoice.tax_rate_bps != null ? String(invoice.tax_rate_bps) : '-'}</dd>
      </dl>
      <div className={styles.table}>
        <div className={styles.tableHead}>
          <span>Description</span>
          <span>Quantity</span>
          <span>Amount</span>
          <span />
        </div>
        {(invoice.lines ?? []).map((line, index) => (
          <div key={`${line.description ?? index}`} className={styles.tableRow}>
            <span>{line.description ?? '-'}</span>
            <span>{line.quantity != null ? String(line.quantity) : '-'}</span>
            <span>{formatAmountMicro(line.amount_micro, invoice.currency)}</span>
            <span />
          </div>
        ))}
      </div>
    </div>
  );
}

export function InvoiceLedgerPanel({ invoiceId }: { invoiceId: string }) {
  const [lines, setLines] = useState<InvoiceLedgerLine[]>([]);
  const [nextCursor, setNextCursor] = useState<string | undefined>(undefined);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const loadPage = async (pageCursor: string | undefined, append: boolean) => {
    if (append) setLoadingMore(true);
    else setLoading(true);
    setError(null);
    try {
      const result = await fetchInvoiceLedgerLines(invoiceId, pageCursor);
      const items = result.items ?? [];
      setLines((prev) => (append ? [...prev, ...items] : items));
      setNextCursor(result.next_cursor);
    } catch (err) {
      setError(err);
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  };

  useEffect(() => {
    void loadPage(undefined, false);
  }, [invoiceId]);

  if (loading && lines.length === 0) return <PageSkeleton rows={5} />;
  if (error) return <ErrorBlock error={error} fallbackTitle="Failed to load ledger lines" />;

  return (
    <div className={styles.panel}>
      <div className={styles.table}>
        <div className={styles.tableHead}>
          <span>Description</span>
          <span>Amount</span>
          <span>Created</span>
          <span>ID</span>
        </div>
        {lines.map((row, index) => (
          <div key={row.id ?? `${row.created_at ?? index}`} className={styles.tableRow}>
            <span>{row.description ?? '-'}</span>
            <span>{formatAmountMicro(row.amount_micro)}</span>
            <span>{formatDate(row.created_at)}</span>
            <span className={styles.mono}>{row.id ?? '-'}</span>
          </div>
        ))}
      </div>
      {nextCursor ? (
        <div className={styles.actions}>
          <Button
            variant="secondary"
            size="sm"
            type="button"
            disabled={loadingMore}
            onClick={() => void loadPage(nextCursor, true)}
          >
            {loadingMore ? 'Loading...' : 'Load more'}
          </Button>
        </div>
      ) : null}
    </div>
  );
}

export function InvoiceDeliveriesPanel({ invoiceId }: { invoiceId: string }) {
  const url = `/api/v1/billing/invoices/${encodeURIComponent(invoiceId)}/deliveries`;
  const { data, loading, error } = useResource<{ items?: InvoiceDelivery[] }>(url);

  if (loading) return <PageSkeleton rows={4} />;
  if (error) return <ErrorBlock error={error} fallbackTitle="Failed to load deliveries" />;

  const items = data?.items ?? [];

  return (
    <div className={styles.panel}>
      <div className={styles.table}>
        <div className={styles.tableHead}>
          <span>Status</span>
          <span>Provider</span>
          <span>Recipient</span>
          <span>Created</span>
        </div>
        {items.map((row, index) => (
          <div key={row.id ?? `${row.created_at ?? index}`} className={styles.tableRow}>
            <span>{row.status ?? '-'}</span>
            <span>{row.provider ?? '-'}</span>
            <span>{row.recipient ?? '-'}</span>
            <span>{formatDate(row.created_at)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

export function InvoicePdfPanel({ invoiceId }: { invoiceId: string }) {
  const pdfUrl = invoicePdfUrl(invoiceId);
  return (
    <div className={styles.panel}>
      <a className={styles.pdfLink} href={pdfUrl} target="_blank" rel="noreferrer">
        Download invoice PDF
      </a>
    </div>
  );
}
