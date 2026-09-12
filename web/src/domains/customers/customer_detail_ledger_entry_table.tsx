import type { BalanceLedgerEntry } from '@/api/types';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { displayTimestamp } from '@/lib/display';
import { adminTypography } from '@/lib/admin_kit';

export type CustomerDetailLedgerEntryTableProps = {
  items: BalanceLedgerEntry[];
};

export function CustomerDetailLedgerEntryTable({ items }: CustomerDetailLedgerEntryTableProps) {
  return (
    <DirectoryTable nested>
      <TableHeader>
        <TableRow>
          <DirectoryTableHead>ID</DirectoryTableHead>
          <DirectoryTableHead>Type</DirectoryTableHead>
          <DirectoryTableHead align="end">Amount</DirectoryTableHead>
          <DirectoryTableHead>Campaign</DirectoryTableHead>
          <DirectoryTableHead>Created</DirectoryTableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((row) => (
          <TableRow key={row.id ?? `${row.created_at}-${row.type}`}>
            <TableCell className="tabular-nums">{row.id ?? ''}</TableCell>
            <TableCell>{row.type ?? ''}</TableCell>
            <TableCell className="text-right tabular-nums">{row.amount ?? ''}</TableCell>
            <TableCell className={adminTypography.monoData}>{row.campaign_id ?? ''}</TableCell>
            <TableCell>{displayTimestamp(row.created_at)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </DirectoryTable>
  );
}
