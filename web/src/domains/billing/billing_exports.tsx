import {
  DirectoryFilterForm,
  FilterField,
  FilterPanel,
  FILTER_PANEL_SUMMARY_CLASS,
  INLINE_FILTER_ACTION_GRID_TWO_ACTIONS_CLASS,
} from '@/shell/filter_panel';
import { BillingNav, billingPanelError } from '@/domains/billing/billing_nav';
import { PageChrome } from '@/shell/page_chrome';
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
    <PageChrome
      title="Billing ledger exports"
      controlPanel={
        <div className="grid gap-3">
          <BillingNav />
          <FilterPanel>
            <DirectoryFilterForm
              layout="auto-fill"
              onSubmit={(event) => event.preventDefault()}
            >
              <div className="grid gap-2 md:col-span-2">
                <Label htmlFor="export-customer-id">Customer ID</Label>
                <Input
                  id="export-customer-id"
                  value={draftCustomerId}
                  onChange={(event) => onDraftCustomerIdChange(event.target.value)}
                />
              </div>
              <DatetimePicker
                id="export-from"
                label="From"
                value={draftFrom}
                onChange={onDraftFromChange}
              />
              <DatetimePicker id="export-to" label="To" value={draftTo} onChange={onDraftToChange} />
              <div className="grid gap-2">
                <Label htmlFor="export-format">Format</Label>
                <Select
                  value={draftFormat}
                  onValueChange={(value) => onDraftFormatChange(value as 'csv' | 'ndjson')}
                >
                  <SelectTrigger id="export-format" className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="csv">CSV</SelectItem>
                    <SelectItem value="ndjson">NDJSON</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <Button
                disabled={creating || !draftCustomerId.trim()}
                onClick={onCreateJob}
                type="button"
              >
                {creating ? 'Enqueueing...' : 'Start export'}
              </Button>
            </DirectoryFilterForm>
          </FilterPanel>
          <FilterPanel>
            <DirectoryFilterForm
              className={INLINE_FILTER_ACTION_GRID_TWO_ACTIONS_CLASS}
              onSubmit={(event) => event.preventDefault()}
            >
              <FilterField htmlFor="export-job-id" label="Job ID">
                <Input
                  id="export-job-id"
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
                Poll
              </Button>
              <Button disabled={!canDownload} onClick={onDownloadJob} type="button" variant="secondary">
                Download
              </Button>
            </DirectoryFilterForm>
          </FilterPanel>
        </div>
      }
    >
      {job ? (
        <div className={FILTER_PANEL_SUMMARY_CLASS}>
          <p>
            Status: <strong>{job.status ?? ''}</strong>
          </p>
          {job.bytes != null ? <p>Bytes: {job.bytes}</p> : null}
          {job.error ? <p className="text-destructive">{job.error}</p> : null}
        </div>
      ) : null}

      {actionError ? billingPanelError(actionError, 'Export action failed') : null}
      {error ? billingPanelError(error, 'Could not load job') : null}
    </PageChrome>
  );
}
