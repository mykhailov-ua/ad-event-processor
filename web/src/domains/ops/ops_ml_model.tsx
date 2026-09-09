import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { EmptyState } from '@/shell/empty_state';
import type { MLManualLabel, OpsMlModelEvalResponse, OpsMlModelStatusResponse } from '@/api/types';
import { JsonPayloadView } from '@/shell/json_payload_view';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsActionGroup, OpsPageWithLoad } from '@/domains/ops/ops_page_shell';
import {
  OpsTable,
  OpsTableCell,
  OpsTableHead,
  OpsTableHeaderRow,
  OpsTableRow,
} from '@/domains/ops/ops_table';

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
  const labelRows = labels ?? [];

  return (
    <OpsPageWithLoad
      blockingErrorTitle="Could not load ML status"
      fetchState={{
        fetching: fetchingStatus,
        error: statusError,
        hasSnapshot: hasStatusSnapshot,
      }}
      refreshErrorTitle="ML status refresh failed"
      title="ML model ops"
      alerts={
        <>
          {evalError && !hasEvalSnapshot
            ? opsPanelError(evalError, 'Could not load ML eval')
            : null}
          {labelsError && !hasLabelsSnapshot
            ? opsPanelError(labelsError, 'Could not load ML labels')
            : null}
          {saveError ? opsPanelError(saveError, 'Could not add ML label') : null}
        </>
      }
      filters={
        <>
          <div >
            <Label htmlFor="ml-ip-hash">IP hash</Label>
            <Input
              id="ml-ip-hash"
              value={draftIpHash}
              onChange={(event) => onDraftIpHashChange(event.target.value)}
            />
          </div>
          <div >
            <Label htmlFor="ml-label">Label</Label>
            <Input
              id="ml-label"
              inputMode="numeric"
              value={draftLabel}
              onChange={(event) => onDraftLabelChange(event.target.value)}
            />
          </div>
          <div >
            <Label htmlFor="ml-reason">Reason</Label>
            <Input
              id="ml-reason"
              value={draftReason}
              onChange={(event) => onDraftReasonChange(event.target.value)}
            />
          </div>
        </>
      }
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
      {status ? <JsonPayloadView payload={status} /> : null}
      {evalBlock ? <JsonPayloadView payload={evalBlock} /> : null}

      {saveSuccess ? (
        <p  role="status">
          Label stored.
        </p>
      ) : null}

      {labelRows.length === 0 && hasLabelsSnapshot ? (
        <EmptyState description="Fleet manual label list is empty." title="No ML labels" />
      ) : null}

      {labelRows.length > 0 ? (
        <OpsTable
          horizontalScroll
          head={
            <OpsTableHeaderRow>
              <OpsTableHead>IP hash</OpsTableHead>
              <OpsTableHead>Label</OpsTableHead>
              <OpsTableHead>Reason</OpsTableHead>
            </OpsTableHeaderRow>
          }
        >
          {labelRows.map((row, index) => (
            <OpsTableRow
              key={
                row.ip_hash && row.created_at
                  ? `${row.ip_hash}:${row.created_at}`
                  : (row.ip_hash ?? `row-${index}`)
              }
            >
              <OpsTableCell >
                {row.ip_hash ?? ''}
              </OpsTableCell>
              <OpsTableCell>{row.label ?? ''}</OpsTableCell>
              <OpsTableCell>{row.reason ?? ''}</OpsTableCell>
            </OpsTableRow>
          ))}
        </OpsTable>
      ) : null}
    </OpsPageWithLoad>
  );
}
