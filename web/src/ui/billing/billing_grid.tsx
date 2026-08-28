import { memo, useCallback, useMemo, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { touchCustomerContext } from '../../helpers/customer_context.js';
import type { Invoice } from '../../helpers/billing_api.js';
import { useGridRowActivate } from '../../helpers/use_grid_row_action.js';
import { formatAmountMicro } from '../../helpers/money.js';
import { EmptyState } from '../system/empty_state.js';
import gridStyles from './billing_grid.module.css';

export type BillingGridProps = {
  items: Invoice[];
  loading: boolean;
};

function shortId(value: string | undefined): string {
  if (!value) return '-';
  if (value.length <= 12) return value;
  return `${value.slice(0, 8)}...`;
}

function displayStatus(status: string | undefined): string {
  if (!status) return '-';
  return status.replace(/_/g, ' ');
}

function buildRowView(items: Invoice[]) {
  const len = items.length;
  const ids = new Array<string>(len);
  const customerIds = new Array<string>(len);
  const customerLabels = new Array<string>(len);
  const months = new Array<string>(len);
  const totals = new Array<string>(len);
  const statuses = new Array<string>(len);
  const currencies = new Array<string>(len);
  for (let i = 0; i < len; i += 1) {
    const invoice = items[i];
    const id = invoice.id ?? '';
    ids[i] = id;
    customerIds[i] = invoice.customer_id ?? '';
    customerLabels[i] = shortId(invoice.customer_id);
    months[i] = invoice.billing_month ?? '-';
    totals[i] = formatAmountMicro(invoice.total_micro, invoice.currency ?? 'USD');
    statuses[i] = displayStatus(invoice.status);
    currencies[i] = invoice.currency ?? 'USD';
  }
  return { ids, customerIds, customerLabels, months, totals, statuses, currencies, len };
}

function SkeletonRows() {
  return (
    <>
      {Array.from({ length: 5 }, (_, index) => (
        <div key={`skel-${index}`} className={[gridStyles.dataRow, gridStyles.skeletonRow].join(' ')}>
          <span className={gridStyles.bar} />
          <span className={gridStyles.bar} />
          <span className={gridStyles.bar} />
          <span className={gridStyles.bar} />
          <span className={gridStyles.bar} />
          <span className={gridStyles.bar} />
        </div>
      ))}
    </>
  );
}

type BillingGridRowProps = {
  id: string;
  customerLabel: string;
  month: string;
  total: string;
  status: string;
  currency: string;
  customerId: string;
  onActivate: ReturnType<typeof useGridRowActivate>;
};

const BillingGridRow = memo(function BillingGridRow({
  id,
  customerLabel,
  month,
  total,
  status,
  currency,
  customerId,
  onActivate,
}: BillingGridRowProps) {
  return (
    <div
      data-row-id={id}
      data-customer-id={customerId || undefined}
      className={[gridStyles.dataRow, gridStyles.dataRowClickable].join(' ')}
      role="row"
      tabIndex={0}
      onClick={onActivate.onClick}
      onKeyDown={onActivate.onKeyDown}
    >
      <div className={gridStyles.monoCell} role="gridcell">
        {shortId(id)}
      </div>
      <div className={gridStyles.mutedCell} role="gridcell">
        {customerLabel}
      </div>
      <div role="gridcell">{month}</div>
      <div className={gridStyles.mutedCell} role="gridcell">
        {total}
      </div>
      <div role="gridcell">{status}</div>
      <div role="gridcell">{currency}</div>
    </div>
  );
});

export function BillingGrid({ items, loading }: BillingGridProps) {
  const navigate = useNavigate();
  const rowView = useMemo(() => buildRowView(items), [items]);
  const customerByInvoiceRef = useRef(new Map<string, string>());
  customerByInvoiceRef.current = useMemo(() => {
    const map = new Map<string, string>();
    for (let i = 0; i < rowView.len; i += 1) {
      map.set(rowView.ids[i], rowView.customerIds[i]);
    }
    return map;
  }, [rowView]);
  const openInvoice = useCallback(
    (id: string) => {
      const customerId = customerByInvoiceRef.current.get(id);
      if (customerId) touchCustomerContext(customerId);
      navigate(`/billing/invoices/${id}`);
    },
    [navigate]
  );
  const rowActivate = useGridRowActivate(openInvoice);

  return (
    <div className={gridStyles.grid} role="grid" aria-label="Invoices">
      <div className={gridStyles.headerRow} role="row">
        <div className={gridStyles.headerCell} role="columnheader">
          ID
        </div>
        <div className={gridStyles.headerCell} role="columnheader">
          Customer
        </div>
        <div className={gridStyles.headerCell} role="columnheader">
          Month
        </div>
        <div className={gridStyles.headerCell} role="columnheader">
          Total
        </div>
        <div className={gridStyles.headerCell} role="columnheader">
          Status
        </div>
        <div className={gridStyles.headerCell} role="columnheader">
          Currency
        </div>
      </div>

      {loading && items.length === 0 ? <SkeletonRows /> : null}

      {!loading && items.length === 0 ? (
        <div className={gridStyles.emptyWrap}>
          <EmptyState message="No invoices match these filters." />
        </div>
      ) : null}

      {Array.from({ length: rowView.len }, (_, index) => (
        <BillingGridRow
          key={rowView.ids[index]}
          id={rowView.ids[index]}
          customerLabel={rowView.customerLabels[index]}
          month={rowView.months[index]}
          total={rowView.totals[index]}
          status={rowView.statuses[index]}
          currency={rowView.currencies[index]}
          customerId={rowView.customerIds[index]}
          onActivate={rowActivate}
        />
      ))}
    </div>
  );
}
