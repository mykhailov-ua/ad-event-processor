import { memo, useCallback, useMemo, type ChangeEvent, type MouseEvent } from 'react';
import { Link } from 'react-router-dom';
import { touchCustomerContext } from '../../helpers/customer_context.js';
import type { Campaign, CampaignSortField, CampaignSortOrder } from '../../helpers/campaigns_api.js';
import { formatLocaleDateTime } from '../../helpers/format_display.js';
import { useGridRowCheckboxChange } from '../../helpers/use_grid_row_action.js';
import { formatUsdDecimal } from '../../helpers/money.js';
import { EmptyState } from '../system/empty_state.js';
import gridStyles from './campaigns_grid.module.css';

export type CampaignsGridProps = {
  items: Campaign[];
  loading: boolean;
  sort: CampaignSortField;
  order: CampaignSortOrder;
  selectedIds: Set<string>;
  canBulk: boolean;
  customerScoped: boolean;
  onSortHeader: (field: CampaignSortField) => void;
  onToggleRow: (id: string, checked: boolean) => void;
  onToggleAll: (checked: boolean, ids: string[]) => void;
};

function displayStatus(status: string | undefined): string {
  if (!status) return '-';
  return status.replace(/_/g, ' ');
}

function statusClass(status: string | undefined): string {
  if (!status) return gridStyles.statusChip;
  const normalized = status.toLowerCase();
  if (normalized === 'active' || normalized === 'running') {
    return [gridStyles.statusChip, gridStyles.statusActive].join(' ');
  }
  if (normalized === 'paused') {
    return [gridStyles.statusChip, gridStyles.statusPaused].join(' ');
  }
  return gridStyles.statusChip;
}

function formatMoney(value: string | undefined): string {
  if (!value) return '-';
  return formatUsdDecimal(value);
}

function buildRowView(items: Campaign[]) {
  const len = items.length;
  const ids = new Array<string>(len);
  const names = new Array<string>(len);
  const statusLabels = new Array<string>(len);
  const statusClasses = new Array<string>(len);
  const budgetLabels = new Array<string>(len);
  const pacingLabels = new Array<string>(len);
  const spendLabels = new Array<string>(len);
  const updatedLabels = new Array<string>(len);
  const customerIds = new Array<string>(len);
  for (let i = 0; i < len; i += 1) {
    const campaign = items[i];
    const id = campaign.id ?? '';
    ids[i] = id;
    names[i] = campaign.name ?? id;
    statusLabels[i] = displayStatus(campaign.status);
    statusClasses[i] = statusClass(campaign.status);
    budgetLabels[i] = formatMoney(campaign.budget_limit);
    pacingLabels[i] = campaign.pacing_mode ?? '-';
    spendLabels[i] = formatMoney(campaign.current_spend);
    updatedLabels[i] = formatLocaleDateTime(campaign.updated_at);
    customerIds[i] = campaign.customer_id ?? '';
  }
  return { ids, names, statusLabels, statusClasses, budgetLabels, pacingLabels, spendLabels, updatedLabels, customerIds, len };
}

function sortIcon(active: boolean, order: CampaignSortOrder): string {
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
  field?: CampaignSortField;
  activeSort: CampaignSortField;
  activeOrder: CampaignSortOrder;
  sortable: boolean;
  onSort: (field: CampaignSortField) => void;
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
          <span className={gridStyles.bar} />
        </div>
      ))}
    </>
  );
}

type CampaignGridRowProps = {
  id: string;
  name: string;
  statusLabel: string;
  statusClassName: string;
  budgetLabel: string;
  pacingLabel: string;
  spendLabel: string;
  updatedLabel: string;
  customerId: string;
  checked: boolean;
  canBulk: boolean;
  customerScoped: boolean;
  onRowCheck: (event: ChangeEvent<HTMLInputElement>) => void;
  onCustomerTouch: (event: MouseEvent<HTMLAnchorElement>) => void;
};

