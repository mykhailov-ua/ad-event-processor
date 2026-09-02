import { useEffect, useState } from 'react';

import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { PageChrome } from '@/shell/page_chrome';
import { CustomerScopeBar } from '@/shell/customer_scope_bar';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import { RowActionsMenu } from '@/shell/row_actions_menu';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
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
import type { ReportSchedule } from '@/api/types';
import { PortalsNav, portalsPanelError } from '@/domains/portals/portals_nav';
import { displayTimestamp } from '@/lib/display';

export type ReportSchedulesPanelProps = {
  schedules: ReportSchedule[];
  appliedCustomerId: string;
  draftCustomerId: string;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  draftReportKey: string;
  draftCronExpr: string;
  draftFormat: string;
  editRows: Record<
    string,
    { report_key: string; cron_expr: string; format: string; enabled: string }
  >;
  acting: boolean;
  actionError: Error | undefined;
  createSuccess: boolean;
  onDraftCustomerIdChange: (value: string) => void;
  onApplyCustomerScope: () => void;
  onDraftReportKeyChange: (value: string) => void;
  onDraftCronExprChange: (value: string) => void;
  onDraftFormatChange: (value: string) => void;
  onEditRowChange: (
    id: string,
    field: 'report_key' | 'cron_expr' | 'format' | 'enabled',
    value: string,
  ) => void;
  onCreateSchedule: () => void;
  onUpdateSchedule: (scheduleId: string) => void;
  onDeleteSchedule: (scheduleId: string) => void;
};

