import { SecondaryActionButton } from '@/components/system/action_buttons';
import { DirectoryListMeta } from '@/components/system/directory_list_meta';
import {
  DirectoryFilterForm,
  FilterPanel,
} from '@/components/system/filter_panel';
import { PageChrome } from '@/components/system/page_chrome';
import { EmptyState } from '@/components/system/empty_state';
import { ErrorBlock } from '@/components/system/error_block';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { PaginationPrevNext } from '@/components/system/pagination_prev_next';
import { Badge } from '@/components/ui/badge';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { AuditLog } from '@/api/types';
import { displayTimestamp } from '@/lib/display';

export type AuditDirectoryProps = {
  items: AuditLog[];
  total: number;
  limit: number;
  offset: number;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  draftRedactPii: boolean;
  exporting: boolean;
  exportError: Error | undefined;
  exportTruncated: boolean;
  exportNextCursor?: string;
  onDraftRedactPiiChange: (value: boolean) => void;
  onExportCsv: () => void;
  onPageChange: (nextOffset: number) => void;
};

export function AuditDirectory({
  items,
  total,
  limit,
  offset,
  fetching,
  error,
  hasSnapshot,
  draftRedactPii,
  exporting,
  exportError,
  exportTruncated,
  exportNextCursor,
  onDraftRedactPiiChange,
  onExportCsv,
  onPageChange,
}: AuditDirectoryProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load audit log" message={error.message} />;
  }

  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;

  return (
    <PageChrome title="Audit">
      <FilterPanel>
        <DirectoryFilterForm
          onSubmit={(event) => event.preventDefault()}
        >
        <div className="flex items-center gap-2">
          <Checkbox
            checked={draftRedactPii}
            id="audit-redact-pii"
            onCheckedChange={(checked) => onDraftRedactPiiChange(checked === true)}
          />
          <Label htmlFor="audit-redact-pii">Redact PII in export</Label>
        </div>
        <SecondaryActionButton
          disabled={exporting}
          loading={exporting}
          onClick={onExportCsv}
          type="button"
        >
          Export CSV
        </SecondaryActionButton>
        <PaginationPrevNext
          canGoPrev={canGoPrev}
          canGoNext={canGoNext}
          disabled={fetching}
          onPrev={() => onPageChange(Math.max(0, offset - limit))}
          onNext={() => onPageChange(offset + limit)}
        />
        </DirectoryFilterForm>
        <DirectoryListMeta>
          {total > 0
            ? `Showing ${offset + 1}-${Math.min(offset + items.length, total)} of ${total}`
            : 'No audit entries'}
        </DirectoryListMeta>
      </FilterPanel>

      {exportError ? <ErrorBlock title="Export failed" message={exportError.message} /> : null}
      {exportTruncated ? (
        <p className="text-sm text-muted-foreground" role="status">
          Export truncated.{exportNextCursor ? ` Next cursor: ${exportNextCursor}` : ''}
        </p>
      ) : null}

      {items.length === 0 ? (
        <EmptyState
          title="No audit entries"
          description="Admin actions will appear here when recorded."
        />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Time</TableHead>
                <TableHead>Admin</TableHead>
                <TableHead>Action</TableHead>
                <TableHead>Target</TableHead>
                <TableHead>Target ID</TableHead>
                <TableHead>Masked</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((row) => (
                <TableRow key={row.id ?? `${row.created_at}-${row.action}`}>
                  <TableCell>{displayTimestamp(row.created_at, row.created_at_display)}</TableCell>
                  <TableCell>{row.admin_id ?? ''}</TableCell>
                  <TableCell>{row.action ?? ''}</TableCell>
                  <TableCell>{row.target_type ?? ''}</TableCell>
                  <TableCell className="font-mono text-xs">{row.target_id ?? ''}</TableCell>
                  <TableCell>
                    {row.is_masked ? <Badge variant="secondary">masked</Badge> : ''}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {error && hasSnapshot && (
        <ErrorBlock title="Refresh failed" message={error.message} />
      )}
    </PageChrome>
  );
}