const CampaignGridRow = memo(function CampaignGridRow({
  id,
  name,
  statusLabel,
  statusClassName,
  budgetLabel,
  pacingLabel,
  spendLabel,
  updatedLabel,
  customerId,
  checked,
  canBulk,
  customerScoped,
  onRowCheck,
  onCustomerTouch,
}: CampaignGridRowProps) {
  return (
    <div className={gridStyles.dataRow} role="row">
      <div className={gridStyles.checkCell} role="gridcell">
        {canBulk ? (
          <input
            type="checkbox"
            data-row-id={id}
            aria-label={`Select ${name}`}
            checked={checked}
            onChange={onRowCheck}
          />
        ) : null}
      </div>
      <div className={gridStyles.nameCell} role="gridcell">
        <Link
          to={`/campaigns/${id}`}
          data-customer-id={customerScoped ? customerId : undefined}
          onClick={customerScoped ? onCustomerTouch : undefined}
        >
          {name}
        </Link>
      </div>
      <div role="gridcell">
        <span className={statusClassName}>{statusLabel}</span>
      </div>
      <div className={gridStyles.mutedCell} role="gridcell">
        {budgetLabel}
      </div>
      <div role="gridcell">{pacingLabel}</div>
      <div className={gridStyles.mutedCell} role="gridcell">
        {spendLabel}
      </div>
      <div className={gridStyles.mutedCell} role="gridcell">
        {updatedLabel}
      </div>
    </div>
  );
});

export function CampaignsGrid({
  items,
  loading,
  sort,
  order,
  selectedIds,
  canBulk,
  customerScoped,
  onSortHeader,
  onToggleRow,
  onToggleAll,
}: CampaignsGridProps) {
  const rowIds = useMemo(() => {
    const ids: string[] = [];
    for (let i = 0; i < items.length; i += 1) {
      const id = items[i].id ?? '';
      if (id) ids.push(id);
    }
    return ids;
  }, [items]);
  const rowView = useMemo(() => buildRowView(items), [items]);
  const allSelected = rowIds.length > 0 && rowIds.every((id) => selectedIds.has(id));
  const onRowCheck = useGridRowCheckboxChange(onToggleRow);
  const onCustomerTouch = useCallback((event: MouseEvent<HTMLAnchorElement>) => {
    const customerId = event.currentTarget.dataset.customerId;
    if (customerId) touchCustomerContext(customerId);
  }, []);

  return (
    <div className={gridStyles.grid} role="grid" aria-label="Campaigns">
      <div className={gridStyles.headerRow} role="row">
        {canBulk ? (
          <div className={gridStyles.checkCell} role="columnheader">
            <input
              type="checkbox"
              aria-label="Select all campaigns on page"
              checked={allSelected}
              onChange={(event) => onToggleAll(event.target.checked, rowIds)}
            />
          </div>
        ) : (
          <div className={gridStyles.checkCell} role="columnheader" aria-hidden="true" />
        )}
        <SortHeader
          label="Name"
          field="name"
          activeSort={sort}
          activeOrder={order}
          sortable
          onSort={onSortHeader}
        />
        <SortHeader
          label="Status"
          activeSort={sort}
          activeOrder={order}
          sortable={false}
          onSort={onSortHeader}
        />
        <SortHeader
          label="Budget"
          activeSort={sort}
          activeOrder={order}
          sortable={false}
          onSort={onSortHeader}
        />
        <SortHeader
          label="Pacing"
          activeSort={sort}
          activeOrder={order}
          sortable={false}
          onSort={onSortHeader}
        />
        <SortHeader
          label="Spend"
          field="spend"
          activeSort={sort}
          activeOrder={order}
          sortable
          onSort={onSortHeader}
        />
        <SortHeader
          label="Updated"
          field="updated_at"
          activeSort={sort}
          activeOrder={order}
          sortable
          onSort={onSortHeader}
        />
      </div>

      {loading && items.length === 0 ? <SkeletonRows /> : null}

      {!loading && items.length === 0 ? (
        <div className={gridStyles.emptyWrap}>
          <EmptyState message="No campaigns match these filters." />
        </div>
      ) : null}

      {Array.from({ length: rowView.len }, (_, index) => (
        <CampaignGridRow
          key={rowView.ids[index]}
          id={rowView.ids[index]}
          name={rowView.names[index]}
          statusLabel={rowView.statusLabels[index]}
          statusClassName={rowView.statusClasses[index]}
          budgetLabel={rowView.budgetLabels[index]}
          pacingLabel={rowView.pacingLabels[index]}
          spendLabel={rowView.spendLabels[index]}
          updatedLabel={rowView.updatedLabels[index]}
          customerId={rowView.customerIds[index]}
          checked={selectedIds.has(rowView.ids[index])}
          canBulk={canBulk}
          customerScoped={customerScoped}
          onRowCheck={onRowCheck}
          onCustomerTouch={onCustomerTouch}
        />
      ))}
    </div>
  );
}