export function ReportSchedulesPanel({
  schedules,
  appliedCustomerId,
  draftCustomerId,
  fetching,
  error,
  hasSnapshot,
  draftReportKey,
  draftCronExpr,
  draftFormat,
  editRows,
  acting,
  actionError,
  createSuccess,
  onDraftCustomerIdChange,
  onApplyCustomerScope,
  onDraftReportKeyChange,
  onDraftCronExprChange,
  onDraftFormatChange,
  onEditRowChange,
  onCreateSchedule,
  onUpdateSchedule,
  onDeleteSchedule,
}: ReportSchedulesPanelProps) {
  const [createOpen, setCreateOpen] = useState(false);
  const [deleteScheduleId, setDeleteScheduleId] = useState<string | undefined>();

  useEffect(() => {
    if (createSuccess) {
      setCreateOpen(false);
    }
  }, [createSuccess]);

  if (!appliedCustomerId) {
    return (
      <PageChrome title="Report schedules">
        <PortalsNav />
        <CustomerScopeBar
          appliedCustomerId={appliedCustomerId}
          draftCustomerId={draftCustomerId}
          onApply={onApplyCustomerScope}
          onDraftCustomerIdChange={onDraftCustomerIdChange}
        />
        <EmptyState
          title="Customer required"
          description="Apply a customer ID to list and manage report schedules."
        />
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="Report schedules"
      actions={
        <Button className="text-sm" onClick={() => setCreateOpen(true)} type="button">
          Create schedule
        </Button>
      }
    >
      <PortalsNav />

      <CustomerScopeBar
        appliedCustomerId={appliedCustomerId}
        draftCustomerId={draftCustomerId}
        onApply={onApplyCustomerScope}
        onDraftCustomerIdChange={onDraftCustomerIdChange}
      />

      <Dialog onOpenChange={setCreateOpen} open={createOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Create schedule</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="schedule-report-key">Report key</Label>
              <Input
                id="schedule-report-key"
                className="text-sm"
                value={draftReportKey}
                onChange={(event) => onDraftReportKeyChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="schedule-cron-expr">Cron expression</Label>
              <Input
                id="schedule-cron-expr"
                className="text-sm"
                value={draftCronExpr}
                onChange={(event) => onDraftCronExprChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="schedule-format">Format</Label>
              <Input
                id="schedule-format"
                className="text-sm"
                value={draftFormat}
                onChange={(event) => onDraftFormatChange(event.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <PrimaryActionButton
              disabled={acting}
              loading={acting}
              onClick={onCreateSchedule}
              type="button"
            >
              Create
            </PrimaryActionButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        onOpenChange={(open) => {
          if (!open) {
            setDeleteScheduleId(undefined);
          }
        }}
        open={Boolean(deleteScheduleId)}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Delete schedule</DialogTitle>
          </DialogHeader>
          <DialogFooter className="gap-2 sm:gap-0">
            <SecondaryActionButton onClick={() => setDeleteScheduleId(undefined)} type="button">
              Cancel
            </SecondaryActionButton>
            <Button
              className="text-sm"
              disabled={acting}
              loading={acting}
              onClick={() => {
                if (deleteScheduleId) {
                  onDeleteSchedule(deleteScheduleId);
                  setDeleteScheduleId(undefined);
                }
              }}
              shape="pill"
              type="button"
              variant="destructive"
            >
              Delete schedule
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {fetching && !hasSnapshot && !error ? (
        <PageSkeleton />
      ) : error && !hasSnapshot ? (
        portalsPanelError(error, 'Could not load report schedules')
      ) : (
        <div aria-atomic="true" aria-live="polite">
          {schedules.length === 0 ? (
            <EmptyState
              variant="blank-slate"
              title="No schedules"
              description="Schedule recurring report exports for this customer."
              actionLabel="Create schedule"
              onAction={() => setCreateOpen(true)}
            />
          ) : (
            <div className="ui-table-frame">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Report</TableHead>
                    <TableHead>Format</TableHead>
                    <TableHead>Cron</TableHead>
                    <TableHead>Enabled</TableHead>
                    <TableHead>Updated</TableHead>
                    <TableHead />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {schedules.map((row) => {
                    const id = row.id ?? '';
                    const edit = editRows[id] ?? {
                      report_key: row.report_key ?? '',
                      cron_expr: row.cron_expr ?? '',
                      format: row.format ?? '',
                      enabled: row.enabled === false ? 'no' : 'yes',
                    };
                    return (
                      <TableRow key={id || row.report_key}>
                        <TableCell>
                          {id ? (
                            <Input
                              className="font-mono text-xs"
                              value={edit.report_key}
                              onChange={(event) =>
                                onEditRowChange(id, 'report_key', event.target.value)
                              }
                            />
                          ) : (
                            <span className="font-mono text-xs">{row.report_key ?? ''}</span>
                          )}
                        </TableCell>
                        <TableCell>
                          {id ? (
                            <Input
                              className="text-sm"
                              value={edit.format}
                              onChange={(event) =>
                                onEditRowChange(id, 'format', event.target.value)
                              }
                            />
                          ) : (
                            row.format ?? ''
                          )}
                        </TableCell>
                        <TableCell>
                          {id ? (
                            <Input
                              className="font-mono text-xs"
                              value={edit.cron_expr}
                              onChange={(event) =>
                                onEditRowChange(id, 'cron_expr', event.target.value)
                              }
                            />
                          ) : (
                            <span className="font-mono text-xs">{row.cron_expr ?? ''}</span>
                          )}
                        </TableCell>
                        <TableCell>
                          {id ? (
                            <Input
                              className="text-sm"
                              value={edit.enabled}
                              onChange={(event) =>
                                onEditRowChange(id, 'enabled', event.target.value)
                              }
                            />
                          ) : row.enabled === false ? (
                            'no'
                          ) : (
                            'yes'
                          )}
                        </TableCell>
                        <TableCell>{displayTimestamp(row.updated_at)}</TableCell>
                        <TableCell>
                          {id ? (
                            <RowActionsMenu ariaLabel="Schedule actions" disabled={acting}>
                              <DropdownMenuItem
                                disabled={acting}
                                onClick={() => onUpdateSchedule(id)}
                              >
                                Save
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                className="text-destructive focus:text-destructive"
                                disabled={acting}
                                onClick={() => setDeleteScheduleId(id)}
                              >
                                Delete
                              </DropdownMenuItem>
                            </RowActionsMenu>
                          ) : null}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>
          )}
        </div>
      )}

      {actionError ? portalsPanelError(actionError, 'Schedule action failed') : null}
      {error && hasSnapshot ? portalsPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
