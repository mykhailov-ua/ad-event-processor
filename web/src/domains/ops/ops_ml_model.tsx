import { useMemo, useState } from 'react';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import type { MLManualLabel, OpsMlModelEvalResponse, OpsMlModelStatusResponse } from '@/api/types';
import { JsonPayloadView } from '@/shell/json_payload_view';
import { EmptyState } from '@/shell/empty_state';
import type { DirectoryOverviewField } from '@/shell/directory_overview_dialog';
import { DirectorySelectOverviewTable } from '@/shell/directory_select_overview_table';
import { TableHost } from '@/shell/ui_bands';
import { adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsActionGroup, OpsPageLoading, OpsPageShell } from '@/domains/ops/ops_page_shell';

export type OpsMlModelProps = {
  status: OpsMlModelStatusResponse | undefined;
  evalBlock: OpsMlModelEvalResponse | undefined;
  labels?: MLManualLabel[];
  draftIpHash: string;
  draftLabel: string;
  draftReason: string;
  fetchingStatus: boolean;
  fetchingEval: boolean;
  fetchingLabels: boolean;
  savingLabel: boolean;
  statusError: Error | undefined;
  evalError: Error | undefined;
  labelsError: Error | undefined;
  saveError: Error | undefined;
  saveSuccess: boolean;
  hasStatusSnapshot: boolean;
  hasEvalSnapshot: boolean;
  hasLabelsSnapshot: boolean;
  onDraftIpHashChange: (value: string) => void;
  onDraftLabelChange: (value: string) => void;
  onDraftReasonChange: (value: string) => void;
  onLoadStatus: () => void;
  onLoadEval: () => void;
  onLoadLabels: () => void;
  onAddLabel: () => void;
};

function mlLabelRowId(row: MLManualLabel, index: number): string | undefined {
  if (row.ip_hash && row.created_at) {
    return `${row.ip_hash}:${row.created_at}`;
  }
  return row.ip_hash ?? `label-${index}`;
}

function mlLabelRowLabel(row: MLManualLabel): string {
  return row.ip_hash ?? 'ML label';
}

function buildMlLabelOverviewFields(row: MLManualLabel): DirectoryOverviewField[] {
  return [
    {
      label: 'IP hash',
      value: (
        <span className={cn(adminTypography.monoData, 'text-muted-foreground')}>
          {row.ip_hash ?? '-'}
        </span>
      ),
    },
    { label: 'Label', value: row.label ?? '-' },
    { label: 'Reason', value: row.reason ?? '-' },
    { label: 'Source', value: row.source ?? '-' },
    {
      label: 'Created',
      value: row.created_at ?? '-',
    },
  ];
}

export function OpsMlModel({
  status,
  evalBlock,
  labels,
  draftIpHash,
  draftLabel,
  draftReason,
  fetchingStatus,
  fetchingEval,
  fetchingLabels,
  savingLabel,
  statusError,
  evalError,
  labelsError,
  saveError,
  saveSuccess,
  hasStatusSnapshot,
  hasEvalSnapshot,
  hasLabelsSnapshot,
  onDraftIpHashChange,
  onDraftLabelChange,
  onDraftReasonChange,
  onLoadStatus,
  onLoadEval,
  onLoadLabels,
  onAddLabel,
}: OpsMlModelProps) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const labelRows = labels ?? [];
  const recordById = useMemo(() => {
    const map = new Map<string, MLManualLabel>();
    labelRows.forEach((row, index) => {
      const id = mlLabelRowId(row, index);
      if (id) {
        map.set(id, row);
      }
    });
    return map;
  }, [labelRows]);
  const rows = useMemo(() => {
    const operateRows: { id: string; label: string }[] = [];
    labelRows.forEach((row, index) => {
      const id = mlLabelRowId(row, index);
      if (!id) {
        return;
      }
      operateRows.push({ id, label: mlLabelRowLabel(row) });
    });
    return operateRows;
  }, [labelRows]);

  if (fetchingStatus && !hasStatusSnapshot && !statusError) {
    return <OpsPageLoading />;
  }

  return (
    <OpsPageShell
      filters={
        <>
          <div className="grid gap-2">
            <Label htmlFor="ml-ip-hash">IP hash</Label>
            <Input
              id="ml-ip-hash"
              value={draftIpHash}
              onChange={(event) => onDraftIpHashChange(event.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="ml-label">Label</Label>
            <Input
              id="ml-label"
              inputMode="numeric"
              value={draftLabel}
              onChange={(event) => onDraftLabelChange(event.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="ml-reason">Reason</Label>
            <Input
              id="ml-reason"
              value={draftReason}
              onChange={(event) => onDraftReasonChange(event.target.value)}
            />
          </div>
        </>
      }
      title="ML model ops"
      actions={
        <>
          <OpsActionGroup label="ML data">
            <Button
              disabled={fetchingStatus}
              loading={fetchingStatus}
              type="button"
              onClick={onLoadStatus}
            >
              Load status
            </Button>
            <Button
              disabled={fetchingEval}
              loading={fetchingEval}
              type="button"
              onClick={onLoadEval}
            >
              Load eval
            </Button>
            <Button
              disabled={fetchingLabels}
              loading={fetchingLabels}
              type="button"
              onClick={onLoadLabels}
            >
              Load labels
            </Button>
          </OpsActionGroup>
          <OpsActionGroup label="Labels">
            <Button disabled={savingLabel} loading={savingLabel} type="button" onClick={onAddLabel}>
              Add label
            </Button>
          </OpsActionGroup>
        </>
      }
    >
      {statusError && !hasStatusSnapshot
        ? opsPanelError(statusError, 'Could not load ML status')
        : null}
      {statusError && hasStatusSnapshot
        ? opsPanelError(statusError, 'ML status refresh failed')
        : null}
      {evalError && !hasEvalSnapshot ? opsPanelError(evalError, 'Could not load ML eval') : null}
      {evalError && hasEvalSnapshot ? opsPanelError(evalError, 'ML eval refresh failed') : null}
      {labelsError && !hasLabelsSnapshot
        ? opsPanelError(labelsError, 'Could not load ML labels')
        : null}
      {labelsError && hasLabelsSnapshot
        ? opsPanelError(labelsError, 'ML labels refresh failed')
        : null}

      {status ? <JsonPayloadView payload={status} /> : null}
      {evalBlock ? <JsonPayloadView payload={evalBlock} /> : null}

      {saveError ? opsPanelError(saveError, 'Could not add ML label') : null}
      {saveSuccess ? (
        <p className="text-muted-foreground" role="status">
          Label stored.
        </p>
      ) : null}

      {labelRows.length === 0 && hasLabelsSnapshot ? (
        <EmptyState description="Fleet manual label list is empty." title="No ML labels" />
      ) : null}

      {labelRows.length > 0 ? (
        <TableHost>
          <DirectorySelectOverviewTable
            buildOverviewFields={buildMlLabelOverviewFields}
            disabled={fetchingLabels || savingLabel}
            nameColumnLabel="ML label"
            overviewTitle={(row) => mlLabelRowLabel(row)}
            recordById={recordById}
            rows={rows}
            selectedId={selectedId}
            onSelectedIdChange={setSelectedId}
          />
        </TableHost>
      ) : null}
    </OpsPageShell>
  );
}
