import { Link } from 'react-router-dom';
import { getCampaignIntegrationPanel } from '@/api/campaigns_api';
import type { IntegrationHealthRow } from '@/api/types';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
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
import { CopyButton } from '@/shell/copy_button';
import { DirectoryFilterForm, FilterField, FilterFormActions } from '@/shell/filter_panel';
import {
  formatIntegrationHealthSlug,
  formatIntegrationHealthStatus,
} from '@/domains/campaigns/editor/integration_health_labels';
import { campaignPanelError } from '@/domains/campaigns/editor/campaign_editor_shared';
import type { CampaignIntegrationPanelWorkspace } from '@/domains/campaigns/editor/use_campaign_integration_panel_workspace';

function integrationHealthBadgeVariant(
  status: string
): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (status.trim().toLowerCase()) {
    case 'ok':
      return 'default';
    case 'warn':
      return 'secondary';
    case 'fail':
      return 'destructive';
    default:
      return 'outline';
  }
}

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
    dryRunning,
    applyResult,
    dryRunResult,
    applyError,
    dryRunError,
    health,
    healthError,
    healthLoading,
    clickCopyURL,
    postbackCopyURL,
    statusIntegrationSchemaName,
    onLoadHealth,
    onApplyTemplates,
    onDryRunTemplates,
  } = workspace;

  return (
    <div className="grid gap-4">
      {statusIntegrationSchemaName ? (
        <p className="text-sm text-muted-foreground">
          Linked status preset: <strong>{statusIntegrationSchemaName}</strong>
        </p>
      ) : null}
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

      <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
        <FilterField htmlFor="apply-traffic-source" label="Apply template: traffic source">
          <Input
            id="apply-traffic-source"
            value={draftTrafficSource}
            onChange={(event) => setDraftTrafficSource(event.target.value)}
          />
        </FilterField>
        <FilterField htmlFor="apply-affiliate-network" label="Affiliate network">
          <Input
            id="apply-affiliate-network"
            value={draftAffiliateNetwork}
            onChange={(event) => setDraftAffiliateNetwork(event.target.value)}
          />
        </FilterField>
        <FilterField htmlFor="apply-tracking-domain" label="Tracking domain">
          <Input
            id="apply-tracking-domain"
            value={draftTrackingDomain}
            onChange={(event) => setDraftTrackingDomain(event.target.value)}
          />
        </FilterField>
        <FilterFormActions>
          <Button disabled={applying} onClick={onApplyTemplates} type="button">
            {applying ? 'Applying...' : 'Apply templates'}
          </Button>
          <Button disabled={dryRunning} onClick={onDryRunTemplates} type="button" variant="outline">
            {dryRunning ? 'Dry-running...' : 'Dry-run postback'}
          </Button>
          <Button disabled={healthLoading} onClick={onLoadHealth} type="button" variant="outline">
            {healthLoading ? 'Loading...' : 'Load health'}
          </Button>
        </FilterFormActions>
      </DirectoryFilterForm>

      {clickCopyURL || postbackCopyURL || panel?.browser_pixel_snippet ? (
        <div className={cn(adminChrome.panel, 'grid gap-3 p-4 text-sm')}>
          {clickCopyURL ? (
            <div className="flex items-center gap-2">
              <span className="font-medium">Click URL</span>
              <code className="min-w-0 flex-1 truncate text-xs">{clickCopyURL}</code>
              <CopyButton label="Click URL" value={clickCopyURL} />
            </div>
          ) : null}
          {postbackCopyURL ? (
            <div className="flex items-center gap-2">
              <span className="font-medium">Postback URL</span>
              <code className="min-w-0 flex-1 truncate text-xs">{postbackCopyURL}</code>
              <CopyButton label="Postback URL" value={postbackCopyURL} />
            </div>
          ) : null}
          {panel?.browser_pixel_snippet ? (
            <div className="grid gap-2">
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-medium">Browser pixel (track.js)</span>
                {panel.browser_pixel_first_party ? (
                  <Badge variant="secondary">First-party /_aed/track.js</Badge>
                ) : (
                  <Badge variant="outline">Tracker /static/track.js</Badge>
                )}
              </div>
              {panel.browser_pixel_script_url ? (
                <p className="text-xs text-muted-foreground break-all">
                  {panel.browser_pixel_script_url}
                </p>
              ) : null}
              <pre className="max-h-48 overflow-auto rounded border bg-muted/40 p-2 text-xs whitespace-pre-wrap">
                {panel.browser_pixel_snippet}
              </pre>
              <CopyButton label="Browser pixel snippet" value={panel.browser_pixel_snippet} />
              <p className="text-xs text-muted-foreground">
                Same-origin script when LANDER_PUBLIC_BASE_URL is set; POST /track still targets the
                tracker host (add lander origin to TRACK_CORS_ORIGINS or rely on auto-merge).
              </p>
            </div>
          ) : null}
        </div>
      ) : null}

      {applyError ? campaignPanelError(applyError, 'Apply templates failed') : null}
      {dryRunError ? campaignPanelError(dryRunError, 'Dry-run failed') : null}
      {healthError ? campaignPanelError(healthError, 'Could not load integration health') : null}
      {applyResult ? (
        <p className="text-sm text-muted-foreground" role="status">
          Templates applied for campaign {applyResult.campaign_id}.
          {applyResult.affiliate_status?.mappings_applied_count != null
            ? ` Status mappings: ${applyResult.affiliate_status.mappings_applied_count}.`
            : null}
        </p>
      ) : null}
      {dryRunResult?.postback_dry_run ? (
        <p className="text-sm text-muted-foreground" role="status">
          Dry-run {dryRunResult.postback_dry_run.ok ? 'succeeded' : 'failed'}
          {dryRunResult.postback_dry_run.rendered_url
            ? `: ${dryRunResult.postback_dry_run.rendered_url}`
            : dryRunResult.postback_dry_run.error
              ? `: ${dryRunResult.postback_dry_run.error}`
              : ''}
        </p>
      ) : null}
      {health ? (
        <div className="grid gap-2">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm font-medium">Integration health</span>
            <Badge variant={integrationHealthBadgeVariant(health.summary)}>
              {formatIntegrationHealthStatus(health.summary)}
            </Badge>
          </div>
          <DirectoryTable>
            <TableHeader>
              <TableRow>
                <DirectoryTableHead>Check</DirectoryTableHead>
                <DirectoryTableHead>Status</DirectoryTableHead>
                <DirectoryTableHead>Detail</DirectoryTableHead>
                <DirectoryTableHead>Fix</DirectoryTableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {health.rows?.map((row, index) => (
                <TableRow key={`health-${index}`}>
                  <TableCell>{formatIntegrationHealthSlug(row.slug ?? '')}</TableCell>
                  <TableCell>
                    <Badge variant={integrationHealthBadgeVariant(row.status ?? '')}>
                      {formatIntegrationHealthStatus(row.status ?? '')}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground">{row.message ?? ''}</TableCell>
                  <TableCell>
                    {row.fix_route ? (
                      <Link
                        className="text-primary underline-offset-4 hover:underline"
                        to={row.fix_route}
                      >
                        Open fix
                      </Link>
                    ) : null}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </DirectoryTable>
        </div>
      ) : null}
    </div>
  );
}
