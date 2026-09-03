import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { EmptyState } from '@/shell/empty_state';
import type { MLManualLabel } from '@/api/types';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsActionGroup, OpsPageLoading, OpsPageShell } from '@/domains/ops/ops_page_shell';
import {
  OpsTable,
  OpsTableCell,
  OpsTableHead,
  OpsTableHeaderRow,
  OpsTableRow,
} from '@/domains/ops/ops_table';

export type OpsMlModelProps = {
  status: Record<string, unknown> | undefined;
  evalBlock: Record<string, unknown> | undefined;
  labels: MLManualLabel[];
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
            <Button disabled={fetchingStatus} loading={fetchingStatus} type="button" onClick={onLoadStatus}>
              Load status
            </Button>
            <Button disabled={fetchingEval} loading={fetchingEval} type="button" onClick={onLoadEval}>
              Load eval
            </Button>
            <Button disabled={fetchingLabels} loading={fetchingLabels} type="button" onClick={onLoadLabels}>
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
      {statusError && !hasStatusSnapshot ? opsPanelError(statusError, 'Could not load ML status') : null}
      {evalError && !hasEvalSnapshot ? opsPanelError(evalError, 'Could not load ML eval') : null}
      {labelsError && !hasLabelsSnapshot ? opsPanelError(labelsError, 'Could not load ML labels') : null}

      {status ? <JsonDashboardView payload={status} /> : null}
      {evalBlock ? <JsonDashboardView payload={evalBlock} /> : null}

      {saveError ? opsPanelError(saveError, 'Could not add ML label') : null}
      {saveSuccess ? (
        <p className="text-zinc-500 dark:text-zinc-400" role="status">
          Label stored.
        </p>
      ) : null}

      {labels.length === 0 && hasLabelsSnapshot ? (
        <EmptyState description="Fleet manual label list is empty." title="No ML labels" />
      ) : null}

      {labels.length > 0 ? (
        <OpsTable
          head={
            <OpsTableHeaderRow>
              <OpsTableHead>IP hash</OpsTableHead>
              <OpsTableHead>Label</OpsTableHead>
              <OpsTableHead>Reason</OpsTableHead>
            </OpsTableHeaderRow>
          }
        >
          {labels.map((row, index) => (
            <OpsTableRow key={`${row.ip_hash ?? 'row'}-${index}`}>
              <OpsTableCell className="font-mono text-xs text-zinc-500 dark:text-zinc-400">
                {row.ip_hash ?? ''}
              </OpsTableCell>
              <OpsTableCell>{row.label ?? ''}</OpsTableCell>
              <OpsTableCell>{row.reason ?? ''}</OpsTableCell>
            </OpsTableRow>
          ))}
        </OpsTable>
      ) : null}
    </OpsPageShell>
  );
}
