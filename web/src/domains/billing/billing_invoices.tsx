import { Link } from 'react-router-dom';

import { FilterApplyButton } from '@/components/system/action_buttons';
import { PaginationPrevNext } from '@/components/system/pagination_prev_next';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { EmptyState } from '@/components/system/empty_state';
import { ErrorBlock } from '@/components/system/error_block';
import { PageSkeleton } from '@/components/system/page_skeleton';
import type { Invoice } from '@/api/types';
import { displayMicro } from '@/lib/display';

export type InvoiceStatusFilter = '' | 'draft' | 'finalized' | 'void';

export type BillingInvoicesProps = {
  items: Invoice[];
  total: number;
  limit: number;
  offset: number;
  draftMonth: string;
  draftStatus: InvoiceStatusFilter;
  fetching: boolean;
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
    return <ErrorBlock title="Could not load invoices" message={error.message} />;
  }

  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;

  return (
    <section className="grid gap-6">
      <h2 className="text-base font-semibold">Invoices</h2>

      <form
        className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4"
        onSubmit={(event) => {
          event.preventDefault();
          onApplyFilters();
        }}
      >
        <div className="grid gap-2">
          <Label htmlFor="billing-month">Month</Label>
          <Input
            id="billing-month"
            type="month"
            className="h-9 text-sm"
            value={draftMonth}
            onChange={(event) => onDraftMonthChange(event.target.value)}
          />
        </div>

        <div className="grid gap-2">
          <Label htmlFor="billing-status">Status</Label>
          <Select
            value={draftStatus || 'all'}
            onValueChange={(value) =>
              onDraftStatusChange(value === 'all' ? '' : (value as InvoiceStatusFilter))
            }
          >
            <SelectTrigger id="billing-status" className="h-9 w-full text-sm">
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

        <PaginationPrevNext
          canGoPrev={canGoPrev}
          canGoNext={canGoNext}
          disabled={fetching}
          onPrev={() => onPageChange(Math.max(0, offset - limit))}
          onNext={() => onPageChange(offset + limit)}
        />
      </form>

      {items.length === 0 ? (
        <EmptyState title="No invoices" description="No invoices match the current filters." />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Invoice</TableHead>
                <TableHead>Billing month</TableHead>
                <TableHead>Customer</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Subtotal (micro)</TableHead>
                <TableHead>Tax (micro)</TableHead>
                <TableHead>Total (micro)</TableHead>
                <TableHead>Currency</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((invoice) => (
                <TableRow key={invoice.id}>
                  <TableCell>
                    {invoice.id ? (
                      <Link className="font-mono text-xs hover:underline" to={`/billing/invoices/${invoice.id}`}>
                        {invoice.id}
                      </Link>
                    ) : (
                      ''
                    )}
                  </TableCell>
                  <TableCell>{invoice.billing_month}</TableCell>
                  <TableCell className="font-mono text-xs">{invoice.customer_id}</TableCell>
                  <TableCell>{invoice.status ?? ''}</TableCell>
                  <TableCell className="tabular-nums">{displayMicro(invoice.subtotal_micro, invoice.subtotal_micro_display)}</TableCell>
                  <TableCell className="tabular-nums">{displayMicro(invoice.tax_micro, invoice.tax_micro_display)}</TableCell>
                  <TableCell className="tabular-nums">{displayMicro(invoice.total_micro, invoice.total_micro_display)}</TableCell>
                  <TableCell>{invoice.currency}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </section>
  );
}
