import { useCallback, useMemo, useRef } from 'react';
import type { RtbDeal } from '../../helpers/rtb_api.js';
import { formatLocaleDateTime } from '../../helpers/format_display.js';
import { useGridRowDatasetAction } from '../../helpers/use_grid_row_action.js';
import { formatAmountMicro } from '../../helpers/money.js';
import { windowRtbRows } from '../../helpers/rtb_api.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import gridStyles from './deals_grid.module.css';
import styles from './deals_directory.module.css';

export type DealsGridProps = {
  items: RtbDeal[];
  loading: boolean;
  canWrite: boolean;
  onEdit: (deal: RtbDeal) => void;
  onDelete: (deal: RtbDeal) => void;
  onCreate: () => void;
};

function shortId(value: string | undefined): string {
  if (!value) return '-';
  if (value.length <= 12) return value;
  return `${value.slice(0, 8)}...`;
}

function buildRowView(rows: RtbDeal[]) {
  const len = rows.length;
  const keys = new Array<string>(len);
  const ids = new Array<string>(len);
  const dealIds = new Array<string>(len);
  const floors = new Array<string>(len);
  const customers = new Array<string>(len);
  const pacingLabels = new Array<string>(len);
  const seatLabels = new Array<string>(len);
  const updatedLabels = new Array<string>(len);
  const deals = new Array<RtbDeal>(len);
  for (let i = 0; i < len; i += 1) {
    const deal = rows[i];
    deals[i] = deal;
    const id = deal.id;
    keys[i] = id != null ? String(id) : deal.deal_id ?? 'deal';
    ids[i] = id != null ? String(id) : '-';
    dealIds[i] = deal.deal_id ?? '-';
    floors[i] = formatAmountMicro(deal.floor_micro, '');
    customers[i] = shortId(deal.customer_id);
    pacingLabels[i] = deal.pacing ?? '-';
    seatLabels[i] = deal.seats != null ? String(deal.seats) : '-';
    updatedLabels[i] = formatLocaleDateTime(deal.updated_at);
  }
  return { keys, ids, dealIds, floors, customers, pacingLabels, seatLabels, updatedLabels, deals, len };
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
          <span className={gridStyles.bar} />
          <span className={gridStyles.bar} />
        </div>
      ))}
    </>
  );
}

export function DealsGrid({
  items,
  loading,
  canWrite,
  onEdit,
  onDelete,
  onCreate,
}: DealsGridProps) {
  const { rows, truncated } = windowRtbRows(items);
  const rowView = useMemo(() => buildRowView(rows), [rows]);
  const dealsByKeyRef = useRef(new Map<string, RtbDeal>());
  dealsByKeyRef.current = useMemo(() => {
    const map = new Map<string, RtbDeal>();
    for (let i = 0; i < rowView.len; i += 1) {
      map.set(rowView.keys[i], rowView.deals[i]);
    }
    return map;
  }, [rowView]);
  const onRowActionHandler = useCallback(
    (key: string, action: string) => {
      const deal = dealsByKeyRef.current.get(key);
      if (!deal) return;
      if (action === 'edit') onEdit(deal);
      else if (action === 'delete') onDelete(deal);
    },
    [onDelete, onEdit]
  );
  const onRowAction = useGridRowDatasetAction(onRowActionHandler);

  return (
    <>
      {truncated ? (
        <div className={styles.windowNote}>Showing first 500 deals.</div>
      ) : null}
      <div className={gridStyles.grid} role="grid" aria-label="RTB deals">
        <div className={gridStyles.headerRow} role="row">
          <div className={gridStyles.headerCell} role="columnheader">
            ID
          </div>
          <div className={gridStyles.headerCell} role="columnheader">
            Deal ID
          </div>
          <div className={gridStyles.headerCell} role="columnheader">
            Floor
          </div>
          <div className={gridStyles.headerCell} role="columnheader">
            Customer
          </div>
          <div className={gridStyles.headerCell} role="columnheader">
            Pacing
          </div>
          <div className={gridStyles.headerCell} role="columnheader">
            Seats
          </div>
          <div className={gridStyles.headerCell} role="columnheader">
            Updated
          </div>
          <div className={gridStyles.headerCell} role="columnheader">
            Actions
          </div>
        </div>

        {loading && items.length === 0 ? <SkeletonRows /> : null}

        {!loading && items.length === 0 ? (
          <div className={gridStyles.emptyWrap}>
            <EmptyState
              message="No RTB deals yet."
              action={
                canWrite ? (
                  <Button variant="primary" size="sm" onClick={onCreate}>
                    Create deal
                  </Button>
                ) : undefined
              }
            />
          </div>
        ) : null}

        {Array.from({ length: rowView.len }, (_, index) => (
          <div key={rowView.keys[index]} className={gridStyles.dataRow} role="row">
            <div className={gridStyles.monoCell} role="gridcell">
              {rowView.ids[index]}
            </div>
            <div className={gridStyles.monoCell} role="gridcell">
              {rowView.dealIds[index]}
            </div>
            <div className={gridStyles.mutedCell} role="gridcell">
              {rowView.floors[index]}
            </div>
            <div className={gridStyles.mutedCell} role="gridcell">
              {rowView.customers[index]}
            </div>
            <div role="gridcell">{rowView.pacingLabels[index]}</div>
            <div role="gridcell">{rowView.seatLabels[index]}</div>
            <div className={gridStyles.mutedCell} role="gridcell">
              {rowView.updatedLabels[index]}
            </div>
            <div className={gridStyles.actions} role="gridcell">
              {canWrite ? (
                <>
                  <Button
                    variant="secondary"
                    size="sm"
                    data-row-id={rowView.keys[index]}
                    data-row-action="edit"
                    onClick={onRowAction}
                  >
                    Edit
                  </Button>
                  <Button
                    variant="danger"
                    size="sm"
                    data-row-id={rowView.keys[index]}
                    data-row-action="delete"
                    onClick={onRowAction}
                  >
                    Delete
                  </Button>
                </>
              ) : (
                '-'
              )}
            </div>
          </div>
        ))}
      </div>
    </>
  );
}
