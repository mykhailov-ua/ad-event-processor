import { useCallback, useEffect, useState } from 'react';
import {
  fetchPostbackCampaignStatus,
  fetchPostbackDlq,
  retryPostbackDlq,
  type PostbackCampaignStatus,
  type PostbackDlqRow,
} from '../helpers/integrations_api.js';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { PostbacksPanel } from '../ui/postbacks/postbacks_panel.js';

export function IntegrationsPostbacksPage() {
  const [campaignStatus, setCampaignStatus] = useState<PostbackCampaignStatus[]>([]);
  const [dlq, setDlq] = useState<PostbackDlqRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [reloadToken, setReloadToken] = useState(0);
  const [retryBusyId, setRetryBusyId] = useState<number | null>(null);

  const reload = useCallback(() => {
    setReloadToken((token) => token + 1);
  }, []);

  useEffect(() => {
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      const [statusResult, statusErr] = await to(fetchPostbackCampaignStatus(ctrl.signal));
      if (cancelled) return;
      if (statusErr && statusErr.name !== 'AbortError') {
        setError(statusErr);
        setLoading(false);
        return;
      }
      setCampaignStatus(statusResult ?? []);

      const [dlqResult, dlqErr] = await to(fetchPostbackDlq(ctrl.signal));
      if (cancelled) return;
      if (dlqErr && dlqErr.name !== 'AbortError') {
        setError(dlqErr);
        setLoading(false);
        return;
      }
      setDlq(dlqResult ?? []);
      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [reloadToken]);

  const onRetryDlq = useCallback(
    async (id: number) => {
      setRetryBusyId(id);
      try {
        await retryPostbackDlq(id);
        pushToastMessage({ title: 'Retry queued', message: `DLQ #${id}` });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Retry failed',
          message: err instanceof Error ? err.message : 'Retry failed',
        });
      } finally {
        setRetryBusyId(null);
      }
    },
    [reload]
  );

  return (
    <PostbacksPanel
      campaignStatus={campaignStatus}
      dlq={dlq}
      loading={loading}
      error={error}
      retryBusyId={retryBusyId}
      onRetryDlq={(id) => {
        void onRetryDlq(id);
      }}
    />
  );
}
