import { getCampaignIntegrationPanel } from '@/api/campaigns_api';
import type { IntegrationHealthRow } from '@/api/types';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { adminChrome } from '@/lib/admin_chrome';
import { cn } from '@/lib/utils';
import { campaignPanelError } from '@/domains/campaigns/editor/campaign_editor_shared';
import {
  formatIntegrationHealthSlug,
  formatIntegrationHealthStatus,
} from '@/domains/campaigns/editor/integration_health_labels';
import type { CampaignIntegrationPanelWorkspace } from '@/domains/campaigns/editor/use_campaign_integration_panel_workspace';

export type CampaignIntegrationPanelProps = {
  panel: Awaited<ReturnType<typeof getCampaignIntegrationPanel>> | undefined;
  panelFetching: boolean;
  loadError: Error | undefined;
  workspace: CampaignIntegrationPanelWorkspace;
};

export function CampaignIntegrationPanel({
  panel,
  panelFetching,
  loadError,
  workspace,
}: CampaignIntegrationPanelProps) {
  const {
    draftTrafficSource,
    setDraftTrafficSource,
    draftAffiliateNetwork,
    setDraftAffiliateNetwork,
    draftTrackingDomain,
    setDraftTrackingDomain,
    applying,
    applyResult,
    applyError,
    health,
    healthError,
    healthLoading,
    onLoadHealth,
    onApplyTemplates,
  } = workspace;

  return (
    <div className="grid gap-4">
      {panelFetching && !panel ? (
        <p className="text-sm text-muted-foreground">Loading panel...</p>
      ) : null}
      {loadError && !panel
        ? campaignPanelError(loadError, 'Could not load integration panel')
        : null}

      {panel ? (
        <div className={cn(adminChrome.panel, 'grid gap-3 p-4 text-sm')}>
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-medium">{panel.overall_status_label}</span>
            <Badge variant="outline">{formatIntegrationHealthStatus(panel.overall_status)}</Badge>
          </div>
          {((panel.rows ?? []) as IntegrationHealthRow[]).length > 0 ? (
            <ul className="grid gap-2">
              {(panel.rows as IntegrationHealthRow[] | undefined)?.map((row, index) => {
                const slug = row.slug;
                const message = row.message;
                return (
                  <li key={`integration-row-${index}`} className="leading-relaxed">
                    <span className="font-medium text-foreground">
                      {formatIntegrationHealthSlug(slug)}
                    </span>
                    {message ? (
                      <span className="text-muted-foreground">{`: ${message}`}</span>
                    ) : null}
                  </li>
                );
              })}
            </ul>
          ) : null}
        </div>
      ) : null}

      <div className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4">
        <div className="grid gap-2">
          <Label htmlFor="apply-traffic-source">Apply template: traffic source</Label>
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

      {applyError ? campaignPanelError(applyError, 'Apply templates failed') : null}
      {healthError ? campaignPanelError(healthError, 'Could not load integration health') : null}
      {applyResult ? (
        <p className="text-sm text-muted-foreground" role="status">
          Templates applied for campaign {applyResult.campaign_id}.
        </p>
      ) : null}
      {health ? (
        <DirectoryTable>
          <TableHeader>
            <TableRow>
              <DirectoryTableHead>Check</DirectoryTableHead>
              <DirectoryTableHead>Status</DirectoryTableHead>
              <DirectoryTableHead>Detail</DirectoryTableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {health.rows?.map((row, index) => (
              <TableRow key={`health-${index}`}>
                <TableCell>{formatIntegrationHealthSlug(row.slug ?? '')}</TableCell>
                <TableCell>{formatIntegrationHealthStatus(row.status ?? '')}</TableCell>
                <TableCell className="text-muted-foreground">{row.message ?? ''}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </DirectoryTable>
      ) : null}
    </div>
  );
}
