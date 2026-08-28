import { useCallback, useEffect, useState } from 'react';
import { to } from '../lib/to.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { fetchCustomerPayments } from '../helpers/billing_admin_api.js';
import { formatPaymentStatus } from '../helpers/selfserve_billing_api.js';
import type { PaymentHistoryRowDTO } from '../types/billing.js';
import { formatAmountMicro } from '../helpers/money.js';
import { CopyableUuid } from './copyable_uuid.js';
import { PaginationBar } from './pagination_bar.js';

const PAGE_SIZE = 10;

export type BillingPaymentHistoryPanelProps = {
  customerId: string;
};

function TableSkeleton({ cols }: { cols: number }) {
  return (
    <>
      {Array.from({ length: 3 }, (_, i) => (
        <tr key={`pay-sk-${i}`} className="data-table__row--skeleton" aria-hidden="true">
          {Array.from({ length: cols }, (__, j) => (
            <td key={`pay-sk-${i}-${j}`}>
              <span className="skeleton-bar" />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}

export function BillingPaymentHistoryPanel({ customerId }: BillingPaymentHistoryPanelProps) {
  const [rows, setRows] = useState<PaymentHistoryRowDTO[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    if (!customerId) return;
    setLoading(true);
    const offset = page * PAGE_SIZE;
    const [res, err] = await to(fetchCustomerPayments(customerId, PAGE_SIZE, offset));
    setLoading(false);
    if (err) {
      pushToastMessage({ title: 'Payments failed', message: mapServiceError(err).message });
      setRows([]);
      setTotal(0);
      return;
    }
    setRows(res?.items ?? []);
    setTotal(res?.total ?? 0);
  }, [customerId, page]);

  useEffect(() => {
    void load();
  }, [load]);

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <section className="section-card stack" data-testid="billing-payment-history-panel">
      <div className="flex items-center justify-between gap-2">
        <h3 className="subsection-title">Payment history</h3>
        {totalPages > 1 ? (
          <PaginationBar
            label={`${page + 1} / ${totalPages}`}
            prevDisabled={page === 0}
            nextDisabled={page >= totalPages - 1}
            onPrev={() => setPage((p) => Math.max(0, p - 1))}
            onNext={() => setPage((p) => p + 1)}
          />
        ) : null}
      </div>
      <div className="table-wrapper">
        <table
          className="data-table"
          aria-label="Payment history"
          data-testid="billing-payments-table"
        >
          <thead>
            <tr>
              <th>Intent</th>
              <th>Amount</th>
              <th>Status</th>
              <th>Provider</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>
            {loading ? <TableSkeleton cols={5} /> : null}
            {!loading && rows.length === 0 ? (
              <tr>
                <td colSpan={5} className="text-muted">
                  No payment intents.
                </td>
              </tr>
            ) : null}
            {!loading &&
              rows.map((row) => (
                <tr
                  key={row.intent_id ?? row.created_at}
                  data-testid={`payment-row-${row.intent_id}`}
                >
                  <td>{row.intent_id ? <CopyableUuid uuid={row.intent_id} /> : '-'}</td>
                  <td className="font-mono">
                    {formatAmountMicro(row.amount_micro ?? 0, row.currency)}
                  </td>
                  <td>{row.status ? formatPaymentStatus(row.status) : '-'}</td>
                  <td>{row.provider ?? '-'}</td>
                  <td className="text-muted text-sm">
                    {row.created_at ? new Date(row.created_at).toLocaleString() : '-'}
                  </td>
                </tr>
              ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
