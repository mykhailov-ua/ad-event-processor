import { useCallback } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import {
  parseInvoiceDetailTab,
  type InvoiceDetail,
  type InvoiceDetailTab,
} from '../helpers/billing_api.js';
import { touchCustomerContext } from '../helpers/customer_context.js';
import { useResource } from '../helpers/use_resource.js';
import { InvoiceDetailView } from '../ui/billing/invoice_detail.js';
import { ErrorBlock } from '../ui/system/error_block.js';
import { PageSkeleton } from '../ui/system/page_skeleton.js';

export function InvoiceDetailPage() {
  const { id } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const invoiceId = id ?? '';
  const tab = parseInvoiceDetailTab(searchParams.get('tab'));

  const listUrl = invoiceId
    ? `/api/v1/billing/invoices/${encodeURIComponent(invoiceId)}`
    : null;
  const { data, loading, error, reload } = useResource<InvoiceDetail>(listUrl, {
    skip: !invoiceId,
  });

  const onTabChange = useCallback(
    (next: InvoiceDetailTab) => {
      const params = new URLSearchParams(searchParams);
      if (next === 'header') {
        params.delete('tab');
      } else {
        params.set('tab', next);
      }
      setSearchParams(params, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  if (!invoiceId) {
    return <ErrorBlock error={new Error('missing invoice id')} fallbackTitle="Invalid route" />;
  }

  if (loading && !data) {
    return <PageSkeleton rows={6} />;
  }

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load invoice" />;
  }

  if (!data) {
    return <ErrorBlock error={new Error('empty invoice')} fallbackTitle="Invoice not found" />;
  }

  if (data.customer_id) {
    touchCustomerContext(data.customer_id);
  }

  return (
    <InvoiceDetailView
      invoiceId={invoiceId}
      invoice={data}
      tab={tab}
      onTabChange={onTabChange}
      onReload={reload}
    />
  );
}
