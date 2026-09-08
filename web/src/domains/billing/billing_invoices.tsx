import { Link } from 'react-router-dom';

import { FilterApplyButton } from '@/shell/action_buttons';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import { DirectoryFilterForm, FilterPanel } from '@/shell/filter_panel';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
  directoryTableRevalidatingClass,
} from '@/shell/directory_table';
import { MonthPicker } from '@/components/ui/datetime_picker';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import type { Invoice } from '@/api/types';
import { billingPanelError } from '@/domains/billing/billing_nav';
import { displayMicro } from '@/lib/display';

export type InvoiceStatusFilter = '' | 'draft' | 'finalized' | 'void';

export type BillingInvoicesProps = {
  items?: Invoice[];
  total: number;
  limit: number;
  offset: number;
  draftMonth: string;
  draftStatus: InvoiceStatusFilter;
  fetching: boolean;
  listRevalidating?: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  onDraftMonthChange: (month: string) => void;
  onDraftStatusChange: (status: InvoiceStatusFilter) => void;
  onApplyFilters: () => void;
  onPageChange: (nextOffset: number) => void;
};

export function BillingInvoices({
  items,
  total,
  limit,
  offset,
  draftMonth,
  draftStatus,
  fetching,
  listRevalidating = false,
  error,
  hasSnapshot,
  onDraftMonthChange,
  onDraftStatusChange,
  onApplyFilters,
  onPageChange,
}: BillingInvoicesProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return billingPanelError(error, 'Could not load invoices');
  }

  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;

  return (
    <section className="grid gap-6">
      <h2 className="text-base font-semibold">Invoices</h2>

      <FilterPanel>
        <DirectoryFilterForm
          layout="auto-fill"
          onSubmit={(event) => {
            event.preventDefault();
            onApplyFilters();
          }}
        >
          <div className="grid gap-2">
            <Label htmlFor="billing-month">Month</Label>
            <MonthPicker id="billing-month" value={draftMonth} onChange={onDraftMonthChange} />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="billing-status">Status</Label>
            <Select
              value={draftStatus || 'all'}
              onValueChange={(value) =>
                onDraftStatusChange(value === 'all' ? '' : (value as InvoiceStatusFilter))
              }
            >
              <SelectTrigger id="billing-status" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All</SelectItem>
                <SelectItem value="draft">Draft</SelectItem>
                <SelectItem value="finalized">Finalized</SelectItem>
                <SelectItem value="void">Void</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <FilterApplyButton disabled={fetching}>Apply</FilterApplyButton>
        </DirectoryFilterForm>
      </FilterPanel>

      <DirectoryPaginationFooter
        canGoNext={canGoNext}
        canGoPrev={canGoPrev}
        disabled={fetching}
        onNext={() => onPageChange(offset + limit)}
        onPrev={() => onPageChange(Math.max(0, offset - limit))}
      />

      {(items ?? []).length === 0 ? (
        <EmptyState title="No invoices" description="No invoices match the current filters." />
      ) : (
        <DirectoryTable className={directoryTableRevalidatingClass(listRevalidating)}>
          <TableHeader>
            <TableRow>
              <DirectoryTableHead>Invoice</DirectoryTableHead>
              <DirectoryTableHead>Billing month</DirectoryTableHead>
              <DirectoryTableHead>Customer</DirectoryTableHead>
              <DirectoryTableHead>Status</DirectoryTableHead>
              <DirectoryTableHead>Subtotal (micro)</DirectoryTableHead>
              <DirectoryTableHead>Tax (micro)</DirectoryTableHead>
              <DirectoryTableHead>Total (micro)</DirectoryTableHead>
              <DirectoryTableHead>Currency</DirectoryTableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(items ?? []).map((invoice) => (
              <TableRow key={invoice.id}>
                <TableCell>
                  {invoice.id ? (
                    <Link
                      className="font-mono text-xs hover:underline"
                      to={`/billing/invoices/${invoice.id}`}
                    >
                      {invoice.id}
                    </Link>
                  ) : (
                    ''
                  )}
                </TableCell>
                <TableCell>{invoice.billing_month}</TableCell>
                <TableCell className="font-mono text-xs">{invoice.customer_id}</TableCell>
                <TableCell>{invoice.status ?? ''}</TableCell>
                <TableCell className="tabular-nums">
                  {displayMicro(invoice.subtotal_micro, invoice.subtotal_micro_display)}
                </TableCell>
                <TableCell className="tabular-nums">
                  {displayMicro(invoice.tax_micro, invoice.tax_micro_display)}
                </TableCell>
                <TableCell className="tabular-nums">
                  {displayMicro(invoice.total_micro, invoice.total_micro_display)}
                </TableCell>
                <TableCell>{invoice.currency}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </DirectoryTable>
      )}
    </section>
  );
}
