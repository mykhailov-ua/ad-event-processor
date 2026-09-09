import type { ReactNode } from 'react';

import type { DirectoryOverviewField } from '@/shell/directory_overview_dialog';
import {
  directoryOperateRowsIndexed,
  directoryRecordMapIndexed,
} from '@/shell/directory_select_overview_table';

export type ReportDirectoryColumn<Row> = {
  id: string;
  label: string;
  align?: 'end';
  cell: (row: Row) => ReactNode;
};

export function buildReportColumnOverviewFields<Row>(
  row: Row,
  columns: ReportDirectoryColumn<Row>[],
  options?: { skipColumnIds?: string[] }
): DirectoryOverviewField[] {
  const skip = new Set(options?.skipColumnIds ?? []);
  return columns
    .filter((column) => !skip.has(column.id))
    .map((column) => ({
      label: column.label,
      value: column.cell(row),
    }));
}

export function reportRowLabelFromColumn<Row>(
  row: Row,
  column: ReportDirectoryColumn<Row> | undefined,
  fallback: string
): ReactNode {
  if (!column) {
    return fallback;
  }
  const value = column.cell(row);
  if (value == null || value === '') {
    return fallback;
  }
  return value;
}

export { directoryOperateRowsIndexed, directoryRecordMapIndexed };
