import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import {
  createReportSchedule,
  deleteReportSchedule,
  fetchReportSchedules,
  type ReportSchedule,
} from '../helpers/report_schedules_api.js';
import { isBuyerBoundUser } from '../helpers/permissions.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { ReportSchedulesPanel } from '../ui/settings/report_schedules_panel.js';

export function SettingsReportSchedulesPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const user = auth.getUser();
  const buyerBound = isBuyerBoundUser(user?.role);
  const boundCustomerId = user?.customer_id ?? '';

  const customerId = searchParams.get('customer_id') ?? '';

  const [schedules, setSchedules] = useState<ReportSchedule[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [reloadToken, setReloadToken] = useState(0);

  useEffect(() => {
    if (buyerBound && boundCustomerId && !searchParams.get('customer_id')) {
      const next = new URLSearchParams(searchParams);
      next.set('customer_id', boundCustomerId);
      setSearchParams(next, { replace: true });
    }
  }, [buyerBound, boundCustomerId, searchParams, setSearchParams]);

  const reload = useCallback(() => {
    setReloadToken((token) => token + 1);
  }, []);

  useEffect(() => {
    if (!customerId) {
      setSchedules([]);
      setLoading(false);
      return undefined;
    }
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      const [result, err] = await to(fetchReportSchedules(customerId, ctrl.signal));
      if (cancelled) return;
      if (err && err.name !== 'AbortError') setError(err);
      else setSchedules(result ?? []);
      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [customerId, reloadToken]);

  const onCustomerApply = useCallback(
    (nextCustomerId: string) => {
      const next = new URLSearchParams(searchParams);
      if (nextCustomerId) next.set('customer_id', nextCustomerId);
      else next.delete('customer_id');
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  const onCreate = useCallback(
    async (body: {
      customer_id: string;
      report_key: string;
      cron_expr: string;
      format?: string;
      enabled?: boolean;
    }) => {
      setBusy(true);
      try {
        await createReportSchedule(body);
        pushToastMessage({ title: 'Schedule created', message: body.report_key });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Create failed',
          message: err instanceof Error ? err.message : 'Create failed',
        });
      } finally {
        setBusy(false);
      }
    },
    [reload]
  );

  const onDelete = useCallback(
    async (id: string) => {
      setBusy(true);
      try {
        await deleteReportSchedule(id);
        pushToastMessage({ title: 'Schedule deleted', message: id });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Delete failed',
          message: err instanceof Error ? err.message : 'Delete failed',
        });
      } finally {
        setBusy(false);
      }
    },
    [reload]
  );

  return (
    <ReportSchedulesPanel
      customerId={customerId}
      schedules={schedules}
      loading={loading}
      error={error}
      busy={busy}
      onCustomerApply={onCustomerApply}
      onCreate={(body) => {
        void onCreate(body);
      }}
      onDelete={(id) => {
        void onDelete(id);
      }}
    />
  );
}
