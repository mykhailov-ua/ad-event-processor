import { useCallback, useEffect, useState } from 'react';

import {
  applyCampaignTemplates,
  getCampaignIntegrationHealth,
  getCampaignIntegrationPanel,
} from '@/api/campaigns_api';
import { ApiError } from '@/api/client';
import type { ApplyCampaignTemplatesResult, CampaignIntegrationHealth } from '@/api/types';
import { ErrorBlock } from '@/components/system/error_block';
import { StubBanner } from '@/components/system/stub_banner';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
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
import { useResource } from '@/hooks/use_resource';

function panelError(error: Error, title: string) {
  if (error instanceof ApiError && error.status === 501) {
    return <StubBanner title={`${title} unavailable`} message={error.message} />;
  }
  return <ErrorBlock title={title} message={error.message} />;
}

export function CampaignIntegrationPanel({ campaignId }: { campaignId: string }) {
  const [draftTrafficSource, setDraftTrafficSource] = useState('');
  const [draftAffiliateNetwork, setDraftAffiliateNetwork] = useState('');
  const [draftTrackingDomain, setDraftTrackingDomain] = useState('');
  const [applying, setApplying] = useState(false);
  const [applyResult, setApplyResult] = useState<ApplyCampaignTemplatesResult | undefined>();
  const [applyError, setApplyError] = useState<Error | undefined>();
  const [health, setHealth] = useState<CampaignIntegrationHealth | undefined>();
  const [healthError, setHealthError] = useState<Error | undefined>();
  const [healthLoading, setHealthLoading] = useState(false);

  const panelResource = useResource(
    (signal) => getCampaignIntegrationPanel(campaignId, signal),
    [campaignId],
  );

  useEffect(() => {
    setApplyResult(undefined);
    setApplyError(undefined);
    setHealth(undefined);
    setHealthError(undefined);
  }, [campaignId]);

  const onLoadHealth = useCallback(async () => {
    setHealthLoading(true);
    setHealthError(undefined);
    try {
      const result = await getCampaignIntegrationHealth(campaignId);
      setHealth(result);
    } catch (err) {
      setHealthError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setHealthLoading(false);
    }
  }, [campaignId]);

  const onApplyTemplates = useCallback(async () => {
    setApplying(true);
    setApplyError(undefined);
    setApplyResult(undefined);
    try {
      const body: Parameters<typeof applyCampaignTemplates>[1] = {};
      if (draftTrafficSource.trim()) {
        body.traffic_source = draftTrafficSource.trim();
      }
      if (draftAffiliateNetwork.trim()) {
        body.affiliate_network = draftAffiliateNetwork.trim();
      }
      if (draftTrackingDomain.trim()) {
        body.tracking_domain = draftTrackingDomain.trim();
      }
      const result = await applyCampaignTemplates(campaignId, body);
      setApplyResult(result);
    } catch (err) {
      setApplyError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setApplying(false);
    }
  }, [campaignId, draftAffiliateNetwork, draftTrackingDomain, draftTrafficSource]);

  const panel = panelResource.data;

  return (
    <div className="grid gap-4">
      {panelResource.fetching && !panel ? <p className="text-sm text-muted-foreground">Loading panel...</p> : null}
      {panelResource.error && !panel
        ? panelError(panelResource.error, 'Could not load integration panel')
        : null}

      {panel ? (
        <div className="ui-surface grid gap-3 p-3 text-sm">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-medium">{panel.overall_status_label}</span>
            <Badge variant="outline">{panel.overall_status}</Badge>
          </div>
          {(panel.rows ?? []).length > 0 ? (
            <ul className="grid gap-1">
              {panel.rows?.map((row, index) => (
                <li key={`integration-row-${index}`}>
                  {String((row as Record<string, unknown>).slug ?? `Row ${index + 1}`)}:{' '}
                  {String((row as Record<string, unknown>).message ?? '')}
                </li>
              ))}
            </ul>
          ) : null}
        </div>
      ) : null}

      <div className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4">
        <div className="grid gap-2">
          <Label htmlFor="apply-traffic-source">Traffic source</Label>
          <Input
            id="apply-traffic-source"
            value={draftTrafficSource}
            onChange={(event) => setDraftTrafficSource(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="apply-affiliate-network">Affiliate network</Label>
          <Input
            id="apply-affiliate-network"
            value={draftAffiliateNetwork}
            onChange={(event) => setDraftAffiliateNetwork(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="apply-tracking-domain">Tracking domain</Label>
          <Input
            id="apply-tracking-domain"
            value={draftTrackingDomain}
            onChange={(event) => setDraftTrackingDomain(event.target.value)}
          />
        </div>
        <Button disabled={applying} onClick={onApplyTemplates} type="button">
          {applying ? 'Applying...' : 'Apply templates'}
        </Button>
        <Button disabled={healthLoading} onClick={onLoadHealth} type="button" variant="outline">
          {healthLoading ? 'Loading...' : 'Load health'}
        </Button>
      </div>

      {applyError ? panelError(applyError, 'Apply templates failed') : null}
      {healthError ? panelError(healthError, 'Could not load integration health') : null}
      {applyResult ? (
        <p className="text-sm text-muted-foreground" role="status">
          Templates applied for campaign {applyResult.campaign_id}.
        </p>
      ) : null}
      {health ? (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Check</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Detail</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {health.rows?.map((row, index) => (
                <TableRow key={`health-${index}`}>
                  <TableCell>{row.slug ?? ''}</TableCell>
                  <TableCell>{row.status ?? ''}</TableCell>
                  <TableCell className="text-muted-foreground">{row.message ?? ''}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : null}
    </div>
  );
}
