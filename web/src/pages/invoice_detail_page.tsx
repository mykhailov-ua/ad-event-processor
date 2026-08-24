import { useCallback, useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';
import { to } from '../lib/to.js';
import { apiConfirmed } from '../helpers/confirmed_api.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { can } from '../helpers/permissions.js';
import * as auth from '../helpers/auth.js';
import { formatAmountMicro } from '../helpers/money.js';
import { isPageBlockingError, mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { surfaceServiceErrorToast } from '../helpers/service_error_toast.js';
import {
  fetchInvoiceDeliveries,
  fetchInvoiceLedgerLines,
  retryInvoiceDelivery,
} from '../helpers/billing_admin_api.js';
import type {
  InvoiceDTO,
  InvoiceDeliveryDTO,
  InvoiceLedgerLineDTO,
  InvoiceLineDTO,
} from '../types/index.js';
import { shortCustomerId } from '../helpers/customer_context.js';
import { displayLabel } from '../helpers/display_labels.js';
import { useResource } from '../helpers/use_resource.js';
import { Breadcrumbs, type BreadcrumbItem } from '../components/breadcrumbs.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { Icon } from '../components/icon.js';
import { StatusBadge } from '../components/status_badge.js';

export function InvoiceDetailPage() {
  const { id = '' } = useParams();
  const user = auth.getUser();
  const canVoid = can(user?.permissions ?? [], 'customers:write');
  const canRetryDelivery = can(user?.permissions ?? [], 'customers:write');

  const [voidLoading, setVoidLoading] = useState(false);
  const [deliveries, setDeliveries] = useState<InvoiceDeliveryDTO[]>([]);
  const [deliveriesLoading, setDeliveriesLoading] = useState(false);
  const [deliveryRetryLoading, setDeliveryRetryLoading] = useState(false);
  const [deliveriesLoaded, setDeliveriesLoaded] = useState(false);
  const [ledgerLines, setLedgerLines] = useState<InvoiceLedgerLineDTO[]>([]);
  const [ledgerLinesLoading, setLedgerLinesLoading] = useState(false);
  const [ledgerLinesTotal, setLedgerLinesTotal] = useState(0);
  const [ledgerNextCursor, setLedgerNextCursor] = useState('');

  const {
    data: invoice,
    loading,
    error,
    reload,
  } = useResource<InvoiceDTO>(id ? `/api/v1/billing/invoices/${id}` : null);

  useEffect(() => {
    if (error) surfaceServiceErrorToast(error);
  }, [error]);

  const loadDeliveries = useCallback(async () => {
    if (!id) return;
    setDeliveriesLoading(true);
    const [res, err] = await to(fetchInvoiceDeliveries(id));
    setDeliveriesLoading(false);
    if (err) {
      setDeliveries([]);
      const view = mapServiceError(err);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      return;
    }
    setDeliveries(res?.items ?? []);
    setDeliveriesLoaded(true);
  }, [id]);

  useEffect(() => {
    if (invoice && !deliveriesLoaded && !deliveriesLoading) {
      void loadDeliveries();
    }
  }, [invoice, deliveriesLoaded, deliveriesLoading, loadDeliveries]);

  const loadLedgerLines = useCallback(
    async (cursor = '', append = false) => {
      if (!id) return;
      setLedgerLinesLoading(true);
      const [res, err] = await to(fetchInvoiceLedgerLines(id, cursor));
      setLedgerLinesLoading(false);
      if (err) {
        if (!append) {
          setLedgerLines([]);
          setLedgerLinesTotal(0);
        }
        setLedgerNextCursor('');
        const view = mapServiceError(err);
        pushToastMessage({ title: view.title, message: view.message, code: view.code });
        return;
      }
      const items = res?.items ?? [];
      setLedgerLines((prev) => (append ? [...prev, ...items] : items));
      setLedgerLinesTotal(res?.total ?? 0);
      setLedgerNextCursor(res?.next_cursor ?? '');
    },
    [id]
  );

  const ledgerLoadedRef = useRef(false);
  useEffect(() => {
    if (!invoice || ledgerLoadedRef.current) return;
    ledgerLoadedRef.current = true;
    void loadLedgerLines('');
  }, [invoice, loadLedgerLines]);

  const handleVoid = async () => {
    setVoidLoading(true);
    const [, voidErr] = await to(
      apiConfirmed(`/api/v1/billing/invoices/${id}/void`, { method: 'POST' })
    );
    if (voidErr) {
      if (voidErr instanceof ConfirmCancelledError) {
        setVoidLoading(false);
        return;
      }
      const view = mapServiceError(voidErr);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      setVoidLoading(false);
      return;
    }
    pushToastMessage({ title: 'Invoice voided', message: id });
    reload();
    setVoidLoading(false);
  };

  const handleDeliveryRetry = async () => {
    if (!canRetryDelivery || deliveryRetryLoading) return;
    setDeliveryRetryLoading(true);
    const [, err] = await to(retryInvoiceDelivery(id));
    setDeliveryRetryLoading(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      const view = mapServiceError(err);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      return;
    }
    pushToastMessage({ title: 'Delivery retry queued', message: id });
    void loadDeliveries();
  };

  if (loading) return <span className="text-muted">Loading...</span>;

  if (error) {
    const view = mapServiceError(error);
    if (isPageBlockingError(view) || view.kind === 'empty') {
      return <ErrorBlock error={error} />;
    }
    return null;
  }

  if (!invoice) return null;

  const invoiceVoid = String(invoice.status ?? '').toUpperCase() === 'VOID';
  const crumbs: BreadcrumbItem[] = [{ label: 'Billing', href: '/billing' }];
  if (invoice.customer_id) {
    crumbs.push({
      label: shortCustomerId(invoice.customer_id, 12),
      href: `/customers/${invoice.customer_id}`,
    });
  }
  crumbs.push({ label: shortCustomerId(id, 12) });

  return (
    <>
      <div className="page-header">
        <Breadcrumbs items={crumbs} />
        <div className="page-header__row">
          <div className="flex items-center gap-2">
            <Icon name="file-text" size={20} className="text-muted" />
            <h1 className="page-header__title">Invoice</h1>
          </div>
          {invoice.status ? <StatusBadge status={invoice.status} kind="invoice" /> : null}
          <div className="flex items-center gap-2 ml-auto">
            <Button
              label="PDF"
              variant="secondary"
              size="sm"
              icon="download"
              onClick={() =>
                window.open(`/api/v1/billing/invoices/${id}/pdf`, '_blank', 'noopener,noreferrer')
              }
            />
            {canVoid && !invoiceVoid ? (
              <Button
                label="Void"
                variant="danger"
                size="sm"
                icon="trash"
                loading={voidLoading}
                disabled={voidLoading}
                onClick={() => void handleVoid()}
              />
            ) : null}
          </div>
        </div>
      </div>

      <div className="grid-stats section-block">
        <div className="metric-card">
          <div className="metric-card__label">Month</div>
          <div className="metric-card__value">{invoice.billing_month ?? '-'}</div>
        </div>
        <div className="metric-card">
          <div className="metric-card__label">Total</div>
          <div className="metric-card__value font-mono">
            {formatAmountMicro(invoice.total_micro ?? 0, invoice.currency)}
          </div>
        </div>
        <div className="metric-card">
          <div className="metric-card__label">Tax</div>
          <div className="metric-card__value font-mono">
            {formatAmountMicro(invoice.tax_micro ?? 0, invoice.currency)}
          </div>
        </div>
        <div className="metric-card">
          <div className="metric-card__label">Customer</div>
          <div className="metric-card__value font-mono">{invoice.customer_id}</div>
        </div>
      </div>

      {(invoice.lines?.length ?? 0) > 0 ? (
        <div className="table-wrapper elevation-raised mt-4">
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">Ledger type</th>
                <th scope="col">Amount (micro)</th>
                <th scope="col">Entries</th>
              </tr>
            </thead>
            <tbody>
              {(invoice.lines ?? []).map((line: InvoiceLineDTO, index) => (
                <tr key={`${line.ledger_type}-${index}`}>
                  <td>{displayLabel(line.ledger_type)}</td>
                  <td className="font-mono">{String(line.amount_micro ?? 0)}</td>
                  <td>{String(line.entry_count ?? 0)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      <section className="section-block" data-testid="invoice-ledger-lines">
        <h2 className="subsection-title">Ledger lines</h2>
        {ledgerLinesLoading ? <span className="text-muted">Loading...</span> : null}
        <div className="table-wrapper elevation-raised">
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">ID</th>
                <th scope="col">Type</th>
                <th scope="col">Amount</th>
                <th scope="col">Created</th>
              </tr>
            </thead>
            <tbody>
              {ledgerLines.length === 0 && !ledgerLinesLoading ? (
                <tr>
                  <td colSpan={4} className="data-table__empty">
                    <div className="empty-state">
                      <div className="empty-state__title">No ledger lines</div>
                      <div className="empty-state__desc text-muted text-sm">
                        No matching balance_ledger rows for this invoice month.
                      </div>
                    </div>
                  </td>
                </tr>
              ) : null}
              {ledgerLines.map((line) => (
                <tr key={line.id}>
                  <td className="font-mono">{String(line.id ?? '-')}</td>
                  <td>{displayLabel(line.ledger_type)}</td>
                  <td className="font-mono">
                    {formatAmountMicro(line.amount_micro ?? 0, invoice.currency)}
                  </td>
                  <td className="text-muted text-sm">
                    {line.created_at ? new Date(line.created_at).toLocaleString() : '-'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {ledgerLinesTotal > 0 ? (
          <p className="text-muted text-sm mt-2">
            Showing {ledgerLines.length} of {ledgerLinesTotal} lines
          </p>
        ) : null}
        {ledgerNextCursor ? (
          <Button
            label="Load more"
            variant="secondary"
            size="sm"
            className="mt-2"
            loading={ledgerLinesLoading}
            disabled={ledgerLinesLoading}
            onClick={() => void loadLedgerLines(ledgerNextCursor, true)}
          />
        ) : null}
      </section>

      <section className="section-block" data-testid="invoice-deliveries">
        <div className="flex items-center gap-2 mb-3">
          <h2 className="subsection-title">Delivery attempts</h2>
          {canRetryDelivery && !invoiceVoid ? (
            <Button
              label="Retry delivery"
              variant="secondary"
              size="sm"
              className="ml-auto"
              loading={deliveryRetryLoading}
              disabled={deliveryRetryLoading}
              data-testid="invoice-delivery-retry"
              onClick={() => void handleDeliveryRetry()}
            />
          ) : null}
        </div>
        {deliveriesLoading ? <span className="text-muted">Loading...</span> : null}
        <div className="table-wrapper elevation-raised">
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">Channel</th>
                <th scope="col">Status</th>
                <th scope="col">Recipient</th>
                <th scope="col">Last error</th>
                <th scope="col">Sent at</th>
              </tr>
            </thead>
            <tbody>
              {deliveries.length === 0 && !deliveriesLoading ? (
                <tr>
                  <td colSpan={5} className="data-table__empty">
                    <div className="empty-state">
                      <div className="empty-state__title">No deliveries yet</div>
                      <div className="empty-state__desc text-muted text-sm">
                        Invoice delivery attempts will appear here after send.
                      </div>
                    </div>
                  </td>
                </tr>
              ) : null}
              {deliveries.map((d) => (
                <tr key={`${d.provider}-${d.updated_at}-${d.recipient}`}>
                  <td>{displayLabel(d.provider)}</td>
                  <td>{d.status}</td>
                  <td className="font-mono text-xs">{d.recipient ?? '-'}</td>
                  <td className="text-xs text-muted">{d.error_message ?? '-'}</td>
                  <td className="text-muted text-xs">
                    {d.updated_at ? new Date(d.updated_at).toLocaleString() : '-'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </>
  );
}
