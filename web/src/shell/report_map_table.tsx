import type { ReportMapRow } from '@/api/types';
import { formatMapCell, reportMapRowKey } from '@/lib/report_table';
import { cn } from '@/lib/utils';
import { shellChrome } from '@/shell/shell_chrome';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
  directoryTableRevalidatingClass,
} from '@/shell/directory_table';

export type ReportMapTableProps = {
  caption?: string;
  columns: readonly string[];
  rows: readonly ReportMapRow[];
  rowKeyPrefix?: string;
  className?: string;
  revalidating?: boolean;
  formatColumn?: (column: string) => string;
};

/** Captioned or bare map-row table for ops reports and RTB overview. */
export function ReportMapTable({
  caption,
  columns,
  rows,
  rowKeyPrefix,
  className,
  revalidating = false,
  formatColumn,
}: ReportMapTableProps) {
  const keyPrefix = rowKeyPrefix ?? caption;
  const table = (
    <DirectoryTable
      className={cn(
        caption ? 'rounded-t-none' : undefined,
        directoryTableRevalidatingClass(revalidating),
        className
      )}
    >
      <TableHeader>
        <TableRow>
          {columns.map((column) => (
            <DirectoryTableHead key={column}>
              {formatColumn ? formatColumn(column) : column}
            </DirectoryTableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((row, index) => (
          <TableRow key={reportMapRowKey(row, columns, index, keyPrefix)}>
            {columns.map((column) => (
              <TableCell key={column}>{formatMapCell(row[column])}</TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </DirectoryTable>
  );

  if (!caption) {
    return table;
  }

  return (
    <div className="grid gap-0">
      <p className={cn(shellChrome.tableCaptionBandClass, 'rounded-t-md')}>
        {caption}
      </p>
      {table}
    </div>
  );
}
