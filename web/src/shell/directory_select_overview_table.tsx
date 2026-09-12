import { useMemo, useState, type ReactNode } from 'react';

import {
  ControlPlaneSelectTable,
  type ControlPlaneSelectRow,
  type ControlPlaneSelectTableProps,
} from '@/shell/control_plane_select_table';
import {
  DirectoryOverviewDialog,
  type DirectoryOverviewField,
} from '@/shell/directory_overview_dialog';

export type DirectorySelectOverviewTableProps<T> = {
  recordById: Map<string, T>;
  buildOverviewFields: (record: T) => DirectoryOverviewField[];
  overviewTitle: (record: T) => ReactNode;
  overviewFooter?: (record: T) => ReactNode;
  renderActions?: (row: ControlPlaneSelectRow, record: T, openOverview: () => void) => ReactNode;
  actionsAriaLabel?: (row: ControlPlaneSelectRow, record: T) => string;
} & Omit<ControlPlaneSelectTableProps, 'renderActions'>;

export function DirectorySelectOverviewTable<T>({
  recordById,
  buildOverviewFields,
  overviewTitle,
  overviewFooter,
  renderActions,
  actionsAriaLabel,
  rows,
  disabled,
  ...tableProps
}: DirectorySelectOverviewTableProps<T>) {
  const [overviewId, setOverviewId] = useState<string | null>(null);
  const overviewRecord = overviewId ? recordById.get(overviewId) : undefined;

  const overviewFields = useMemo(
    () => (overviewRecord ? buildOverviewFields(overviewRecord) : []),
    [buildOverviewFields, overviewRecord]
  );

  return (
    <>
      <ControlPlaneSelectTable
        disabled={disabled}
        onNameClick={(row) => {
          if (recordById.has(row.id)) {
            setOverviewId(row.id);
          }
        }}
        renderActions={
          renderActions
            ? (row) => {
                const record = recordById.get(row.id);
                if (!record) {
                  return null;
                }
                const openOverview = () => setOverviewId(row.id);
                return renderActions(row, record, openOverview);
              }
            : undefined
        }
        rows={rows}
        {...tableProps}
      />
      {overviewRecord ? (
        <DirectoryOverviewDialog
          fields={overviewFields}
          footer={overviewFooter?.(overviewRecord)}
          open={overviewId != null}
          title={overviewTitle(overviewRecord)}
          onOpenChange={(open) => {
            if (!open) {
              setOverviewId(null);
            }
          }}
        />
      ) : null}
    </>
  );
}

/** Build Map index for DirectorySelectOverviewTable from list rows. */
export function directoryRecordMap<T>(
  items: T[] | undefined,
  getId: (item: T) => string | undefined | null
): Map<string, T> {
  const map = new Map<string, T>();
  for (const item of items ?? []) {
    const id = getId(item);
    if (id) {
      map.set(id, item);
    }
  }
  return map;
}

/** Indexed variant when row keys need list position. */
export function directoryRecordMapIndexed<T>(
  items: T[] | undefined,
  getId: (item: T, index: number) => string | undefined | null
): Map<string, T> {
  const map = new Map<string, T>();
  for (const [index, item] of (items ?? []).entries()) {
    const id = getId(item, index);
    if (id) {
      map.set(id, item);
    }
  }
  return map;
}

/** Indexed variant when row keys need list position. */
export function directoryOperateRowsIndexed<T>(
  items: T[] | undefined,
  getId: (item: T, index: number) => string | undefined | null,
  getLabel: (item: T) => ReactNode
): ControlPlaneSelectRow[] {
  const rows: ControlPlaneSelectRow[] = [];
  for (const [index, item] of (items ?? []).entries()) {
    const id = getId(item, index);
    if (!id) {
      continue;
    }
    rows.push({ id, label: getLabel(item) });
  }
  return rows;
}

/** Map list items to ControlPlaneSelectTable rows. */
export function directoryOperateRows<T>(
  items: T[] | undefined,
  getId: (item: T) => string | undefined | null,
  getLabel: (item: T) => ReactNode
): ControlPlaneSelectRow[] {
  const rows: ControlPlaneSelectRow[] = [];
  for (const item of items ?? []) {
    const id = getId(item);
    if (!id) {
      continue;
    }
    rows.push({ id, label: getLabel(item) });
  }
  return rows;
}
