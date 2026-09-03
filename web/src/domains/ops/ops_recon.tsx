import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { EmptyState } from '@/shell/empty_state';
import type { ReconRun } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsListFooter } from '@/domains/ops/ops_list_footer';
import {
  OpsActionGroup,
  OpsPageBlockingError,
  OpsPageLoading,
  OpsPageShell,
} from '@/domains/ops/ops_page_shell';
import {
  OpsTable,
  OpsTableCell,
  OpsTableHead,
  OpsTableHeaderRow,
  OpsTableRow,
} from '@/domains/ops/ops_table';

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
    return <OpsPageLoading />;
  }

  if (error && !hasSnapshot) {
    return (
      <OpsPageBlockingError
        error={error}
        pageTitle="Reconciliation runs"
        title="Could not load recon runs"
      />
    );
  }

  const canGoPrev = offset > 0;
  const canGoNext = items.length >= limit;

  return (
    <OpsPageShell
      filters={
        <div className="grid gap-2">
          <Label htmlFor="recon-service">Service</Label>
          <Input
            id="recon-service"
            value={draftService}
            onChange={(event) => onDraftServiceChange(event.target.value)}
          />
        </div>
      }
      footer={
        <OpsListFooter
          canGoNext={canGoNext}
          canGoPrev={canGoPrev}
          disabled={fetching}
          summary={`Offset ${offset}`}
          onNext={() => onPageChange(offset + limit)}
          onPrev={() => onPageChange(Math.max(0, offset - limit))}
        />
      }
      title="Reconciliation runs"
      actions={
        <OpsActionGroup label="Filters">
          <Button disabled={fetching} loading={fetching} type="button" onClick={onApplyFilters}>
            Apply
          </Button>
        </OpsActionGroup>
      }
    >
      {items.length === 0 ? (
        <EmptyState description="No reconciliation runs match filters." title="No recon runs" />
      ) : (
        <OpsTable
          head={
            <OpsTableHeaderRow>
              <OpsTableHead>ID</OpsTableHead>
              <OpsTableHead>Service</OpsTableHead>
              <OpsTableHead>Status</OpsTableHead>
              <OpsTableHead>Period start</OpsTableHead>
              <OpsTableHead>Period end</OpsTableHead>
              <OpsTableHead>Discrepancies</OpsTableHead>
              <OpsTableHead>Created</OpsTableHead>
            </OpsTableHeaderRow>
          }
        >
          {items.map((row) => (
            <OpsTableRow key={`${row.service ?? 'svc'}-${row.id ?? row.created_at}`}>
              <OpsTableCell className="font-mono text-xs text-zinc-500 dark:text-zinc-400">
                {row.id ?? ''}
              </OpsTableCell>
              <OpsTableCell>{row.service ?? ''}</OpsTableCell>
              <OpsTableCell>{row.status ?? ''}</OpsTableCell>
              <OpsTableCell>{displayTimestamp(row.period_start)}</OpsTableCell>
              <OpsTableCell>{displayTimestamp(row.period_end)}</OpsTableCell>
              <OpsTableCell numeric>{row.discrepancies_found ?? ''}</OpsTableCell>
              <OpsTableCell>{displayTimestamp(row.created_at)}</OpsTableCell>
            </OpsTableRow>
          ))}
        </OpsTable>
      )}

      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </OpsPageShell>
  );
}
