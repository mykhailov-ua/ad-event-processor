import { memo, useCallback, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { touchCustomerContext } from '../../helpers/customer_context.js';
import type { Customer, CustomerSortField, CustomerSortOrder } from '../../helpers/customers_api.js';
import { formatLocaleDate } from '../../helpers/format_display.js';
import { useGridRowActivate } from '../../helpers/use_grid_row_action.js';
import { formatUsdDecimal } from '../../helpers/money.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import gridStyles from './customers_grid.module.css';

export type CustomersGridProps = {
  items: Customer[];
  loading: boolean;
  sort: CustomerSortField;
  order: CustomerSortOrder;
  onSortHeader: (field: CustomerSortField) => void;
};

function formatBalance(balance: string | undefined): string {
  if (!balance) return '-';
  return formatUsdDecimal(balance);
}

function buildRowView(items: Customer[]) {
  const len = items.length;
  const ids = new Array<string>(len);
  const names = new Array<string>(len);
  const balances = new Array<string>(len);
  const currencies = new Array<string>(len);
  const campaignCounts = new Array<string>(len);
  const createdLabels = new Array<string>(len);
  for (let i = 0; i < len; i += 1) {
    const customer = items[i];
    const id = customer.id ?? '';
    ids[i] = id;
    names[i] = customer.name ?? id;
    balances[i] = formatBalance(customer.balance);
    currencies[i] = customer.currency ?? 'USD';
    campaignCounts[i] = String(customer.active_campaigns ?? 0);
    createdLabels[i] = formatLocaleDate(customer.created_at);
  }
  return { ids, names, balances, currencies, campaignCounts, createdLabels, len };
}

function sortIcon(active: boolean, order: CustomerSortOrder): string {
  if (!active) return '|';
  return order === 'asc' ? '^' : 'v';
}

function SortHeader({
  label,
  field,
  activeSort,
  activeOrder,
  sortable,
  onSort,
}: {
  label: string;
  field?: CustomerSortField;
  activeSort: CustomerSortField;
  activeOrder: CustomerSortOrder;
  sortable: boolean;
  onSort: (field: CustomerSortField) => void;
}) {
  if (!sortable || !field) {
    return <div className={gridStyles.headerStatic}>{label}</div>;
  }
  const active = activeSort === field;
  return (
    <button
      type="button"
      className={[gridStyles.sortButton, active ? gridStyles.sortButtonActive : ''].filter(Boolean).join(' ')}
      aria-sort={active ? (activeOrder === 'asc' ? 'ascending' : 'descending') : 'none'}
      onClick={() => onSort(field)}
    >
      <span>{label}</span>
      <span className={gridStyles.sortIcon} aria-hidden="true">
        {sortIcon(active, activeOrder)}
      </span>
    </button>
  );
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
        </div>
      ))}
    </>
  );
}

type CustomerGridRowProps = {
  id: string;
  name: string;
  balance: string;
  currency: string;
  campaignCount: string;
  createdLabel: string;
  onActivate: ReturnType<typeof useGridRowActivate>;
};

const CustomerGridRow = memo(function CustomerGridRow({
  id,
  name,
  balance,
  currency,
  campaignCount,
  createdLabel,
  onActivate,
}: CustomerGridRowProps) {
  return (
    <div
      data-row-id={id}
      className={[gridStyles.dataRow, gridStyles.dataRowClickable].join(' ')}
      role="row"
      tabIndex={0}
      onClick={onActivate.onClick}
      onKeyDown={onActivate.onKeyDown}
    >
      <div className={gridStyles.nameCell} role="gridcell">
        {name}
      </div>
      <div className={[gridStyles.monoCell].join(' ')} role="gridcell">
        {balance}
      </div>
      <div role="gridcell">{currency}</div>
      <div role="gridcell">{campaignCount}</div>
      <div className={gridStyles.mutedCell} role="gridcell">
        {createdLabel}
      </div>
    </div>
  );
});

export function CustomersGrid({ items, loading, sort, order, onSortHeader }: CustomersGridProps) {
  const navigate = useNavigate();
  const rowView = useMemo(() => buildRowView(items), [items]);
  const openCustomer = useCallback(
    (id: string) => {
      touchCustomerContext(id);
      navigate(`/customers/${id}`);
    },
    [navigate]
  );
  const rowActivate = useGridRowActivate(openCustomer);

  return (
    <div className={gridStyles.grid} role="grid" aria-label="Customers">
      <div className={gridStyles.headerRow} role="row">
        <SortHeader
          label="Name"
          field="name"
          activeSort={sort}
          activeOrder={order}
          sortable
          onSort={onSortHeader}
        />
        <SortHeader
          label="Balance"
          activeSort={sort}
          activeOrder={order}
          sortable={false}
          onSort={onSortHeader}
        />
        <SortHeader
          label="Currency"
          activeSort={sort}
          activeOrder={order}
          sortable={false}
          onSort={onSortHeader}
        />
        <SortHeader
          label="Campaigns"
          activeSort={sort}
          activeOrder={order}
          sortable={false}
          onSort={onSortHeader}
        />
        <SortHeader
          label="Created"
          field="created_at"
          activeSort={sort}
          activeOrder={order}
          sortable
          onSort={onSortHeader}
        />
      </div>

      {loading && items.length === 0 ? <SkeletonRows /> : null}

      {!loading && items.length === 0 ? (
        <div className={gridStyles.emptyWrap}>
          <EmptyState
            message="No customers yet."
            action={
              <Button variant="secondary" size="sm" onClick={() => navigate('/billing')}>
                Open billing
              </Button>
            }
          />
        </div>
      ) : null}

      {Array.from({ length: rowView.len }, (_, index) => (
        <CustomerGridRow
          key={rowView.ids[index]}
          id={rowView.ids[index]}
          name={rowView.names[index]}
          balance={rowView.balances[index]}
          currency={rowView.currencies[index]}
          campaignCount={rowView.campaignCounts[index]}
          createdLabel={rowView.createdLabels[index]}
          onActivate={rowActivate}
        />
      ))}
    </div>
  );
}
