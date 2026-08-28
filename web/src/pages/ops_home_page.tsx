import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  buildDlqListUrl,
  buildDoctorUrl,
  buildOutboxListUrl,
  isApiStubError,
  reloadRoles,
  retryDlq,
  type DashboardSummary,
  type DLQList,
  type OpsDoctor,
  type OutboxList,
} from '../helpers/ops_api.js';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { useResource } from '../helpers/use_resource.js';
import { OpsHub } from '../ui/ops/ops_hub.js';

const SUMMARY_POLL_MS = 30_000;
const TAB_LIMIT = 25;

function parseTab(raw: string | null): string {
  if (raw === 'outbox' || raw === 'dlq') return raw;
  return 'overview';
}

export function OpsHomePage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const activeTab = parseTab(searchParams.get('tab'));

  const summaryUrl = '/api/v1/ops/dashboard/summary';
  const {
    data: summary,
    loading: summaryLoading,
    error: summaryError,
    reload: reloadSummary,
  } = useResource<DashboardSummary>(summaryUrl);

  const doctorUrl = activeTab === 'overview' ? buildDoctorUrl() : null;
  const { data: doctor } = useResource<OpsDoctor>(doctorUrl);

  const outboxUrl =
    activeTab === 'outbox'
      ? buildOutboxListUrl({ limit: TAB_LIMIT, cursor: searchParams.get('cursor') ?? undefined })
      : null;
  const {
    data: outbox,
    loading: outboxLoading,
    error: outboxError,
    reload: reloadOutbox,
  } = useResource<OutboxList>(outboxUrl);

  const dlqUrl =
    activeTab === 'dlq' ? buildDlqListUrl({ limit: TAB_LIMIT }) : null;
  const {
    data: dlq,
    loading: dlqLoading,
    error: dlqError,
    reload: reloadDlq,
  } = useResource<DLQList>(dlqUrl);

  const [rolesBusy, setRolesBusy] = useState(false);
  const [dlqRetryBusyId, setDlqRetryBusyId] = useState<string | null>(null);

  const summaryStub = useMemo(() => isApiStubError(summaryError), [summaryError]);

  useEffect(() => {
    if (activeTab !== 'overview') return undefined;
    const timer = window.setInterval(() => {
      reloadSummary();
    }, SUMMARY_POLL_MS);
    return () => window.clearInterval(timer);
  }, [activeTab, reloadSummary]);

  const onTabChange = useCallback(
    (tabId: string) => {
      const next = new URLSearchParams(searchParams);
      if (tabId === 'overview') {
        next.delete('tab');
      } else {
        next.set('tab', tabId);
      }
      next.delete('cursor');
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  const onReloadRoles = useCallback(() => {
    setRolesBusy(true);
    void (async () => {
      try {
        const result = await reloadRoles();
        pushToastMessage({
          title: 'RBAC reloaded',
          message: result.path ?? result.status ?? 'ok',
        });
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'RBAC reload failed',
          message: err instanceof Error ? err.message : 'Reload failed',
        });
      } finally {
        setRolesBusy(false);
      }
    })();
  }, []);

  const onRetryDlq = useCallback(
    (id: string) => {
      setDlqRetryBusyId(id);
      void (async () => {
        try {
          await retryDlq(id);
          pushToastMessage({ title: 'DLQ retry enqueued', message: id });
          reloadDlq();
          reloadOutbox();
        } catch (err) {
          if (err instanceof ConfirmCancelledError) return;
          pushToastMessage({
            title: 'DLQ retry failed',
            message: err instanceof Error ? err.message : 'Retry failed',
          });
        } finally {
          setDlqRetryBusyId(null);
        }
      })();
    },
    [reloadDlq, reloadOutbox]
  );

  return (
    <OpsHub
      activeTab={activeTab}
      onTabChange={onTabChange}
      summary={summary}
      doctor={doctor}
      outbox={outbox}
      dlq={dlq}
      summaryLoading={summaryLoading}
      tabLoading={activeTab === 'outbox' ? outboxLoading : dlqLoading}
      summaryError={summaryError}
      tabError={activeTab === 'outbox' ? outboxError : dlqError}
      summaryStub={summaryStub}
      onReloadRoles={onReloadRoles}
      rolesBusy={rolesBusy}
      onRetryDlq={onRetryDlq}
      dlqRetryBusyId={dlqRetryBusyId}
    />
  );
}
