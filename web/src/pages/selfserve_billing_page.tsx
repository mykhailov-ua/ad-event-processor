import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { currentStatementMonth } from '../helpers/customers_api.js';
import {
  buildSelfServeInvoicesUrl,
  buildSelfServeStatementUrl,
} from '../helpers/selfserve_api.js';
import type { BillingStatement } from '../helpers/customers_api.js';
import type { SelfServeInvoiceListResponse } from '../helpers/selfserve_api.js';
import { useResource } from '../helpers/use_resource.js';
import { BillingPanel } from '../ui/selfserve/billing_panel.js';
import { SelfServeShell } from '../ui/selfserve/selfserve_shell.js';

const INVOICE_LIMIT = 25;

function parseOffset(raw: string | null): number {
  const value = Number.parseInt(raw ?? '', 10);
  if (!Number.isFinite(value) || value < 0) return 0;
  return value;
}

export function SelfServeBillingPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const month = searchParams.get('month') ?? currentStatementMonth();
  const offset = parseOffset(searchParams.get('offset'));

  const statementUrl = useMemo(() => buildSelfServeStatementUrl(month), [month]);
  const invoicesUrl = useMemo(
    () => buildSelfServeInvoicesUrl(INVOICE_LIMIT, offset),
    [offset]
  );

  const {
    data: statement,
    loading: statementLoading,
    error: statementError,
  } = useResource<BillingStatement>(statementUrl);
  const {
    data: invoicesData,
    loading: invoicesLoading,
    error: invoicesError,
  } = useResource<SelfServeInvoiceListResponse>(invoicesUrl);

  const onMonthChange = useCallback(
    (nextMonth: string) => {
      const next = new URLSearchParams(searchParams);
      if (nextMonth) next.set('month', nextMonth);
      else next.delete('month');
      next.delete('offset');
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  const onOffsetChange = useCallback(
    (nextOffset: number) => {
      const next = new URLSearchParams(searchParams);
      if (nextOffset <= 0) next.delete('offset');
      else next.set('offset', String(nextOffset));
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  return (
    <SelfServeShell>
      <BillingPanel
        month={month}
        statement={statement}
        statementLoading={statementLoading}
        statementError={statementError}
        invoices={invoicesData?.invoices ?? []}
        invoiceTotal={invoicesData?.total ?? 0}
        limit={INVOICE_LIMIT}
        offset={offset}
        invoicesLoading={invoicesLoading}
        invoicesError={invoicesError}
        onMonthChange={onMonthChange}
        onOffsetChange={onOffsetChange}
      />
    </SelfServeShell>
  );
}
