import { Link } from 'react-router-dom';

import {
  FilterApplyButton,
  PrimaryActionButton,
  SecondaryActionButton,
} from '@/shell/action_buttons';
import { PageLayout } from '@/shell/page_layout';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import {
  DirectoryFilterForm,
  FilterField,
  FilterPanel,
  INLINE_FILTER_ACTION_GRID_CLASS,
} from '@/shell/filter_panel';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
  directoryTableRevalidatingClass,
} from '@/shell/directory_table';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { MLManualLabel } from '@/api/types';
import type { FraudLabelBulkDraftRow } from '@/domains/fraud/use_fraud_labels_page_workspace';
import { displayTimestamp } from '@/lib/display';

export type FraudLabelsProps = {
  items?: MLManualLabel[];
  total: number;
  limit: number;
  offset: number;
  customerId: string;
  draftCustomerId: string;
  draftIpHash: string;
  draftLabel: string;
  draftReason: string;
  fetching: boolean;
  listRevalidating?: boolean;
  saving: boolean;
  error: Error | undefined;
  saveError: Error | undefined;
  saveSuccess: boolean;
  bulkRows: FraudLabelBulkDraftRow[];
  bulkSaving: boolean;
  bulkError: Error | undefined;
  bulkSuccess: boolean;
  bulkUpserted?: number;
  bulkMaxRows: number;
  hasSnapshot: boolean;
  onAddBulkRow: () => void;
  onRemoveBulkRow: (rowId: string) => void;
  onBulkRowChange: (
    rowId: string,
    patch: Partial<Pick<FraudLabelBulkDraftRow, 'ip_hash' | 'label' | 'reason'>>
  ) => void;
  onBulkUpsert: () => void;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftIpHashChange: (value: string) => void;
  onDraftLabelChange: (value: string) => void;
  onDraftReasonChange: (value: string) => void;
  onApplyCustomer: () => void;
  onPageChange: (nextOffset: number) => void;
  onSaveLabel: () => void;
};

