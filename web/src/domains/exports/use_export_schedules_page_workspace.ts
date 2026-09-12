import { useCallback, useMemo, useState } from 'react';
import { toast } from 'sonner';

import {
  createReportSchedule,
  deleteReportSchedule,
  listReportSchedules,
  runReportScheduleNow,
  updateReportSchedule,
} from '@/api/report_schedules_api';
import { useResource } from '@/api/use_resource';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import type { ReportSchedule } from '@/api/types';
import { useSession } from '@/hooks/use_session';
import { exportHubErrorMessage } from '@/domains/exports/export_hub_errors';
import type { ExportHubNotifyChannel } from '@/domains/exports/export_hub_notify_fields';
import type { ExportHubDestination } from '@/domains/exports/export_hub_recent';
import { sessionHasPermission } from '@/lib/session_permissions';
import { confirmDestructiveAction } from '@/lib/mutation_audit';
import {
  requireNonEmpty,
  toastValidationError,
  validationError,
  type AdminValidationError,
} from '@/lib/admin_validation_error';

export type ExportSchedulesDraft = {
  reportKey: string;
  cronExpr: string;
  format: 'csv' | 'xlsx' | 'json';
  destination: ExportHubDestination;
  ownerUserId: string;
  spreadsheetId: string;
  sheetTitle: string;
  fromOffsetDays: string;
  notifyChannel: ExportHubNotifyChannel;
  notifyEmail: string;
  notifyWebhookUrl: string;
  enabled: boolean;
};

const DEFAULT_DRAFT: ExportSchedulesDraft = {
  reportKey: 'pacing-drift',
  cronExpr: '0 6 * * *',
  format: 'csv',
  destination: 'download',
  ownerUserId: '',
  spreadsheetId: '',
  sheetTitle: '',
  fromOffsetDays: '7',
  notifyChannel: 'in_app',
  notifyEmail: '',
  notifyWebhookUrl: '',
  enabled: true,
};

function buildScheduleGoogleSheetBody(draft: ExportSchedulesDraft) {
  if (draft.destination !== 'google_sheet') {
    return undefined;
  }
  const spreadsheetId = draft.spreadsheetId.trim();
  return {
    mode: spreadsheetId ? ('append' as const) : ('create' as const),
    spreadsheet_id: spreadsheetId || undefined,
    sheet_title: draft.sheetTitle.trim() || undefined,
  };
}

