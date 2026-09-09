import { useMemo, useState } from 'react';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import type { ReconRun } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';
import { EmptyState } from '@/shell/empty_state';
import type { DirectoryOverviewField } from '@/shell/directory_overview_dialog';
import {
  DirectorySelectOverviewTable,
  directoryOperateRows,
  directoryRecordMap,
} from '@/shell/directory_select_overview_table';
import { TableHost } from '@/shell/ui_bands';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsListFooter } from '@/domains/ops/ops_list_footer';
import {
  OpsActionGroup,
  OpsPageBlockingError,
  OpsPageLoading,
  OpsPageShell,
} from '@/domains/ops/ops_page_shell';

export type OpsReconProps = {
  items?: ReconRun[];
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

function reconRowId(row: ReconRun): string | undefined {
  if (row.id != null) {
    return String(row.id);
  }
  if (row.service && row.created_at) {
    return `${row.service}-${row.created_at}`;
  }
  return row.service ?? undefined;
}

function reconRowLabel(row: ReconRun): string {
  if (row.service && row.id != null) {
    return `${row.service} / ${row.id}`;
  }
  return row.id != null ? String(row.id) : (row.service ?? 'Recon run');
}

function buildReconOverviewFields(row: ReconRun): DirectoryOverviewField[] {
  return [
    {
      label: 'ID',
      value: (
        <span className={cn(adminTypography.monoData, 'text-muted-foreground')}>
          {row.id ?? '-'}
        </span>
      ),
    },
    { label: 'Service', value: row.service ?? '-' },
    { label: 'Status', value: row.status ?? '-' },
    {
      label: 'Period start',
      value: displayTimestamp(row.period_start) || '-',
    },
    {
      label: 'Period end',
      value: displayTimestamp(row.period_end) || '-',
    },
    { label: 'Discrepancies', value: row.discrepancies_found ?? '-' },
    {
      label: 'Created',
      value: displayTimestamp(row.created_at) || '-',
    },
  ];
}

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
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const list = items ?? [];
  const recordById = useMemo(() => directoryRecordMap(list, reconRowId), [list]);
  const rows = useMemo(() => directoryOperateRows(list, reconRowId, reconRowLabel), [list]);

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
  const canGoNext = list.length >= limit;

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
      {list.length === 0 ? (
        <EmptyState description="No reconciliation runs match filters." title="No recon runs" />
      ) : (
        <TableHost>
          <DirectorySelectOverviewTable
            buildOverviewFields={buildReconOverviewFields}
            disabled={fetching}
            nameColumnLabel="Recon run"
            overviewTitle={(row) => reconRowLabel(row)}
            recordById={recordById}
            rows={rows}
            selectedId={selectedId}
            onSelectedIdChange={setSelectedId}
          />
        </TableHost>
      )}

      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </OpsPageShell>
  );
}