export function FraudLabels({
  items,
  total,
  limit,
  offset,
  customerId,
  draftCustomerId,
  draftIpHash,
  draftLabel,
  draftReason,
  fetching,
  listRevalidating = false,
  saving,
  error,
  saveError,
  saveSuccess,
  bulkRows,
  bulkSaving,
  bulkError,
  bulkSuccess,
  bulkUpserted,
  bulkMaxRows,
  hasSnapshot,
  onAddBulkRow,
  onRemoveBulkRow,
  onBulkRowChange,
  onBulkUpsert,
  onDraftCustomerIdChange,
  onDraftIpHashChange,
  onDraftLabelChange,
  onDraftReasonChange,
  onApplyCustomer,
  onPageChange,
  onSaveLabel,
}: FraudLabelsProps) {
  const ipHashValid = /^[0-9a-fA-F]{32}$/.test(draftIpHash.trim());
  const rowCount = (items ?? []).length;
  const showTable = Boolean(customerId) && rowCount > 0;

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load fraud labels" message={error.message} />;
  }

  return (
    <PageLayout
      controlPanel={
        <div className="grid gap-3">
          <Link className="text-sm text-muted-foreground hover:underline" to="/fraud">
            Back to fraud hub
          </Link>
          <FilterPanel>
            <DirectoryFilterForm
              className={INLINE_FILTER_ACTION_GRID_CLASS}
              onSubmit={(event) => {
                event.preventDefault();
                onApplyCustomer();
              }}
            >
              <FilterField htmlFor="labels-customer-id" label="Customer ID">
                <Input
                  id="labels-customer-id"
                  value={draftCustomerId}
                  onChange={(event) => onDraftCustomerIdChange(event.target.value)}
                />
              </FilterField>
              <FilterApplyButton disabled={fetching || !draftCustomerId.trim()}>
                Load
              </FilterApplyButton>
            </DirectoryFilterForm>
          </FilterPanel>
          <FilterPanel>
            <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
              <div className="grid gap-2 md:col-span-2">
                <Label htmlFor="labels-ip-hash">IP hash (32 hex)</Label>
                <Input
                  id="labels-ip-hash"
                  value={draftIpHash}
                  onChange={(event) => onDraftIpHashChange(event.target.value)}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="labels-label">Label (0 or 1)</Label>
                <Select value={draftLabel} onValueChange={onDraftLabelChange}>
                  <SelectTrigger id="labels-label" className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="0">0 (legitimate)</SelectItem>
                    <SelectItem value="1">1 (fraud)</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2 md:col-span-2">
                <Label htmlFor="labels-reason">Reason</Label>
                <Input
                  id="labels-reason"
                  value={draftReason}
                  onChange={(event) => onDraftReasonChange(event.target.value)}
                />
              </div>
              <PrimaryActionButton
                disabled={!customerId || !ipHashValid}
                loading={saving}
                onClick={onSaveLabel}
                type="button"
              >
                Upsert label
              </PrimaryActionButton>
            </DirectoryFilterForm>
          </FilterPanel>
          {saveError ? <ErrorBlock title="Save failed" message={saveError.message} /> : null}
          {saveSuccess ? (
            <p className="text-sm text-muted-foreground">Label saved. List refreshed.</p>
          ) : null}
          <FilterPanel className="w-full gap-2">
            <p className="m-0 text-sm font-medium text-foreground">Bulk upsert</p>
            <p className="m-0 text-sm text-muted-foreground">
              Add up to {bulkMaxRows} rows. Each IP hash must be 32 hex characters.
            </p>
            <DirectoryTable horizontalScroll>
              <TableHeader>
                <TableRow>
                  <DirectoryTableHead>IP hash</DirectoryTableHead>
                  <DirectoryTableHead>Label</DirectoryTableHead>
                  <DirectoryTableHead>Reason</DirectoryTableHead>
                  <DirectoryTableHead className="w-24" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {bulkRows.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell>
                      <Input
                        className="text-xs"
                        value={row.ip_hash}
                        onChange={(event) =>
                          onBulkRowChange(row.id, { ip_hash: event.target.value })
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Select
                        value={row.label}
                        onValueChange={(value) => onBulkRowChange(row.id, { label: value })}
                      >
                        <SelectTrigger className="w-full">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="0">0 (legitimate)</SelectItem>
                          <SelectItem value="1">1 (fraud)</SelectItem>
                        </SelectContent>
                      </Select>
                    </TableCell>
                    <TableCell>
                      <Input
                        value={row.reason}
                        onChange={(event) =>
                          onBulkRowChange(row.id, { reason: event.target.value })
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <SecondaryActionButton onClick={() => onRemoveBulkRow(row.id)} type="button">
                        Remove
                      </SecondaryActionButton>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </DirectoryTable>
            <div className="flex flex-wrap gap-2">
              <SecondaryActionButton
                disabled={bulkRows.length >= bulkMaxRows}
                onClick={onAddBulkRow}
                type="button"
              >
                Add row
              </SecondaryActionButton>
              <PrimaryActionButton
                disabled={!customerId || bulkRows.every((row) => !row.ip_hash.trim())}
                loading={bulkSaving}
                onClick={onBulkUpsert}
                type="button"
              >
                Bulk upsert
              </PrimaryActionButton>
            </div>
            {bulkError ? (
              <ErrorBlock title="Bulk upsert failed" message={bulkError.message} />
            ) : null}
            {bulkSuccess ? (
              <p className="m-0 text-sm text-muted-foreground" role="status">
                Bulk upsert complete{bulkUpserted != null ? ` (${bulkUpserted} rows)` : ''}.
              </p>
            ) : null}
          </FilterPanel>
        </div>
      }
      footer={
        showTable ? (
          <DirectoryPaginationFooter
            canGoNext={offset + rowCount < total}
            canGoPrev={offset > 0}
            disabled={fetching}
            onNext={() => onPageChange(offset + limit)}
            onPrev={() => onPageChange(Math.max(0, offset - limit))}
          />
        ) : undefined
      }
      title="Fraud labels"
    >
      {!customerId ? (
        <EmptyState title="Customer required" description="Load labels for a customer first." />
      ) : rowCount === 0 ? (
        <EmptyState title="No labels" description="No manual ML labels for this customer." />
      ) : (
        <DirectoryTable className={directoryTableRevalidatingClass(listRevalidating)}>
          <TableHeader>
            <TableRow>
              <DirectoryTableHead>IP hash</DirectoryTableHead>
              <DirectoryTableHead>Label</DirectoryTableHead>
              <DirectoryTableHead>Reason</DirectoryTableHead>
              <DirectoryTableHead>Source</DirectoryTableHead>
              <DirectoryTableHead>Created</DirectoryTableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(items ?? []).map((row) => (
              <TableRow key={`${row.ip_hash}-${row.created_at}`}>
                <TableCell className="text-xs">{row.ip_hash ?? ''}</TableCell>
                <TableCell>{row.label ?? ''}</TableCell>
                <TableCell>{row.reason ?? ''}</TableCell>
                <TableCell>{row.source ?? ''}</TableCell>
                <TableCell>{displayTimestamp(row.created_at, row.created_at_display)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </DirectoryTable>
      )}

      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </PageLayout>
  );
}
