import type { ReportMapRow } from '@/api/types';
import { formatMapCell, reportMapRowKey } from '@/lib/report_table';
import { cn } from '@/lib/utils';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';

export type ReportMapTableProps = {
  caption?: string;
  columns: readonly string[];
  rows: readonly ReportMapRow[];
  rowKeyPrefix?: string;
  className?: string;
};

/** Captioned or bare map-row table for ops reports and RTB overview. */
export function ReportMapTable({
  caption,
  columns,
  rows,
  rowKeyPrefix,
  className,
}: ReportMapTableProps) {
  const keyPrefix = rowKeyPrefix ?? caption;
  const table = (
    <DirectoryTable className={cn(caption ? 'rounded-t-none' : undefined, className)}>
      <TableHeader>
        <TableRow>
          {columns.map((column) => (
            <DirectoryTableHead key={column}>{column}</DirectoryTableHead>
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
      <p className="rounded-t-md border border-b-0 border-zinc-200 px-4 py-2 text-sm font-medium dark:border-zinc-800">
        {caption}
      </p>
      {table}
    </div>
  );
}