export function useExportSchedulesPageWorkspace() {
  const { session, user } = useSession();
  const [customerId, setCustomerId] = useState(session?.default_customer_id ?? '');
  const [draft, setDraft] = useState<ExportSchedulesDraft>(DEFAULT_DRAFT);
  const [selectedScheduleId, setSelectedScheduleId] = useState<string | undefined>();
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const [saving, setSaving] = useState(false);
  const [runningScheduleId, setRunningScheduleId] = useState<string | undefined>();
  const [formValidationError, setFormValidationError] = useState<AdminValidationError | undefined>();

  const canManage = sessionHasPermission(user?.permissions, 'exports:run');

  const trimmedCustomerId = customerId.trim();
  const {
    data: schedules,
    error: schedulesError,
    fetching: schedulesFetching,
  } = useResource(
    (signal) => {
      if (!trimmedCustomerId) {
        return Promise.resolve(undefined);
      }
      return listReportSchedules({ customer_id: trimmedCustomerId }, signal);
    },
    [trimmedCustomerId, refreshToken]
  );

  const selectedSchedule = useMemo(
    () => schedules?.find((row) => row.id === selectedScheduleId),
    [schedules, selectedScheduleId]
  );

  const loadScheduleIntoDraft = useCallback((schedule: ReportSchedule) => {
    const offset =
      schedule.spec && typeof schedule.spec === 'object' && 'from_offset_days' in schedule.spec
        ? String(schedule.spec.from_offset_days ?? 7)
        : '7';
    setDraft({
      reportKey: schedule.report_key ?? 'placements',
      cronExpr: schedule.cron_expr ?? '0 6 * * *',
      format: (schedule.format as ExportSchedulesDraft['format']) || 'csv',
      destination: schedule.destination === 'google_sheet' ? 'google_sheet' : 'download',
      ownerUserId: schedule.owner_user_id ?? '',
      spreadsheetId: schedule.google_sheet?.spreadsheet_id ?? '',
      sheetTitle: schedule.google_sheet?.sheet_title ?? '',
      fromOffsetDays: offset,
      notifyChannel: schedule.notify?.channel ?? 'none',
      notifyEmail: schedule.notify?.email ?? '',
      notifyWebhookUrl: schedule.notify?.webhook_url ?? '',
      enabled: schedule.enabled ?? true,
    });
    setSelectedScheduleId(schedule.id);
    setFormValidationError(undefined);
  }, []);

  const refreshSchedules = useCoalescedBumpRefresh(bumpRefresh, schedulesFetching);

  const onSaveSchedule = useCallback(async () => {
    if (!canManage) {
      toast.error('exports:run permission required');
      return;
    }
    const customerCheck = requireNonEmpty(trimmedCustomerId, 'Customer ID', 'customer_id');
    if (!customerCheck.ok) {
      setFormValidationError(customerCheck.error);
      toastValidationError(customerCheck.error);
      return;
    }
    const reportKeyCheck = requireNonEmpty(draft.reportKey, 'Report key', 'report_key');
    if (!reportKeyCheck.ok) {
      setFormValidationError(reportKeyCheck.error);
      toastValidationError(reportKeyCheck.error);
      return;
    }
    const cronCheck = requireNonEmpty(draft.cronExpr, 'Cron expression', 'cron_expr');
    if (!cronCheck.ok) {
      setFormValidationError(cronCheck.error);
      toastValidationError(cronCheck.error);
      return;
    }
    const offsetDays = Number.parseInt(draft.fromOffsetDays.trim() || '7', 10);
    if (!Number.isFinite(offsetDays) || offsetDays < 1) {
      const error = validationError('Lookback days must be a positive integer.', {
        field: 'from_offset_days',
      });
      setFormValidationError(error);
      toastValidationError(error);
      return;
    }
    const ownerUserId =
      draft.destination === 'google_sheet'
        ? draft.ownerUserId.trim() || user?.id?.trim() || ''
        : draft.ownerUserId.trim();
    if (draft.destination === 'google_sheet' && !ownerUserId) {
      const error = validationError('Owner user ID is required for Google Sheet schedules.', {
        field: 'owner_user_id',
      });
      setFormValidationError(error);
      toastValidationError(error);
      return;
    }
    setSaving(true);
    setFormValidationError(undefined);
    try {
      const body = {
        customer_id: customerCheck.value,
        report_key: reportKeyCheck.value,
        cron_expr: cronCheck.value,
        format: draft.format,
        destination: draft.destination,
        owner_user_id: ownerUserId || undefined,
        google_sheet: buildScheduleGoogleSheetBody(draft),
        enabled: draft.enabled,
        spec: { from_offset_days: offsetDays },
        notify:
          draft.notifyChannel === 'none'
            ? undefined
            : {
                channel: draft.notifyChannel,
                email: draft.notifyEmail.trim() || undefined,
                webhook_url: draft.notifyWebhookUrl.trim() || undefined,
              },
      };
      if (selectedScheduleId) {
        await updateReportSchedule(selectedScheduleId, body);
        toast.success('Schedule updated');
      } else {
        const created = await createReportSchedule(body);
        setSelectedScheduleId(created.id);
        toast.success('Schedule created');
      }
      refreshSchedules();
    } catch (err: unknown) {
      toast.error(exportHubErrorMessage(err));
    } finally {
      setSaving(false);
    }
  }, [canManage, draft, refreshSchedules, selectedScheduleId, trimmedCustomerId, user?.id]);

  const onDeleteSchedule = useCallback(
    async (schedule: ReportSchedule) => {
      if (!canManage || !schedule.id) {
        return;
      }
      if (!confirmDestructiveAction(`Delete schedule ${schedule.report_key}?`)) {
        return;
      }
      try {
        await deleteReportSchedule(schedule.id);
        if (selectedScheduleId === schedule.id) {
          setSelectedScheduleId(undefined);
          setDraft(DEFAULT_DRAFT);
        }
        refreshSchedules();
        toast.success('Schedule deleted');
      } catch (err: unknown) {
        toast.error(exportHubErrorMessage(err));
      }
    },
    [canManage, refreshSchedules, selectedScheduleId]
  );

  const onToggleSchedule = useCallback(
    async (schedule: ReportSchedule) => {
      if (!canManage || !schedule.id) {
        return;
      }
      try {
        await updateReportSchedule(schedule.id, {
          report_key: schedule.report_key ?? 'placements',
          cron_expr: schedule.cron_expr ?? '0 6 * * *',
          format: schedule.format,
          destination: schedule.destination,
          owner_user_id: schedule.owner_user_id,
          google_sheet: schedule.google_sheet,
          notify: schedule.notify,
          spec: schedule.spec,
          enabled: !schedule.enabled,
        });
        refreshSchedules();
        toast.success(schedule.enabled ? 'Schedule disabled' : 'Schedule enabled');
      } catch (err: unknown) {
        toast.error(exportHubErrorMessage(err));
      }
    },
    [canManage, refreshSchedules]
  );

  const onRunScheduleNow = useCallback(
    async (schedule: ReportSchedule) => {
      if (!canManage || !schedule.id) {
        return;
      }
      setRunningScheduleId(schedule.id);
      try {
        const result = await runReportScheduleNow(schedule.id);
        refreshSchedules();
        const jobId = result.job_id;
        toast.success(jobId ? `Export job ${jobId} enqueued` : 'Export job enqueued');
      } catch (err: unknown) {
        toast.error(exportHubErrorMessage(err));
      } finally {
        setRunningScheduleId(undefined);
      }
    },
    [canManage, refreshSchedules]
  );

  const lastJobHref = useCallback((schedule: ReportSchedule) => {
    if (!schedule.last_job_id) {
      return undefined;
    }
    const params = new URLSearchParams();
    params.set('job_id', schedule.last_job_id);
    params.set('kind', 'report');
    if (schedule.customer_id) {
      params.set('customer_id', schedule.customer_id);
    }
    if (schedule.report_key) {
      params.set('report_key', schedule.report_key);
    }
    return `/exports?${params.toString()}`;
  }, []);

  return {
    customerId,
    setCustomerId,
    draft,
    setDraft,
    schedules: schedules ?? [],
    schedulesHasSnapshot: schedules != null,
    schedulesError,
    schedulesFetching,
    selectedScheduleId,
    selectedSchedule,
    loadScheduleIntoDraft,
    setSelectedScheduleId,
    resetDraft: () => {
      setSelectedScheduleId(undefined);
      setDraft(DEFAULT_DRAFT);
      setFormValidationError(undefined);
    },
    canManage,
    saving,
    runningScheduleId,
    formValidationError,
    onSaveSchedule,
    onDeleteSchedule,
    onToggleSchedule,
    onRunScheduleNow,
    lastJobHref,
    refreshSchedules,
  };
}
