import { Link } from 'react-router-dom';

import {
  DirectoryFilterForm,
  FilterField,
  FilterPanel,
  FILTER_PANEL_SUMMARY_CLASS,
  INLINE_FILTER_ACTION_GRID_THREE_ACTIONS_CLASS,
} from '@/shell/filter_panel';
import { PageLayout } from '@/shell/page_layout';
import { ErrorBlock } from '@/shell/error_block';
import { DirectoryStack, MetaLinksBand } from '@/shell/ui_bands';
import { Button } from '@/components/ui/button';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { Input } from '@/components/ui/input';
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
  reportKeyOptions: string[];
  creating: boolean;
  polling: boolean;
  downloading: boolean;
  cancelling: boolean;
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
  reportKeyOptions,
  creating,
  polling,
  downloading,
  cancelling,
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
    <PageLayout
      controlPanel={
        <DirectoryStack>
          <MetaLinksBand>
            <Link to="/reports">Back to catalog</Link>
          </MetaLinksBand>
          <FilterPanel>
            <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
              <FilterField htmlFor="job-customer-id" label="Customer ID" wide>
                <Input
                  id="job-customer-id"
                  value={draftCustomerId}
                  onChange={(event) => onDraftCustomerIdChange(event.target.value)}
                />
              </FilterField>
              <FilterField htmlFor="job-report-key" label="Report key" wide>
                <Select value={draftReportKey} onValueChange={onDraftReportKeyChange}>
                  <SelectTrigger id="job-report-key">
                    <SelectValue placeholder="Select report key" />
                  </SelectTrigger>
                  <SelectContent>
                    {reportKeyOptions.map((key) => (
                      <SelectItem key={key} value={key}>
                        {key}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FilterField>
              <DatetimePicker
                id="job-from"
                label="From"
                value={draftFrom}
                onChange={onDraftFromChange}
              />
              <DatetimePicker id="job-to" label="To" value={draftTo} onChange={onDraftToChange} />
              <FilterField htmlFor="job-format" label="Format">
                <Select
                  value={draftFormat}
                  onValueChange={(value) => onDraftFormatChange(value as 'csv' | 'json')}
                >
                  <SelectTrigger id="job-format">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="csv">csv</SelectItem>
                    <SelectItem value="json">json</SelectItem>
                  </SelectContent>
                </Select>
              </FilterField>
              <Button disabled={creating} onClick={onCreateJob} type="button">
                {creating ? 'Enqueueing...' : 'Enqueue job'}
              </Button>
            </DirectoryFilterForm>
          </FilterPanel>
          <FilterPanel>
            <DirectoryFilterForm
              className={INLINE_FILTER_ACTION_GRID_THREE_ACTIONS_CLASS}
              onSubmit={(event) => event.preventDefault()}
            >
              <FilterField htmlFor="job-id" label="Job ID">
                <Input
                  id="job-id"
                  value={draftJobId}
                  onChange={(event) => onDraftJobIdChange(event.target.value)}
                />
              </FilterField>
              <Button
                disabled={polling || !draftJobId.trim()}
                onClick={onPollJob}
                type="button"
                variant="outline"
              >
                {polling ? 'Polling...' : 'Poll'}
              </Button>
              <Button
                disabled={cancelling || !canCancel || !draftJobId.trim()}
                onClick={onCancelJob}
                type="button"
                variant="outline"
              >
                {cancelling ? 'Cancelling...' : 'Cancel'}
              </Button>
              <Button
                disabled={downloading || !canDownload || !draftJobId.trim()}
                onClick={onDownloadJob}
                type="button"
              >
                {downloading ? 'Downloading...' : 'Download'}
              </Button>
            </DirectoryFilterForm>
          </FilterPanel>
        </DirectoryStack>
      }
      title="Report export jobs"
    >
      {job ? (
        <div className={FILTER_PANEL_SUMMARY_CLASS}>
          <p className="m-0">Status: {job.status ?? 'unknown'}</p>
          {job.report_key ? <p className="m-0">Report: {job.report_key}</p> : null}
          {job.bytes != null ? <p className="m-0">Bytes: {job.bytes}</p> : null}
          {job.error ? <p className="m-0 text-destructive">Error: {job.error}</p> : null}
        </div>
      ) : null}

      {error ? <ErrorBlock title="Job load failed" message={error.message} /> : null}
      {actionError ? <ErrorBlock title="Action failed" message={actionError.message} /> : null}
    </PageLayout>
  );
}
