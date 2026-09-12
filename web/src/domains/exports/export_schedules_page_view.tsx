import { Link } from 'react-router-dom';

import type { ReportSchedule } from '@/api/types';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { ExportHubNotifyFields } from '@/domains/exports/export_hub_notify_fields';
import { ExportsNav } from '@/domains/exports/exports_nav';
import type { ExportSchedulesDraft } from '@/domains/exports/use_export_schedules_page_workspace';
import { DashboardPanelSection } from '@/domains/dashboards/dashboard_panel_section';
import { adminTypography } from '@/lib/admin_kit';
import { adminSpacing, opsControlPanelClass } from '@/lib/admin_spacing';
import { CustomerScopeGate } from '@/shell/customer_scope_gate';
import { DirectoryPageShell } from '@/shell/directory_page_shell';
import { PageChrome } from '@/shell/page_chrome';
import { PageSectionStack } from '@/shell/page_layout';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { FilterField, FilterPanel } from '@/shell/filter_panel';
import {
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import type { AdminValidationError } from '@/lib/admin_validation_error';

export type ExportSchedulesPageViewProps = {
  customerId: string;
  onCustomerIdChange: (value: string) => void;
  draft: ExportSchedulesDraft;
  onDraftChange: (patch: Partial<ExportSchedulesDraft>) => void;
  schedules: ReportSchedule[];
  schedulesHasSnapshot: boolean;
  schedulesError?: Error;
  schedulesFetching: boolean;
  canManage: boolean;
  saving: boolean;
  runningScheduleId?: string;
  formValidationError?: AdminValidationError;
  onSaveSchedule: () => void;
  onResetDraft: () => void;
  onSelectSchedule: (schedule: ReportSchedule) => void;
  onDeleteSchedule: (schedule: ReportSchedule) => void;
  onToggleSchedule: (schedule: ReportSchedule) => void;
  onRunScheduleNow: (schedule: ReportSchedule) => void;
  lastJobHref: (schedule: ReportSchedule) => string | undefined;
};

export function ExportSchedulesPageView({
  customerId,
  onCustomerIdChange,
  draft,
  onDraftChange,
  schedules,
  schedulesHasSnapshot,
  schedulesError,
  schedulesFetching,
  canManage,
  saving,
  runningScheduleId,
  formValidationError,
  onSaveSchedule,
  onResetDraft,
  onSelectSchedule,
  onDeleteSchedule,
  onToggleSchedule,
  onRunScheduleNow,
  lastJobHref,
}: ExportSchedulesPageViewProps) {
  return (
    <PageChrome
      description="Recurring report export jobs for a customer."
      title="Export schedules"
      controlPanel={
        <div className={opsControlPanelClass}>
          <ExportsNav />
          <FilterPanel aria-label="Schedule form">
            {formValidationError ? (
              <ErrorBlock error={formValidationError} title="Check schedule fields" />
            ) : null}
            <FilterField htmlFor="export-schedules-customer-id" label="Customer ID">
              <Input
                id="export-schedules-customer-id"
                value={customerId}
                onChange={(event) => onCustomerIdChange(event.target.value)}
              />
            </FilterField>
            <FilterField htmlFor="export-schedules-report-key" label="Report key">
              <Input
                id="export-schedules-report-key"
                disabled={!canManage || saving}
                value={draft.reportKey}
                onChange={(event) => onDraftChange({ reportKey: event.target.value })}
              />
            </FilterField>
            <FilterField htmlFor="export-schedules-cron" label="Cron">
              <Input
                id="export-schedules-cron"
                disabled={!canManage || saving}
                value={draft.cronExpr}
                onChange={(event) => onDraftChange({ cronExpr: event.target.value })}
              />
            </FilterField>
            <FilterField htmlFor="export-schedules-format" label="Format">
              <Select
                disabled={!canManage || saving}
                value={draft.format}
                onValueChange={(value) =>
                  onDraftChange({ format: value as ExportSchedulesDraft['format'] })
                }
              >
                <SelectTrigger id="export-schedules-format">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="csv">CSV</SelectItem>
                  <SelectItem value="xlsx">XLSX</SelectItem>
                  <SelectItem value="json">JSON</SelectItem>
                </SelectContent>
              </Select>
            </FilterField>
            <FilterField htmlFor="export-schedules-destination" label="Destination">
              <Select
                disabled={!canManage || saving}
                value={draft.destination}
                onValueChange={(value) =>
                  onDraftChange({ destination: value as ExportSchedulesDraft['destination'] })
                }
              >
                <SelectTrigger id="export-schedules-destination">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="download">Download</SelectItem>
                  <SelectItem value="google_sheet">Google Sheet</SelectItem>
                </SelectContent>
              </Select>
            </FilterField>
            <FilterField htmlFor="export-schedules-owner-user-id" label="Owner user ID">
              <Input
                id="export-schedules-owner-user-id"
                disabled={!canManage || saving}
                placeholder={draft.destination === 'google_sheet' ? 'Defaults to signed-in user' : undefined}
                value={draft.ownerUserId}
                onChange={(event) => onDraftChange({ ownerUserId: event.target.value })}
              />
            </FilterField>
            {draft.destination === 'google_sheet' ? (
              <>
                <FilterField htmlFor="export-schedules-spreadsheet-id" label="Spreadsheet ID (append)">
                  <Input
                    id="export-schedules-spreadsheet-id"
                    disabled={!canManage || saving}
                    placeholder="Optional for append mode"
                    value={draft.spreadsheetId}
                    onChange={(event) => onDraftChange({ spreadsheetId: event.target.value })}
                  />
                </FilterField>
                <FilterField htmlFor="export-schedules-sheet-title" label="Sheet title">
                  <Input
                    id="export-schedules-sheet-title"
                    disabled={!canManage || saving}
                    value={draft.sheetTitle}
                    onChange={(event) => onDraftChange({ sheetTitle: event.target.value })}
                  />
                </FilterField>
              </>
            ) : null}
            <FilterField htmlFor="export-schedules-lookback" label="Lookback days">
              <Input
                id="export-schedules-lookback"
                disabled={!canManage || saving}
                inputMode="numeric"
                value={draft.fromOffsetDays}
                onChange={(event) => onDraftChange({ fromOffsetDays: event.target.value })}
              />
            </FilterField>
            <ExportHubNotifyFields
              channel={draft.notifyChannel}
              disabled={!canManage || saving}
              email={draft.notifyEmail}
              webhookUrl={draft.notifyWebhookUrl}
              onChannelChange={(value) => onDraftChange({ notifyChannel: value })}
              onEmailChange={(value) => onDraftChange({ notifyEmail: value })}
              onWebhookUrlChange={(value) => onDraftChange({ notifyWebhookUrl: value })}
            />
            <div aria-label="Schedule form actions" className={adminSpacing.flex.buttonGroup}>
              <Button disabled={!canManage || saving} type="button" onClick={() => void onSaveSchedule()}>
                {saving ? 'Saving...' : 'Save schedule'}
              </Button>
              <Button disabled={!canManage || saving} type="button" variant="outline" onClick={onResetDraft}>
                New schedule
              </Button>
            </div>
          </FilterPanel>
        </div>
      }
    >
      <PageSectionStack>
        {!canManage ? (
          <p className={adminTypography.bodyMuted}>
            Read-only: create, update, run, and delete require exports:run.
          </p>
        ) : null}

        <CustomerScopeGate customerId={customerId} testId="export-schedules-scope">
          <DirectoryPageShell
            blockingErrorTitle="Schedules load failed"
            fetchState={{
              fetching: schedulesFetching,
              error: schedulesError,
              hasSnapshot: schedulesHasSnapshot,
            }}
            refreshErrorTitle="Schedules refresh failed"
            title=""
          >
          {schedulesHasSnapshot && !schedulesFetching && schedules.length === 0 ? (
            <EmptyState
              description="No recurring export schedules exist for this customer yet."
              title="No schedules"
            />
          ) : null}
          {schedules.length > 0 ? (
            <DashboardPanelSection tableAriaLabel="Report schedules" title="Schedules">
            <TableHeader>
              <TableRow>
                <DirectoryTableHead>Report</DirectoryTableHead>
                <DirectoryTableHead>Cron</DirectoryTableHead>
                <DirectoryTableHead>Destination</DirectoryTableHead>
                <DirectoryTableHead>Enabled</DirectoryTableHead>
                <DirectoryTableHead>Next run</DirectoryTableHead>
                <DirectoryTableHead>Last status</DirectoryTableHead>
                <DirectoryTableHead>Actions</DirectoryTableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {schedules.map((schedule) => {
                const jobHref = lastJobHref(schedule);
                return (
                  <TableRow key={schedule.id}>
                    <TableCell>{schedule.report_key ?? ''}</TableCell>
                    <TableCell>{schedule.cron_expr ?? ''}</TableCell>
                    <TableCell>{schedule.destination ?? 'download'}</TableCell>
                    <TableCell>{schedule.enabled ? 'yes' : 'no'}</TableCell>
                    <TableCell>
                      {schedule.next_run_at
                        ? new Date(schedule.next_run_at).toLocaleString()
                        : ''}
                    </TableCell>
                    <TableCell>
                      {schedule.last_run_status
                        ? `${schedule.last_run_status}${
                            schedule.last_run_error_public
                              ? `: ${schedule.last_run_error_public}`
                              : ''
                          }`
                        : ''}
                    </TableCell>
                    <TableCell>
                      <div
                        aria-label={`Actions for ${schedule.id}`}
                        className={adminSpacing.flex.buttonGroup}
                      >
                        <Button type="button" variant="outline" onClick={() => onSelectSchedule(schedule)}>
                          Edit
                        </Button>
                        {canManage ? (
                          <Button
                            disabled={runningScheduleId === schedule.id}
                            type="button"
                            variant="outline"
                            onClick={() => void onRunScheduleNow(schedule)}
                          >
                            {runningScheduleId === schedule.id ? 'Running...' : 'Run now'}
                          </Button>
                        ) : null}
                        {canManage ? (
                          <Button
                            type="button"
                            variant="outline"
                            onClick={() => void onToggleSchedule(schedule)}
                          >
                            {schedule.enabled ? 'Disable' : 'Enable'}
                          </Button>
                        ) : null}
                        {canManage ? (
                          <Button
                            type="button"
                            variant="outline"
                            onClick={() => void onDeleteSchedule(schedule)}
                          >
                            Delete
                          </Button>
                        ) : null}
                        {jobHref ? (
                          <Button asChild type="button" variant="outline">
                            <Link to={jobHref}>View last job</Link>
                          </Button>
                        ) : null}
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
            </DashboardPanelSection>
          ) : null}
          </DirectoryPageShell>
        </CustomerScopeGate>
      </PageSectionStack>
    </PageChrome>
  );
}
