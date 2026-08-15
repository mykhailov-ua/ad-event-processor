import { useMemo, type ReactNode } from 'react';
import {
  sortRows,
  type SortState,
} from '../../ui/data_table.js';
import { Icon } from './icon.js';

export type DataTableColumn<T> = {
  key: string;
  header: string;
  sortable?: boolean;
  render: (row: T) => ReactNode;
  accessor?: (row: T) => unknown;
};

export type DataTableProps<T> = {
  columns: DataTableColumn<T>[];
  rows: T[];
  sortState: SortState;
  onSort: (key: string) => void;
  rowKey: (row: T) => string;
  caption?: string;
};

/**
 * Sortable data table with Geist data-table classes.
 */
export function DataTable<T>({
  columns,
  rows,
  sortState,
  onSort,
  rowKey,
  caption,
}: DataTableProps<T>) {
  const accessors = useMemo(() => {
    const map: Record<string, (row: T) => unknown> = {};
    for (const col of columns) {
      if (col.accessor) map[col.key] = col.accessor;
    }
    return map;
  }, [columns]);

  const sortedRows = useMemo(
    () => sortRows(rows, sortState, accessors),
    [rows, sortState, accessors],
  );

  return (
    <div className="data-table-wrap">
      <table className="data-table">
        {caption ? <caption className="sr-only">{caption}</caption> : null}
        <thead>
          <tr>
            {columns.map((col) => {
              if (!col.sortable) {
                return <th key={col.key} scope="col">{col.header}</th>;
              }
              const active = sortState.key === col.key;
              const iconName = active
                ? (sortState.dir === 'asc' ? 'chevron-up' : 'chevron-down')
                : 'arrow-up-down';
              return (
                <th
                  key={col.key}
                  scope="col"
                  className={[
                    'data-table__th--sortable',
                    active ? 'data-table__th--sorted' : '',
                  ].filter(Boolean).join(' ')}
                  aria-sort={active ? (sortState.dir === 'asc' ? 'ascending' : 'descending') : 'none'}
                  tabIndex={0}
                  onClick={() => onSort(col.key)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault();
                      onSort(col.key);
                    }
                  }}
                >
                  <span className="data-table__th-label">{col.header}</span>
                  <Icon name={iconName} size={13} className="data-table__sort-icon" />
                </th>
              );
            })}
          </tr>
        </thead>
        <tbody>
          {sortedRows.map((row) => (
            <tr key={rowKey(row)}>
              {columns.map((col) => (
                <td key={col.key}>{col.render(row)}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
