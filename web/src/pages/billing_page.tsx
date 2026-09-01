import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';

import {
  getBillingInvariant,
  getBillingSummary,
  listInvoices,
  previewInvoice,
} from '@/api/billing_api';
import type { BillingInvariantQuery, InvoiceListQuery, PreviewInvoiceRequest } from '@/api/types';
import { PageChrome } from '@/components/system/page_chrome';
import {
  BillingInvoices,
  type InvoiceStatusFilter,
} from '@/domains/billing/billing_invoices';
import { BillingSummarySection } from '@/domains/billing/billing_summary';
import { BillingInvariantPanel, BillingPreviewPanel } from '@/domains/billing/billing_tools';
import { useResource } from '@/hooks/use_resource';
import { useSession } from '@/hooks/use_session';
import { DEFAULT_LIST_LIMIT, parseListLimit, parseListOffset } from '@/lib/list_query';

function parseMonth(raw: string | null): string {
  return raw ?? '';
}

function parseStatus(raw: string | null): InvoiceStatusFilter {
  if (raw === 'draft' || raw === 'finalized' || raw === 'void') {
    return raw;
  }
  return '';
}

function buildInvoiceQuery(params: URLSearchParams): InvoiceListQuery {
  const month = params.get('month');
  const status = params.get('status');

  return {
    month: month ?? undefined,
    status: status ?? undefined,
    limit: parseListLimit(params.get('limit')),
    offset: parseListOffset(params.get('offset')),
  };
}

export function BillingPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { session } = useSession();

  const invoiceQuery = useMemo(() => buildInvoiceQuery(searchParams), [searchParams]);
  const appliedMonth = parseMonth(searchParams.get('month'));
  const appliedStatus = parseStatus(searchParams.get('status'));

  const [draftMonth, setDraftMonth] = useState(appliedMonth);
  const [draftStatus, setDraftStatus] = useState<InvoiceStatusFilter>(appliedStatus);
  const [toolsCustomerId, setToolsCustomerId] = useState(session?.default_customer_id ?? '');
  const [previewMonth, setPreviewMonth] = useState('');
  const [invariantQuery, setInvariantQuery] = useState<BillingInvariantQuery | null>(null);
  const [invariantToken, setInvariantToken] = useState(0);
  const [previewRequest, setPreviewRequest] = useState<PreviewInvoiceRequest | null>(null);
  const [previewToken, setPreviewToken] = useState(0);

  useEffect(() => {
    setDraftMonth(appliedMonth);
    setDraftStatus(appliedStatus);
  }, [appliedMonth, appliedStatus]);

  useEffect(() => {
    if (session?.default_customer_id && !toolsCustomerId) {
      setToolsCustomerId(session.default_customer_id);
    }
  }, [session?.default_customer_id, toolsCustomerId]);

  const summaryResource = useResource((signal) => getBillingSummary(signal), []);
  const invoicesResource = useResource(
    (signal) => listInvoices(invoiceQuery, signal),
    [invoiceQuery.limit, invoiceQuery.offset, invoiceQuery.month, invoiceQuery.status],
  );

  const invariantResource = useResource(
    (signal) => {
      if (!invariantQuery) {
        return Promise.resolve(undefined);
      }
      return getBillingInvariant(invariantQuery, signal);
    },
    [invariantQuery?.customer_id, invariantToken],
  );

  const previewResource = useResource(
    (signal) => {
      if (!previewRequest) {
        return Promise.resolve(undefined);
      }
      return previewInvoice(previewRequest, signal);
    },
    [previewRequest?.customer_id, previewRequest?.billing_month, previewToken],
  );

  const updateInvoiceQuery = useCallback(
    (patch: Partial<InvoiceListQuery>) => {
      const next = new URLSearchParams(searchParams);
      const merged: InvoiceListQuery = { ...invoiceQuery, ...patch };

      if (merged.month) {
        next.set('month', merged.month);
      } else {
        next.delete('month');
      }
      if (merged.status) {
        next.set('status', merged.status);
      } else {
        next.delete('status');
      }
      next.set('limit', String(merged.limit ?? DEFAULT_LIST_LIMIT));
      next.set('offset', String(merged.offset ?? 0));

      setSearchParams(next, { replace: true });
    },
    [invoiceQuery, searchParams, setSearchParams],
  );

  const onPageChange = useCallback(
    (nextOffset: number) => {
      updateInvoiceQuery({ offset: Math.max(0, nextOffset) });
    },
    [updateInvoiceQuery],
  );

  const onApplyFilters = useCallback(() => {
    updateInvoiceQuery({
      month: draftMonth || undefined,
      status: draftStatus || undefined,
      offset: 0,
    });
  }, [draftMonth, draftStatus, updateInvoiceQuery]);

  const onCheckInvariant = useCallback(() => {
    const customerId = toolsCustomerId.trim();
    setInvariantQuery(customerId ? { customer_id: customerId } : {});
    setInvariantToken((value) => value + 1);
  }, [toolsCustomerId]);

  const onPreviewInvoice = useCallback(() => {
    const customerId = toolsCustomerId.trim();
    if (!customerId || !previewMonth) {
      return;
    }
    setPreviewRequest({ customer_id: customerId, billing_month: previewMonth });
    setPreviewToken((value) => value + 1);
  }, [previewMonth, toolsCustomerId]);

  return (
    <PageChrome title="Billing">
      <div className="flex flex-wrap gap-4 text-sm">
        <Link className="text-muted-foreground hover:underline" to="/billing/exports">
          Ledger exports
        </Link>
      </div>

      <BillingSummarySection
        summary={summaryResource.data}
        fetching={summaryResource.fetching}
        error={summaryResource.error}
        hasSnapshot={summaryResource.data != null}
      />

      <BillingInvariantPanel
        draftCustomerId={toolsCustomerId}
        invariant={invariantResource.data}
        fetching={invariantResource.fetching}
        error={invariantResource.error}
        hasSnapshot={invariantResource.data != null}
        onDraftCustomerIdChange={setToolsCustomerId}
        onCheck={onCheckInvariant}
      />

      <BillingPreviewPanel
        draftCustomerId={toolsCustomerId}
        draftMonth={previewMonth}
        preview={previewResource.data}
        fetching={previewResource.fetching}
        error={previewResource.error}
        hasSnapshot={previewResource.data != null}
        onDraftCustomerIdChange={setToolsCustomerId}
        onDraftMonthChange={setPreviewMonth}
        onPreview={onPreviewInvoice}
      />

      <BillingInvoices
        items={invoicesResource.data?.items ?? []}
        total={invoicesResource.data?.total ?? 0}
        limit={invoicesResource.data?.limit ?? invoiceQuery.limit ?? DEFAULT_LIST_LIMIT}
        offset={invoicesResource.data?.offset ?? invoiceQuery.offset ?? 0}
        draftMonth={draftMonth}
        draftStatus={draftStatus}
        fetching={invoicesResource.fetching}
        error={invoicesResource.error}
        hasSnapshot={invoicesResource.data != null}
        onDraftMonthChange={setDraftMonth}
        onDraftStatusChange={setDraftStatus}
        onApplyFilters={onApplyFilters}
        onPageChange={onPageChange}
      />
    </PageChrome>
  );
}
