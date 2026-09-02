import { Link } from 'react-router-dom';

import { PageChrome } from '@/shell/page_chrome';
import { ErrorBlock } from '@/shell/error_block';
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
import type { ReportJobStatus } from '@/api/types';

export type ReportJobsProps = {
  draftCustomerId: string;
  draftReportKey: string;
  draftFrom: string;
  draftTo: string;
  draftFormat: 'csv' | 'json';
  draftJobId: string;
  job: ReportJobStatus | undefined;
  creating: boolean;
  polling: boolean;
  error: Error | undefined;
  actionError: Error | undefined;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftReportKeyChange: (value: string) => void;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
  onDraftFormatChange: (value: 'csv' | 'json') => void;
  onDraftJobIdChange: (value: string) => void;
  onCreateJob: () => void;
  onPollJob: () => void;
  onCancelJob: () => void;
  onDownloadJob: () => void;
};

export function ReportJobs({
  draftCustomerId,
  draftReportKey,
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
  onDraftReportKeyChange,
  onDraftFromChange,
  onDraftToChange,
  onDraftFormatChange,
  onDraftJobIdChange,
  onCreateJob,
  onPollJob,
  onCancelJob,
  onDownloadJob,
}: ReportJobsProps) {
  const jobStatus = job?.status ?? '';
  const canDownload = jobStatus === 'completed' || jobStatus === 'done';
  const canCancel = jobStatus === 'pending' || jobStatus === 'running' || jobStatus === 'queued';

  return (
    <PageChrome title="Report export jobs">
      <Link className="text-sm text-muted-foreground hover:underline" to="/reports">
        Back to catalog
      </Link>

      <div className="ui-filter-panel md:grid-cols-[repeat(auto-fill,minmax(12rem,1fr))]">
        <div className="grid gap-2 md:col-span-2">
          <Label htmlFor="job-customer-id">Customer ID</Label>
          <Input
            id="job-customer-id"
            value={draftCustomerId}
            onChange={(event) => onDraftCustomerIdChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2 md:col-span-2">
          <Label htmlFor="job-report-key">Report key</Label>
          <Input
            id="job-report-key"
            value={draftReportKey}
            onChange={(event) => onDraftReportKeyChange(event.target.value)}
          />
        </div>
        <DatetimePicker
          id="job-from"
          label="From"
          value={draftFrom}
          onChange={onDraftFromChange}
        />
        <DatetimePicker id="job-to" label="To" value={draftTo} onChange={onDraftToChange} />
        <div className="grid gap-2">
          <Label htmlFor="job-format">Format</Label>
          <Select value={draftFormat} onValueChange={(value) => onDraftFormatChange(value as 'csv' | 'json')}>
            <SelectTrigger id="job-format">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="csv">csv</SelectItem>
              <SelectItem value="json">json</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button disabled={creating} onClick={onCreateJob} type="button">
          Enqueue job
        </Button>
      </div>

      <div className="grid max-w-xl grid-cols-[1fr_auto_auto_auto] items-end gap-4">
        <div className="grid gap-2">
          <Label htmlFor="job-id">Job ID</Label>
          <Input
            id="job-id"
            value={draftJobId}
            onChange={(event) => onDraftJobIdChange(event.target.value)}
          />
        </div>
        <Button disabled={polling || !draftJobId.trim()} onClick={onPollJob} type="button" variant="outline">
          Poll
        </Button>
        <Button disabled={!canCancel || !draftJobId.trim()} onClick={onCancelJob} type="button" variant="outline">
          Cancel
        </Button>
        <Button disabled={!canDownload || !draftJobId.trim()} onClick={onDownloadJob} type="button">
          Download
        </Button>
      </div>

      {job ? (
        <div className="ui-filter-panel gap-2 text-sm">
          <div>Status: {job.status ?? 'unknown'}</div>
          {job.report_key ? <div>Report: {job.report_key}</div> : null}
          {job.bytes != null ? <div>Bytes: {job.bytes}</div> : null}
          {job.error ? <div className="text-destructive">Error: {job.error}</div> : null}
        </div>
      ) : null}

      {error ? <ErrorBlock title="Job load failed" message={error.message} /> : null}
      {actionError ? <ErrorBlock title="Action failed" message={actionError.message} /> : null}
    </PageChrome>
  );
}
