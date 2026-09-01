import { Link } from 'react-router-dom';

import { FilterApplyButton, PrimaryActionButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { EmptyState } from '@/components/system/empty_state';
import { ErrorBlock } from '@/components/system/error_block';
import { PageSkeleton } from '@/components/system/page_skeleton';
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
import { Textarea } from '@/components/ui/textarea';
import type { MLManualLabel } from '@/api/types';
import { displayTimestamp } from '@/lib/display';

export type FraudLabelsProps = {
  items: MLManualLabel[];
  total: number;
  limit: number;
  offset: number;
  customerId: string;
  draftCustomerId: string;
  draftIpHash: string;
  draftLabel: string;
  draftReason: string;
  fetching: boolean;
  saving: boolean;
  error: Error | undefined;
  saveError: Error | undefined;
  saveSuccess: boolean;
  draftBulkJson: string;
  bulkSaving: boolean;
  bulkError: Error | undefined;
  bulkSuccess: boolean;
  bulkUpserted?: number;
  hasSnapshot: boolean;
  onDraftBulkJsonChange: (value: string) => void;
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
  saving,
  error,
  saveError,
  saveSuccess,
  draftBulkJson,
  bulkSaving,
  bulkError,
  bulkSuccess,
  bulkUpserted,
  hasSnapshot,
  onDraftBulkJsonChange,
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

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load fraud labels" message={error.message} />;
  }

  return (
    <PageChrome title="Fraud labels">
      <Link className="text-sm text-muted-foreground hover:underline" to="/fraud">
        Back to fraud hub
      </Link>

      <form
        className="grid max-w-md grid-cols-[1fr_auto] items-end gap-4"
        onSubmit={(event) => {
          event.preventDefault();
          onApplyCustomer();
        }}
      >
        <div className="grid gap-2">
          <Label htmlFor="labels-customer-id">Customer ID</Label>
          <Input
            id="labels-customer-id"
            className="h-9 text-sm"
            value={draftCustomerId}
            onChange={(event) => onDraftCustomerIdChange(event.target.value)}
          />
        </div>
        <FilterApplyButton disabled={fetching || !draftCustomerId.trim()}>Load</FilterApplyButton>
      </form>

      <div className="ui-filter-panel md:grid-cols-[repeat(auto-fill,minmax(12rem,1fr))]">
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
            <SelectTrigger id="labels-label" className="h-9 w-full text-sm">
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
      </div>

      {saveError ? <ErrorBlock title="Save failed" message={saveError.message} /> : null}
      {saveSuccess ? (
        <p className="text-sm text-muted-foreground">Label saved. List refreshed.</p>
      ) : null}

      <div className="ui-filter-panel">
        <p className="text-sm font-medium">Bulk upsert</p>
        <p className="text-sm text-muted-foreground">
          Paste JSON: {'{ "rows": [ { "ip_hash": "...", "label": 1, "reason": "..." } ] }'}
        </p>
        <Textarea
          id="labels-bulk-json"
          rows={6}
          value={draftBulkJson}
          onChange={(event) => onDraftBulkJsonChange(event.target.value)}
        />
        <PrimaryActionButton
          disabled={!customerId || !draftBulkJson.trim()}
          loading={bulkSaving}
          onClick={onBulkUpsert}
          type="button"
        >
          Bulk upsert
        </PrimaryActionButton>
        {bulkError ? <ErrorBlock title="Bulk upsert failed" message={bulkError.message} /> : null}
        {bulkSuccess ? (
          <p className="text-sm text-muted-foreground" role="status">
            Bulk upsert complete{bulkUpserted != null ? ` (${bulkUpserted} rows)` : ''}.
          </p>
        ) : null}
      </div>

      {!customerId ? (
        <EmptyState title="Customer required" description="Load labels for a customer first." />
      ) : items.length === 0 ? (
        <EmptyState title="No labels" description="No manual ML labels for this customer." />
      ) : (
        <>
          <form
            className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4"
            onSubmit={(event) => event.preventDefault()}
          >
            <PaginationPrevNext
              canGoPrev={offset > 0}
              canGoNext={offset + items.length < total}
              disabled={fetching}
              onPrev={() => onPageChange(Math.max(0, offset - limit))}
              onNext={() => onPageChange(offset + limit)}
            />
          </form>
          <div className="ui-table-frame">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>IP hash</TableHead>
                  <TableHead>Label</TableHead>
                  <TableHead>Reason</TableHead>
                  <TableHead>Source</TableHead>
                  <TableHead>Created</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((row) => (
                  <TableRow key={`${row.ip_hash}-${row.created_at}`}>
                    <TableCell className="font-mono text-xs">{row.ip_hash ?? ''}</TableCell>
                    <TableCell className="tabular-nums">{row.label ?? ''}</TableCell>
                    <TableCell>{row.reason ?? ''}</TableCell>
                    <TableCell>{row.source ?? ''}</TableCell>
                    <TableCell>{displayTimestamp(row.created_at, row.created_at_display)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </>
      )}

      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </PageChrome>
  );
}
