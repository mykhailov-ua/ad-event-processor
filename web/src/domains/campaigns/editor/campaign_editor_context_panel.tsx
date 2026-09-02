import { useEffect, useState } from 'react';

import {
  getCampaignEditorShell,
  getCampaignFraudEditorSummary,
  getCampaignGeoSummary,
} from '@/api/campaigns_api';
import { ApiError } from '@/api/client';
import { ErrorBlock } from '@/shell/error_block';
import { StubBanner } from '@/shell/stub_banner';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { useResource } from '@/api/use_resource';

function panelError(error: Error, title: string) {
  if (error instanceof ApiError && error.status === 501) {
    return <StubBanner title={`${title} unavailable`} message={error.message} />;
  }
  return <ErrorBlock title={title} message={error.message} />;
}

type ContextLoadKey = 'geo' | 'fraud' | 'shell';

export function CampaignEditorContextPanel({ campaignId }: { campaignId: string }) {
  const [loadKey, setLoadKey] = useState<ContextLoadKey | undefined>();
  const [geoExpand, setGeoExpand] = useState(false);
  const [fraudPreview, setFraudPreview] = useState(false);
  const [loadToken, setLoadToken] = useState(0);

  useEffect(() => {
    setLoadKey(undefined);
    setGeoExpand(false);
    setFraudPreview(false);
    setLoadToken(0);
  }, [campaignId]);

  const geoResource = useResource(
    async (signal) => {
      if (loadKey !== 'geo') {
        return undefined;
      }
      return getCampaignGeoSummary(campaignId, { expand: geoExpand }, signal);
    },
    [campaignId, loadKey, geoExpand, loadToken],
  );

  const fraudResource = useResource(
    async (signal) => {
      if (loadKey !== 'fraud') {
        return undefined;
      }
      return getCampaignFraudEditorSummary(campaignId, { preview: fraudPreview }, signal);
    },
    [campaignId, loadKey, fraudPreview, loadToken],
  );

  const shellResource = useResource(
    async (signal) => {
      if (loadKey !== 'shell') {
        return undefined;
      }
      return getCampaignEditorShell(campaignId, signal);
    },
    [campaignId, loadKey, loadToken],
  );

  const onLoadGeo = () => {
    setLoadKey('geo');
    setLoadToken((token) => token + 1);
  };

  const onLoadFraud = () => {
    setLoadKey('fraud');
    setLoadToken((token) => token + 1);
  };

  const onLoadShell = () => {
    setLoadKey('shell');
    setLoadToken((token) => token + 1);
  };

  const busy =
    (loadKey === 'geo' && geoResource.fetching) ||
    (loadKey === 'fraud' && fraudResource.fetching) ||
    (loadKey === 'shell' && shellResource.fetching);

  return (
    <div className="grid gap-4">
      <div className="flex flex-wrap items-center gap-4">
        <div className="flex items-center gap-2">
          <Checkbox
            checked={geoExpand}
            disabled={busy}
            id="editor-context-geo-expand"
            onCheckedChange={(checked) => setGeoExpand(checked === true)}
          />
          <Label htmlFor="editor-context-geo-expand">Expand geo rows</Label>
        </div>
        <div className="flex items-center gap-2">
          <Checkbox
            checked={fraudPreview}
            disabled={busy}
            id="editor-context-fraud-preview"
            onCheckedChange={(checked) => setFraudPreview(checked === true)}
          />
          <Label htmlFor="editor-context-fraud-preview">Fraud preview query</Label>
        </div>
      </div>

      <div className="flex flex-wrap gap-2">
        <Button disabled={busy} onClick={onLoadGeo} type="button" variant="outline">
          {loadKey === 'geo' && geoResource.fetching ? 'Loading...' : 'Geo summary'}
        </Button>
        <Button disabled={busy} onClick={onLoadFraud} type="button" variant="outline">
          {loadKey === 'fraud' && fraudResource.fetching ? 'Loading...' : 'Fraud editor'}
        </Button>
        <Button disabled={busy} onClick={onLoadShell} type="button" variant="outline">
          {loadKey === 'shell' && shellResource.fetching ? 'Loading...' : 'Editor shell'}
        </Button>
      </div>

      {loadKey === 'geo' && geoResource.error
        ? panelError(geoResource.error, 'Could not load geo summary')
        : null}
      {loadKey === 'fraud' && fraudResource.error
        ? panelError(fraudResource.error, 'Could not load fraud editor summary')
        : null}
      {loadKey === 'shell' && shellResource.error
        ? panelError(shellResource.error, 'Could not load editor shell')
        : null}

      {loadKey === 'geo' && geoResource.data ? (
        <JsonDashboardView payload={geoResource.data} />
      ) : null}
      {loadKey === 'fraud' && fraudResource.data ? (
        <JsonDashboardView payload={fraudResource.data} />
      ) : null}
      {loadKey === 'shell' && shellResource.data ? (
        <JsonDashboardView payload={shellResource.data as unknown as Record<string, unknown>} />
      ) : null}
    </div>
  );
}
