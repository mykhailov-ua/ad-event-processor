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
          <Button
            disabled={dryRunning}
            onClick={onDryRunTemplates}
            type="button"
            variant="outline"
          >
            {dryRunning ? 'Dry-running...' : 'Dry-run postback'}
          </Button>
          <Button disabled={healthLoading} onClick={onLoadHealth} type="button" variant="outline">
            {healthLoading ? 'Loading...' : 'Load health'}
          </Button>
        </FilterFormActions>
      </DirectoryFilterForm>

      {(clickCopyURL || postbackCopyURL) ? (
        <div className={cn(adminChrome.panel, 'grid gap-3 p-4 text-sm')}>
          {clickCopyURL ? (
            <div className="flex items-center gap-2">
              <span className="font-medium">Click URL</span>
              <code className="min-w-0 flex-1 truncate font-mono text-xs">{clickCopyURL}</code>
              <CopyButton label="Click URL" value={clickCopyURL} />
            </div>
          ) : null}
          {postbackCopyURL ? (
            <div className="flex items-center gap-2">
              <span className="font-medium">Postback URL</span>
              <code className="min-w-0 flex-1 truncate font-mono text-xs">{postbackCopyURL}</code>
              <CopyButton label="Postback URL" value={postbackCopyURL} />
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
