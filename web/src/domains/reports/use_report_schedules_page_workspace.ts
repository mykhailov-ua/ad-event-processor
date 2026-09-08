// L3 report delivery schedules CRUD: customer-scoped list + cron edit rows.
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';

import {
  createReportSchedule,
  deleteReportSchedule,
  listReportSchedules,
  updateReportSchedule,
} from '@/api/report_schedules_api';
import { confirmDestructiveAction, mutationError } from '@/lib/mutation_audit';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

type ScheduleEditRow = {
  report_key: string;
  cron_expr: string;
  format: string;
  enabled: string;
};

function parseEnabled(value: string): boolean | undefined {
  const trimmed = value.trim().toLowerCase();
  if (trimmed === 'yes' || trimmed === 'true' || trimmed === '1') {
    return true;
  }
  if (trimmed === 'no' || trimmed === 'false' || trimmed === '0') {
    return false;
  }
  return undefined;
}

export function useReportSchedulesPageWorkspace() {
  const {
    appliedCustomerId,
    draftCustomerId,
    setDraftCustomerId,
    applyCustomerScope,
    listQueryPending,
  } = useCustomerScope();

  const { refreshToken, bumpRefresh } = useRefreshToken();
  const shouldFetch = Boolean(appliedCustomerId);

  const {
    data,
    error,
    fetching,
    revalidating: listRevalidating,
  } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return listReportSchedules({ customer_id: appliedCustomerId }, signal);
    },
    [appliedCustomerId, shouldFetch, refreshToken]
  );

  const [editRows, setEditRows] = useState<Record<string, ScheduleEditRow>>({});
  const [acting, setActing] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>(undefined);
  const [createSuccess, setCreateSuccess] = useState(false);

  const listBusy = fetching || acting;
  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, listBusy);

  useEffect(() => {
    if (!data?.length) {
      return;
    }
    const next: Record<string, ScheduleEditRow> = {};
    for (const row of data) {
      if (row.id) {
        next[row.id] = {
          report_key: row.report_key ?? '',
          cron_expr: row.cron_expr ?? '',
          format: row.format ?? '',
          enabled: row.enabled === false ? 'no' : 'yes',
        };
      }
    }
    setEditRows(next);
  }, [data]);

  const [draftReportKey, setDraftReportKey] = useState('');
  const [draftCronExpr, setDraftCronExpr] = useState('');
  const [draftFormat, setDraftFormat] = useState('csv');

  const bumpReload = bumpRefreshCoalesced;

  const onCreateSchedule = useCallback(() => {
    const reportKey = draftReportKey.trim();
    const cronExpr = draftCronExpr.trim();
    if (!appliedCustomerId || !reportKey || !cronExpr) {
      setActionError(new Error('Customer, report key, and cron expression are required'));
      return;
    }
    setActing(true);
    setActionError(undefined);
    setCreateSuccess(false);
    void createReportSchedule({
      customer_id: appliedCustomerId,
      report_key: reportKey,
      cron_expr: cronExpr,
      format: draftFormat.trim() || 'csv',
      enabled: true,
    })
      .then(() => {
        setDraftReportKey('');
        setDraftCronExpr('');
        setCreateSuccess(true);
        toast.success('Report schedule created');
        bumpReload();
      })
      .catch((err: unknown) => {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setActing(false);
      });
  }, [appliedCustomerId, bumpReload, draftCronExpr, draftFormat, draftReportKey]);

  const onUpdateSchedule = useCallback(
    (scheduleId: string) => {
      const edit = editRows[scheduleId];
      if (!edit) {
        return;
      }
      setActing(true);
      setActionError(undefined);
      const enabled = parseEnabled(edit.enabled);
      void updateReportSchedule(scheduleId, {
        report_key: edit.report_key.trim(),
        cron_expr: edit.cron_expr.trim(),
        format: edit.format.trim(),
        enabled,
      })
        .then(() => {
          bumpReload();
        })
        .catch((err: unknown) => {
          setActionError(err instanceof Error ? err : new Error(String(err)));
        })
        .finally(() => {
          setActing(false);
        });
    },
    [bumpReload, editRows]
  );

  const onDeleteSchedule = useCallback(
    (scheduleId: string) => {
      if (acting) {
        return;
      }
      const label = editRows[scheduleId]?.report_key?.trim() || scheduleId;
      if (!confirmDestructiveAction(`Delete report schedule "${label}"?`)) {
        return;
      }
      setActing(true);
      setActionError(undefined);
      void deleteReportSchedule(scheduleId)
        .then(() => {
          toast.success('Report schedule deleted');
          bumpReload();
        })
        .catch((err: unknown) => {
          const nextError = mutationError(err);
          setActionError(nextError);
          toast.error(nextError.message);
        })
        .finally(() => {
          setActing(false);
        });
    },
    [acting, bumpReload, editRows]
  );

  const onEditRowChange = useCallback((id: string, field: keyof ScheduleEditRow, value: string) => {
    setEditRows((prev) => ({
      ...prev,
      [id]: {
        ...prev[id],
        [field]: value,
      },
    }));
  }, []);

  return {
    schedules: data,
    appliedCustomerId,
    draftCustomerId,
    fetching,
    listRevalidating: listRevalidating || listQueryPending,
    error,
    hasSnapshot: data != null,
    draftReportKey,
    draftCronExpr,
    draftFormat,
    editRows,
    acting,
    actionError,
    createSuccess,
    onDraftCustomerIdChange: setDraftCustomerId,
    onApplyCustomerScope: applyCustomerScope,
    onDraftReportKeyChange: setDraftReportKey,
    onDraftCronExprChange: setDraftCronExpr,
    onDraftFormatChange: setDraftFormat,
    onEditRowChange,
    onCreateSchedule,
    onUpdateSchedule,
    onDeleteSchedule,
  };
}
