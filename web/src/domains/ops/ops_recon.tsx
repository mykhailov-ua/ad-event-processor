import { Button } from '@/components/ui/button';
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
import { OpsTable } from '@/domains/ops/ops_table';

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
        <label className="admin-label">
          Service
          <input
            className="admin-input"
            id="recon-service"
            value={draftService}
            onChange={(event) => onDraftServiceChange(event.target.value)}
          />
        </label>
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
            <tr>
              <th>ID</th>
              <th>Service</th>
              <th>Status</th>
              <th>Period start</th>
              <th>Period end</th>
              <th>Discrepancies</th>
              <th>Created</th>
            </tr>
          }
        >
          {items.map((row) => (
            <tr key={`${row.service ?? 'svc'}-${row.id ?? row.created_at}`}>
              <td className="admin-table-td--id">{row.id ?? ''}</td>
              <td>{row.service ?? ''}</td>
              <td>{row.status ?? ''}</td>
              <td>{displayTimestamp(row.period_start)}</td>
              <td>{displayTimestamp(row.period_end)}</td>
              <td className="num">{row.discrepancies_found ?? ''}</td>
              <td>{displayTimestamp(row.created_at)}</td>
            </tr>
          ))}
        </OpsTable>
      )}

      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </OpsPageShell>
  );
}
