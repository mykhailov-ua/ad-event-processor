import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import {
  createMarginGuardPolicy,
  fetchMarginGuardActivity,
  fetchMarginGuardPolicies,
  removeMarginGuardOverride,
  type MarginGuardActivity,
  type MarginGuardPolicy,
} from '../helpers/integrations_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { MarginGuardPanel } from '../ui/margin_guard/margin_guard_panel.js';

export function IntegrationsMarginGuardPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const campaignId = searchParams.get('campaign_id') ?? '';

  const [policies, setPolicies] = useState<MarginGuardPolicy[]>([]);
  const [activity, setActivity] = useState<MarginGuardActivity[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [reloadToken, setReloadToken] = useState(0);

  const reload = useCallback(() => {
    setReloadToken((token) => token + 1);
  }, []);

  useEffect(() => {
    if (!campaignId) {
      setPolicies([]);
      setActivity([]);
      setLoading(false);
      setError(null);
      return undefined;
    }
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      const [policiesResult, policiesErr] = await to(
        fetchMarginGuardPolicies(campaignId, ctrl.signal)
      );
      if (cancelled) return;
      if (policiesErr && policiesErr.name !== 'AbortError') {
        setError(policiesErr);
        setLoading(false);
        return;
      }
      setPolicies(policiesResult ?? []);

      const [activityResult, activityErr] = await to(
        fetchMarginGuardActivity(campaignId, ctrl.signal)
      );
      if (cancelled) return;
      if (activityErr && activityErr.name !== 'AbortError') {
        setError(activityErr);
        setLoading(false);
        return;
      }
      setActivity(activityResult ?? []);
      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [campaignId, reloadToken]);

  const onCampaignApply = useCallback(
    (nextCampaignId: string) => {
      const next = new URLSearchParams(searchParams);
      if (nextCampaignId) next.set('campaign_id', nextCampaignId);
      else next.delete('campaign_id');
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  const onCreatePolicy = useCallback(
    async (body: MarginGuardPolicy) => {
      setBusy(true);
      try {
        await createMarginGuardPolicy(body);
        pushToastMessage({ title: 'Policy saved', message: body.name ?? '' });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Save failed',
          message: err instanceof Error ? err.message : 'Save failed',
        });
      } finally {
        setBusy(false);
      }
    },
    [reload]
  );

  const onRemoveOverride = useCallback(
    async (placementId: string) => {
      if (!campaignId) return;
      setBusy(true);
      try {
        await removeMarginGuardOverride({ campaign_id: campaignId, placement_id: placementId });
        pushToastMessage({ title: 'Override removed', message: placementId });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Override failed',
          message: err instanceof Error ? err.message : 'Override failed',
        });
      } finally {
        setBusy(false);
      }
    },
    [campaignId, reload]
  );

  return (
    <MarginGuardPanel
      campaignId={campaignId}
      policies={policies}
      activity={activity}
      loading={loading}
      error={error}
      busy={busy}
      onCampaignApply={onCampaignApply}
      onCreatePolicy={(body) => {
        void onCreatePolicy(body);
      }}
      onRemoveOverride={(placementId) => {
        void onRemoveOverride(placementId);
      }}
    />
  );
}
