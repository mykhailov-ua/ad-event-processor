import { FilterApplyButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { EmptyState } from '@/components/system/empty_state';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { PaginationPrevNext } from '@/components/system/pagination_prev_next';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { ReconRun } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { OpsNav, opsPanelError } from '@/domains/ops/ops_nav';

export type OpsReconProps = {
  items: ReconRun[];
  draftService: string;
  limit: number;
  offset: number;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  onDraftServiceChange: (value: string) => void;
  onApplyFilters: () => void;
  onPageChange: (nextOffset: number) => void;
};

export function OpsRecon({
  items,
  draftService,
  limit,
  offset,
  fetching,
  error,
  hasSnapshot,
  onDraftServiceChange,
  onApplyFilters,
  onPageChange,
}: OpsReconProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Reconciliation runs">
        <OpsNav />
        {opsPanelError(error, 'Could not load recon runs')}
      </PageChrome>
    );
  }

  const canGoPrev = offset > 0;
  const canGoNext = items.length >= limit;

  return (
    <PageChrome title="Reconciliation runs">
      <OpsNav />

      <form
        className="grid max-w-xl grid-cols-[1fr_auto_auto] items-end gap-4"
        onSubmit={(event) => {
          event.preventDefault();
          onApplyFilters();
        }}
      >
        <div className="grid gap-2">
          <Label htmlFor="recon-service">Service</Label>
          <Input
            id="recon-service"
            value={draftService}
            onChange={(event) => onDraftServiceChange(event.target.value)}
          />
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
        <EmptyState title="No recon runs" description="No reconciliation runs match filters." />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>Service</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Period start</TableHead>
                <TableHead>Period end</TableHead>
                <TableHead>Discrepancies</TableHead>
                <TableHead>Created</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((row) => (
                <TableRow key={`${row.service ?? 'svc'}-${row.id ?? row.created_at}`}>
                  <TableCell>{row.id ?? ''}</TableCell>
                  <TableCell>{row.service ?? ''}</TableCell>
                  <TableCell>{row.status ?? ''}</TableCell>
                  <TableCell>{displayTimestamp(row.period_start)}</TableCell>
                  <TableCell>{displayTimestamp(row.period_end)}</TableCell>
                  <TableCell>{row.discrepancies_found ?? ''}</TableCell>
                  <TableCell>{displayTimestamp(row.created_at)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
