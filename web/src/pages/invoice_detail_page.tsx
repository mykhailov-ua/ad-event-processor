import { useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';

import {
  downloadInvoicePdf,
  getInvoice,
  listInvoiceDeliveries,
  listInvoiceLedgerLines,
  retryInvoiceDelivery,
  voidInvoice,
} from '@/api/billing_api';
import type { BillingLedgerLine } from '@/api/types';
import { InvoiceDetail } from '@/domains/billing/invoice_detail';
import { useBreadcrumbSegmentLabel } from '@/shell/breadcrumb_context';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';
import { triggerBlobDownload } from '@/lib/trigger_blob_download';

const LEDGER_PAGE_LIMIT = 50;

function newIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `retry-${Date.now()}`;
}

export function InvoiceDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [appliedLedgerCursor, setAppliedLedgerCursor] = useState<string | undefined>();
  const [ledgerLines, setLedgerLines] = useState<BillingLedgerLine[]>([]);
  const [ledgerNextCursor, setLedgerNextCursor] = useState<string | undefined>();
  const [ledgerEpoch, setLedgerEpoch] = useState(0);
  const [downloadingPdf, setDownloadingPdf] = useState(false);
  const [voiding, setVoiding] = useState(false);
  const [retryingDelivery, setRetryingDelivery] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();
  const [voidSuccess, setVoidSuccess] = useState(false);
  const [retrySuccess, setRetrySuccess] = useState(false);
  const [invoiceRefreshToken, setInvoiceRefreshToken] = useState(0);

  const { user } = useSession();
  const canMutate = user?.permissions?.includes('customers:write') ?? false;

  const invoiceResource = useResource(
    (signal) => {
      if (!id) {
        return Promise.reject(new Error('Invoice id is required'));
      }
      return getInvoice(id, signal);
    },
    [id, invoiceRefreshToken],
  );

  const deliveriesResource = useResource(
    (signal) => {
      if (!id) {
        return Promise.reject(new Error('Invoice id is required'));
      }
      return listInvoiceDeliveries(id, signal);
    },
    [id, invoiceRefreshToken],
  );

  const ledgerResource = useResource(
    (signal) => {
      if (!id) {
        return Promise.reject(new Error('Invoice id is required'));
      }
      return listInvoiceLedgerLines(
        id,
        { limit: LEDGER_PAGE_LIMIT, cursor: appliedLedgerCursor },
        signal,
      );
    },
    [id, appliedLedgerCursor, ledgerEpoch],
  );

  useEffect(() => {
    setAppliedLedgerCursor(undefined);
    setLedgerLines([]);
    setLedgerNextCursor(undefined);
    setLedgerEpoch((value) => value + 1);
  }, [id]);

  useEffect(() => {
    if (!ledgerResource.data) {
      return;
    }
    const items = ledgerResource.data.items ?? [];
    setLedgerLines((prev) => (appliedLedgerCursor ? [...prev, ...items] : items));
    setLedgerNextCursor(ledgerResource.data.next_cursor);
  }, [appliedLedgerCursor, ledgerResource.data]);

  const resetLedger = useCallback(() => {
    setAppliedLedgerCursor(undefined);
    setLedgerLines([]);
    setLedgerNextCursor(undefined);
    setLedgerEpoch((value) => value + 1);
  }, []);

  const onLoadMoreLedger = useCallback(() => {
    if (!ledgerNextCursor) {
      return;
    }
    setAppliedLedgerCursor(ledgerNextCursor);
  }, [ledgerNextCursor]);

  const onDownloadPdf = useCallback(async () => {
    if (!id) {
      return;
    }
    setDownloadingPdf(true);
    setActionError(undefined);
    try {
      const blob = await downloadInvoicePdf(id);
      triggerBlobDownload(blob, `invoice-${id}.pdf`);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setDownloadingPdf(false);
    }
  }, [id]);

  const onVoid = useCallback(async () => {
    if (!id || !canMutate) {
      return;
    }
    setVoiding(true);
    setActionError(undefined);
    setVoidSuccess(false);
    try {
      await voidInvoice(id);
      setVoidSuccess(true);
      setInvoiceRefreshToken((value) => value + 1);
      resetLedger();
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setVoiding(false);
    }
  }, [canMutate, id, resetLedger]);

  const onRetryDelivery = useCallback(async () => {
    if (!id || !canMutate) {
      return;
    }
    setRetryingDelivery(true);
    setActionError(undefined);
    setRetrySuccess(false);
    try {
      await retryInvoiceDelivery(id, newIdempotencyKey());
      setRetrySuccess(true);
      setInvoiceRefreshToken((value) => value + 1);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setRetryingDelivery(false);
    }
  }, [canMutate, id]);

  const invoiceLabel = invoiceResource.data?.billing_month
    ? `Invoice ${invoiceResource.data.billing_month}`
    : undefined;
  useBreadcrumbSegmentLabel(id, invoiceLabel);

  return (
    <InvoiceDetail
      invoice={invoiceResource.data}
      deliveries={deliveriesResource.data?.items ?? []}
      ledgerLines={ledgerLines}
      ledgerNextCursor={ledgerNextCursor}
      fetching={invoiceResource.fetching}
      deliveriesFetching={deliveriesResource.fetching}
      ledgerFetching={ledgerResource.fetching}
      error={invoiceResource.error}
      deliveriesError={deliveriesResource.error}
      ledgerError={ledgerResource.error}
      actionError={actionError}
      hasSnapshot={invoiceResource.data != null}
      canMutate={canMutate}
      downloadingPdf={downloadingPdf}
      voiding={voiding}
      retryingDelivery={retryingDelivery}
      voidSuccess={voidSuccess}
      retrySuccess={retrySuccess}
      onDownloadPdf={onDownloadPdf}
      onVoid={onVoid}
      onRetryDelivery={onRetryDelivery}
      onLoadMoreLedger={onLoadMoreLedger}
    />
  );
}
