import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import {
  fetchFlows,
  fetchLanders,
  fetchOffers,
  type Flow,
  type Lander,
  type Offer,
} from '../helpers/flows_api.js';
import { can, canReadCampaigns } from '../helpers/permissions.js';
import { to } from '../lib/to.js';
import { ErrorBlock } from '../ui/system/error_block.js';
import { FlowsHub, type FlowsTab } from '../ui/flows/flows_hub.js';
import { FlowsPanel } from '../ui/flows/flows_panel.js';
import { LandersPanel } from '../ui/flows/landers_panel.js';
import { OffersPanel } from '../ui/flows/offers_panel.js';

const DEFAULT_TAB: FlowsTab = 'landers';

function parseTab(raw: string | null): FlowsTab {
  if (raw === 'offers' || raw === 'flows') return raw;
  return DEFAULT_TAB;
}

type FlowsData = {
  landers: Lander[];
  offers: Offer[];
  flows: Flow[];
};

export function CampaignFlowsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const activeTab = parseTab(searchParams.get('tab'));
  const permissions = auth.getUser()?.permissions ?? [];
  const canWrite = can(permissions, 'campaigns:write');

  const [data, setData] = useState<FlowsData>({ landers: [], offers: [], flows: [] });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [reloadToken, setReloadToken] = useState(0);

  const reload = useCallback(() => {
    setReloadToken((token) => token + 1);
  }, []);

  useEffect(() => {
    if (!canReadCampaigns(permissions)) return undefined;
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);

    void (async () => {
      const [landersResult, offersResult, flowsResult] = await Promise.all([
        to(fetchLanders(ctrl.signal)),
        to(fetchOffers(ctrl.signal)),
        to(fetchFlows(ctrl.signal)),
      ]);
      if (cancelled) return;

      const failures = [landersResult[1], offersResult[1], flowsResult[1]].filter(Boolean);
      if (failures.length > 0) {
        setError(failures[0]);
        setLoading(false);
        return;
      }

      setData({
        landers: landersResult[0] ?? [],
        offers: offersResult[0] ?? [],
        flows: flowsResult[0] ?? [],
      });
      setLoading(false);
    })();

    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [permissions, reloadToken]);

  const onTabChange = useCallback(
    (tab: FlowsTab) => {
      const next = new URLSearchParams(searchParams);
      if (tab === DEFAULT_TAB) {
        next.delete('tab');
      } else {
        next.set('tab', tab);
      }
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load campaign flows" />;
  }

  return (
    <FlowsHub activeTab={activeTab} onTabChange={onTabChange}>
      {activeTab === 'landers' ? (
        <LandersPanel
          items={data.landers}
          loading={loading}
          canWrite={canWrite}
          onReload={reload}
        />
      ) : null}
      {activeTab === 'offers' ? (
        <OffersPanel items={data.offers} loading={loading} canWrite={canWrite} onReload={reload} />
      ) : null}
      {activeTab === 'flows' ? (
        <FlowsPanel items={data.flows} loading={loading} canWrite={canWrite} onReload={reload} />
      ) : null}
    </FlowsHub>
  );
}
