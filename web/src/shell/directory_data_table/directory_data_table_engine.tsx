import type { CSSProperties, ReactNode, RefObject } from 'react';

import { DirectoryTable, TableBody, TableFooter, TableHeader } from '@/shell/directory_table';
import {
  directoryTableWidthPx,
  proportionalDirectoryColumnWidths,
} from '@/shell/directory_data_table/layout';
import { useDirectoryTableContainerWidth } from '@/shell/directory_data_table/use_directory_table_container_width';
import { cn } from '@/lib/utils';

export type DirectoryDataTableEngineProps<Row, Col extends string> = {
  columns: readonly Col[];
  columnWidths: Readonly<Record<Col, number>>;
  rows: readonly Row[];
  rowKey: (row: Row) => string;
  fillContainer?: boolean;
  horizontalScroll?: boolean;
  columnMaxWidths?: Partial<Record<Col, number>>;
  resolveColumnMaxWidths?: (containerWidthPx: number) => Partial<Record<Col, number>>;
  footerRow?: Row;
  tableClassName?: string;
  surfaceClassName?: string;
  headerCellClassName?: string;
  bodyCellClassName?: string;
  footerCellClassName?: string;
  tableRef?: RefObject<HTMLTableElement | null>;
  colgroupRef?: RefObject<HTMLTableColElement | null>;
  isResizableColumn: (columnId: Col) => boolean;
  renderHeaderLabel: (columnId: Col) => ReactNode;
  renderResizeHandle?: (columnId: Col) => ReactNode;
  renderBodyCell: (columnId: Col, row: Row) => ReactNode;
  renderFooterCell?: (columnId: Col, row: Row) => ReactNode;
};

export function DirectoryDataTableEngine<Row, Col extends string>({
  columns,
  columnWidths,
  rows,
  rowKey,
  fillContainer = true,
  horizontalScroll = false,
  columnMaxWidths,
  resolveColumnMaxWidths,
  footerRow,
  tableClassName,
  surfaceClassName,
  headerCellClassName,
  bodyCellClassName,
  footerCellClassName,
  tableRef,
  colgroupRef,
  isResizableColumn,
  renderHeaderLabel,
  renderResizeHandle,
  renderBodyCell,
  renderFooterCell,
}: DirectoryDataTableEngineProps<Row, Col>) {
  const { hostRef, containerWidthPx } = useDirectoryTableContainerWidth();

  const maxWidths =
    resolveColumnMaxWidths?.(containerWidthPx) ?? columnMaxWidths ?? ({} as Partial<Record<Col, number>>);

  const layoutWidths =
    fillContainer && containerWidthPx > 0
      ? proportionalDirectoryColumnWidths(columns, columnWidths, containerWidthPx, maxWidths)
      : columnWidths;

  const tableWidthPx = directoryTableWidthPx(columns, layoutWidths);
  const useFillWidth = fillContainer && containerWidthPx > 0 && tableWidthPx <= containerWidthPx;

  const tableStyle: CSSProperties = useFillWidth
    ? { width: '100%', minWidth: '100%', tableLayout: 'fixed' }
    : {
        width: `${tableWidthPx}px`,
        minWidth: `${tableWidthPx}px`,
        tableLayout: 'fixed',
      };

  return (
    <div ref={hostRef} className="min-w-0">
      <DirectoryTable
        className={cn(surfaceClassName, 'rounded-none border-0 shadow-none')}
        fixedLayout
        horizontalScroll={horizontalScroll || !useFillWidth}
        nested
        tableClassName={tableClassName}
        tableRef={tableRef}
        tableStyle={tableStyle}
      >
        <colgroup ref={colgroupRef}>
          {columns.map((columnId) => (
            <col key={columnId} style={{ width: `${layoutWidths[columnId]}px` }} />
          ))}
        </colgroup>
        <TableHeader>
          <tr>
            {columns.map((columnId) => (
              <th key={columnId} className={headerCellClassName}>
                {renderHeaderLabel(columnId)}
                {isResizableColumn(columnId) ? renderResizeHandle?.(columnId) : null}
              </th>
            ))}
          </tr>
        </TableHeader>
        <TableBody>
          {rows.map((row) => (
            <tr key={rowKey(row)}>
              {columns.map((columnId) => (
                <td key={columnId} className={bodyCellClassName}>
                  {renderBodyCell(columnId, row)}
                </td>
              ))}
            </tr>
          ))}
        </TableBody>
        {footerRow && renderFooterCell ? (
          <TableFooter>
            <tr>
              {columns.map((columnId) => (
                <td key={columnId} className={footerCellClassName}>
                  {renderFooterCell(columnId, footerRow)}
                </td>
              ))}
            </tr>
          </TableFooter>
        ) : null}
      </DirectoryTable>
    </div>
  );
}
