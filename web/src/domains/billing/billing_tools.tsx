import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { MonthPicker } from '@/components/ui/datetime_picker';
import {
  DirectoryFilterForm,
  FilterField,
  FilterPanel,
  INLINE_FILTER_ACTION_GRID_CLASS,
} from '@/shell/filter_panel';
import { billingPanelError } from '@/domains/billing/billing_nav';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import type { BillingInvariant, InvoicePreview } from '@/api/types';
import { displayMicro } from '@/lib/display';

export type BillingInvariantPanelProps = {
  draftCustomerId: string;
  invariant: BillingInvariant | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  onDraftCustomerIdChange: (value: string) => void;
  onCheck: () => void;
};

export function BillingInvariantPanel({
  draftCustomerId,
  invariant,
  fetching,
  error,
  hasSnapshot,
  onDraftCustomerIdChange,
  onCheck,
}: BillingInvariantPanelProps) {
  return (
    <FilterPanel className="w-full max-w-xl">
      <h2 className="text-base font-semibold">Ledger invariant</h2>
      <DirectoryFilterForm
        className={INLINE_FILTER_ACTION_GRID_CLASS}
        onSubmit={(event) => event.preventDefault()}
      >
        <FilterField htmlFor="invariant-customer-id" label="Customer ID (optional)">
          <Input
            id="invariant-customer-id"
            value={draftCustomerId}
            onChange={(event) => onDraftCustomerIdChange(event.target.value)}
          />
        </FilterField>
        <Button disabled={fetching} onClick={onCheck} type="button" variant="outline">
          {fetching ? 'Checking...' : 'Check'}
        </Button>
      </DirectoryFilterForm>
      {error && !hasSnapshot ? billingPanelError(error, 'Invariant check failed') : null}
      {invariant ? (
        <div className="flex flex-wrap items-center gap-2 text-sm">
          <Badge variant={invariant.ok ? 'default' : 'destructive'}>
            {invariant.ok ? 'OK' : 'Mismatch'}
          </Badge>
          {invariant.diff_micro != null ? (
            <span>Diff (micro): {displayMicro(invariant.diff_micro)}</span>
          ) : null}
          {invariant.balance_micro != null ? (
            <span>Balance (micro): {displayMicro(invariant.balance_micro)}</span>
          ) : null}
        </div>
      ) : null}
    </FilterPanel>
  );
}

export type BillingPreviewPanelProps = {
  draftCustomerId: string;
  draftMonth: string;
  preview: InvoicePreview | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftMonthChange: (value: string) => void;
  onPreview: () => void;
};

export function BillingPreviewPanel({
  draftCustomerId,
  draftMonth,
  preview,
  fetching,
  error,
  hasSnapshot,
  onDraftCustomerIdChange,
  onDraftMonthChange,
  onPreview,
}: BillingPreviewPanelProps) {
  const lines = preview?.lines ?? [];

  return (
    <FilterPanel>
      <h2 className="text-base font-semibold">Invoice preview</h2>
      <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
        <div className="grid gap-2">
          <Label htmlFor="preview-customer-id">Customer ID</Label>
          <Input
            id="preview-customer-id"
            value={draftCustomerId}
            onChange={(event) => onDraftCustomerIdChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="preview-month">Billing month</Label>
          <MonthPicker id="preview-month" value={draftMonth} onChange={onDraftMonthChange} />
        </div>
        <Button
          disabled={fetching || !draftCustomerId.trim() || !draftMonth}
          onClick={onPreview}
          type="button"
        >
          {fetching ? 'Previewing...' : 'Preview'}
        </Button>
      </DirectoryFilterForm>

      {error && !hasSnapshot ? billingPanelError(error, 'Preview failed') : null}

      {preview ? (
        <div className="grid gap-4">
          <div className="grid gap-2 text-sm">
            <p className="m-0">
              Total: {displayMicro(preview.total_micro, preview.total_micro_display)}{' '}
              {preview.currency ?? ''}
            </p>
            {preview.would_skip ? <Badge variant="secondary">Would skip generation</Badge> : null}
          </div>
          {lines.length > 0 ? (
            <DirectoryTable>
              <TableHeader>
                <TableRow>
                  <DirectoryTableHead>Ledger type</DirectoryTableHead>
                  <DirectoryTableHead className="text-right">Amount (micro)</DirectoryTableHead>
                  <DirectoryTableHead className="text-right">Entries</DirectoryTableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {lines.map((line, index) => (
                  <TableRow key={`${line.ledger_type ?? 'line'}-${index}`}>
                    <TableCell>{line.ledger_type ?? ''}</TableCell>
                    <TableCell className="text-right tabular-nums">
                      {displayMicro(line.amount_micro)}
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {line.entry_count ?? ''}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </DirectoryTable>
          ) : null}
        </div>
      ) : null}
    </FilterPanel>
  );
}
