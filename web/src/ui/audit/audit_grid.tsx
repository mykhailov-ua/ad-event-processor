import { useMemo } from 'react';
import type { AuditLog } from '../../helpers/audit_api.js';
import { formatLocaleDateTime } from '../../helpers/format_display.js';
import { EmptyState } from '../system/empty_state.js';
import gridStyles from './audit_grid.module.css';

export type AuditGridProps = {
  items: AuditLog[];
  loading: boolean;
};

function shortId(value: string | undefined): string {
  if (!value) return '-';
  if (value.length <= 12) return value;
  return `${value.slice(0, 8)}...`;
}

function formatTarget(row: AuditLog): string {
  const type = row.target_type ?? '';
  const id = row.target_id ?? '';
  if (!type && !id) return '-';
  if (!id) return type;
  if (!type) return id;
  return `${type} ${shortId(id)}`;
}

function buildRowView(items: AuditLog[]) {
  const len = items.length;
  const keys = new Array<string>(len);
  const times = new Array<string>(len);
  const actions = new Array<string>(len);
  const targets = new Array<string>(len);
  const admins = new Array<string>(len);
  for (let i = 0; i < len; i += 1) {
    const row = items[i];
    keys[i] = String(row.id ?? `${row.created_at}-${row.action}`);
    times[i] = formatLocaleDateTime(row.created_at);
    actions[i] = row.action ?? '-';
    targets[i] = formatTarget(row);
    admins[i] = shortId(row.admin_id);
  }
  return { keys, times, actions, targets, admins, len };
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
        </div>
      ))}
    </>
  );
}

export function AuditGrid({ items, loading }: AuditGridProps) {
  const rowView = useMemo(() => buildRowView(items), [items]);

  return (
    <div className={gridStyles.grid} role="grid" aria-label="Audit log">
      <div className={gridStyles.headerRow} role="row">
        <div className={gridStyles.headerCell} role="columnheader">
          Time
        </div>
        <div className={gridStyles.headerCell} role="columnheader">
          Action
        </div>
        <div className={gridStyles.headerCell} role="columnheader">
          Target
        </div>
        <div className={gridStyles.headerCell} role="columnheader">
          Admin
        </div>
      </div>

      {loading && items.length === 0 ? <SkeletonRows /> : null}

      {!loading && items.length === 0 ? (
        <div className={gridStyles.emptyWrap}>
          <EmptyState message="No audit entries." />
        </div>
      ) : null}

      {Array.from({ length: rowView.len }, (_, index) => (
        <div key={rowView.keys[index]} className={gridStyles.dataRow} role="row">
          <div className={gridStyles.mutedCell} role="gridcell">
            {rowView.times[index]}
          </div>
          <div role="gridcell">{rowView.actions[index]}</div>
          <div className={gridStyles.mutedCell} role="gridcell">
            {rowView.targets[index]}
          </div>
          <div className={gridStyles.monoCell} role="gridcell">
            {rowView.admins[index]}
          </div>
        </div>
      ))}
    </div>
  );
}
