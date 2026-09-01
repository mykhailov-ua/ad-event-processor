import { Link } from 'react-router-dom';

import { PageChrome } from '@/components/system/page_chrome';
import { ErrorBlock } from '@/components/system/error_block';
import { Button } from '@/components/ui/button';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { BillingExportJob } from '@/api/types';

export type BillingExportsProps = {
  draftCustomerId: string;
  draftFrom: string;
  draftTo: string;
  draftFormat: 'csv' | 'ndjson';
  draftJobId: string;
  job: BillingExportJob | undefined;
  creating: boolean;
  polling: boolean;
  error: Error | undefined;
  actionError: Error | undefined;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
  onDraftFormatChange: (value: 'csv' | 'ndjson') => void;
  onDraftJobIdChange: (value: string) => void;
  onCreateJob: () => void;
  onPollJob: () => void;
  onDownloadJob: () => void;
};

export function BillingExports({
  draftCustomerId,
  draftFrom,
  draftTo,
  draftFormat,
  draftJobId,
  job,
  creating,
  polling,
  error,
  actionError,
  onDraftCustomerIdChange,
  onDraftFromChange,
  onDraftToChange,
  onDraftFormatChange,
  onDraftJobIdChange,
  onCreateJob,
  onPollJob,
  onDownloadJob,
}: BillingExportsProps) {
  const status = (job?.status ?? '').toUpperCase();
  const canDownload = status === 'COMPLETED';

  return (
    <PageChrome title="Billing ledger exports">
      <Link className="text-sm text-muted-foreground hover:underline" to="/billing">
        Back to billing
      </Link>

      <div className="ui-filter-panel md:grid-cols-[repeat(auto-fill,minmax(12rem,1fr))]">
        <div className="grid gap-2 md:col-span-2">
          <Label htmlFor="export-customer-id">Customer ID</Label>
          <Input
            id="export-customer-id"
            value={draftCustomerId}
            onChange={(event) => onDraftCustomerIdChange(event.target.value)}
          />
        </div>
        <DatetimePicker id="export-from" label="From" value={draftFrom} onChange={onDraftFromChange} />
        <DatetimePicker id="export-to" label="To" value={draftTo} onChange={onDraftToChange} />
        <div className="grid gap-2">
          <Label htmlFor="export-format">Format</Label>
          <Select value={draftFormat} onValueChange={(value) => onDraftFormatChange(value as 'csv' | 'ndjson')}>
            <SelectTrigger id="export-format" className="h-9 w-full text-sm">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="csv">CSV</SelectItem>
              <SelectItem value="ndjson">NDJSON</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button disabled={creating || !draftCustomerId.trim()} onClick={onCreateJob} type="button">
          {creating ? 'Enqueueing...' : 'Start export'}
        </Button>
      </div>

      <div className="grid max-w-md grid-cols-[1fr_auto_auto] items-end gap-4">
        <div className="grid gap-2">
          <Label htmlFor="export-job-id">Job ID</Label>
          <Input
            id="export-job-id"
            value={draftJobId}
            onChange={(event) => onDraftJobIdChange(event.target.value)}
          />
        </div>
        <Button disabled={polling || !draftJobId.trim()} onClick={onPollJob} type="button" variant="outline">
          Poll
        </Button>
        <Button disabled={!canDownload} onClick={onDownloadJob} type="button" variant="secondary">
          Download
        </Button>
      </div>

      {job ? (
        <div className="ui-surface p-4 text-sm">
          <p>
            Status: <strong>{job.status ?? ''}</strong>
          </p>
          {job.bytes != null ? <p>Bytes: {job.bytes}</p> : null}
          {job.error ? <p className="text-destructive">{job.error}</p> : null}
        </div>
      ) : null}

      {actionError ? <ErrorBlock title="Export action failed" message={actionError.message} /> : null}
      {error ? <ErrorBlock title="Could not load job" message={error.message} /> : null}
    </PageChrome>
  );
}
