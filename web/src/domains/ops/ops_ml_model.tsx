import { PageChrome } from '@/components/system/page_chrome';
import { EmptyState } from '@/components/system/empty_state';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Button } from '@/components/ui/button';
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
import type { MLManualLabel } from '@/api/types';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { OpsNav, opsPanelError } from '@/domains/ops/ops_nav';

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
    return <PageSkeleton />;
  }

  return (
    <PageChrome title="ML model ops">
      <OpsNav />

      <div className="flex flex-wrap gap-2">
        <Button disabled={fetchingStatus} onClick={onLoadStatus} type="button" variant="outline">
          {fetchingStatus ? 'Loading...' : 'Load status'}
        </Button>
        <Button disabled={fetchingEval} onClick={onLoadEval} type="button" variant="outline">
          {fetchingEval ? 'Loading...' : 'Load eval'}
        </Button>
        <Button disabled={fetchingLabels} onClick={onLoadLabels} type="button" variant="outline">
          {fetchingLabels ? 'Loading...' : 'Load labels'}
        </Button>
      </div>

      {statusError && !hasStatusSnapshot ? opsPanelError(statusError, 'Could not load ML status') : null}
      {evalError && !hasEvalSnapshot ? opsPanelError(evalError, 'Could not load ML eval') : null}
      {labelsError && !hasLabelsSnapshot ? opsPanelError(labelsError, 'Could not load ML labels') : null}

      {status ? <JsonDashboardView payload={status} /> : null}
      {evalBlock ? <JsonDashboardView payload={evalBlock} /> : null}

      <section className="ui-filter-panel">
        <h2 className="text-base font-semibold">Add manual label</h2>
        <div className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4">
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
          <div className="grid gap-2 md:col-span-2">
            <Label htmlFor="ml-reason">Reason</Label>
            <Input
              id="ml-reason"
              value={draftReason}
              onChange={(event) => onDraftReasonChange(event.target.value)}
            />
          </div>
          <Button disabled={savingLabel} onClick={onAddLabel} type="button">
            {savingLabel ? 'Saving...' : 'Add label'}
          </Button>
        </div>
        {saveError ? opsPanelError(saveError, 'Could not add ML label') : null}
        {saveSuccess ? (
          <p className="text-sm text-muted-foreground" role="status">
            Label stored.
          </p>
        ) : null}
      </section>

      {labels.length === 0 && hasLabelsSnapshot ? (
        <EmptyState title="No ML labels" description="Fleet manual label list is empty." />
      ) : null}

      {labels.length > 0 ? (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>IP hash</TableHead>
                <TableHead>Label</TableHead>
                <TableHead>Reason</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {labels.map((row, index) => (
                <TableRow key={`${row.ip_hash ?? 'row'}-${index}`}>
                  <TableCell className="font-mono text-xs">{row.ip_hash ?? ''}</TableCell>
                  <TableCell>{row.label ?? ''}</TableCell>
                  <TableCell>{row.reason ?? ''}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : null}
    </PageChrome>
  );
}
