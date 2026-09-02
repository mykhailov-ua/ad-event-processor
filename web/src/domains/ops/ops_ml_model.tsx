import { Button } from '@/components/ui/button';
import { EmptyState } from '@/shell/empty_state';
import type { MLManualLabel } from '@/api/types';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsActionGroup, OpsPageLoading, OpsPageShell } from '@/domains/ops/ops_page_shell';
import { OpsTable } from '@/domains/ops/ops_table';

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
          <label className="admin-label">
            IP hash
            <input
              className="admin-input"
              id="ml-ip-hash"
              value={draftIpHash}
              onChange={(event) => onDraftIpHashChange(event.target.value)}
            />
          </label>
          <label className="admin-label">
            Label
            <input
              className="admin-input"
              id="ml-label"
              inputMode="numeric"
              value={draftLabel}
              onChange={(event) => onDraftLabelChange(event.target.value)}
            />
          </label>
          <label className="admin-label">
            Reason
            <input
              className="admin-input"
              id="ml-reason"
              value={draftReason}
              onChange={(event) => onDraftReasonChange(event.target.value)}
            />
          </label>
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
        <p className="admin-muted" role="status">
          Label stored.
        </p>
      ) : null}

      {labels.length === 0 && hasLabelsSnapshot ? (
        <EmptyState description="Fleet manual label list is empty." title="No ML labels" />
      ) : null}

      {labels.length > 0 ? (
        <OpsTable
          head={
            <tr>
              <th>IP hash</th>
              <th>Label</th>
              <th>Reason</th>
            </tr>
          }
        >
          {labels.map((row, index) => (
            <tr key={`${row.ip_hash ?? 'row'}-${index}`}>
              <td className="admin-table-td--id">{row.ip_hash ?? ''}</td>
              <td>{row.label ?? ''}</td>
              <td>{row.reason ?? ''}</td>
            </tr>
          ))}
        </OpsTable>
      ) : null}
    </OpsPageShell>
  );
}
