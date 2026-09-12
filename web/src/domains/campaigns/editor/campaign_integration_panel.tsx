import { Link } from 'react-router-dom';
import { getCampaignIntegrationPanel } from '@/api/campaigns_api';
import type { IntegrationHealthRow } from '@/api/types';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Switch } from '@/components/ui/switch';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { adminChrome } from '@/lib/admin_chrome';
import { adminSpacing, adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';
import { CopyButton } from '@/shell/copy_button';
import { DirectoryFilterForm, FilterField, FilterFormActions } from '@/shell/filter_panel';
import {
  formatIntegrationHealthSlug,
  formatIntegrationHealthStatus,
} from '@/domains/campaigns/editor/integration_health_labels';
import { campaignPanelError } from '@/domains/campaigns/editor/campaign_editor_shared';
import { StubBanner } from '@/shell/stub_banner';
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
    schemeDrafts,
    setSchemeDrafts,
    schemeLoading,
    schemeSaving,
    schemeLoaded,
    schemeError,
    schemeSaveSuccess,
    onLoadStatusSchemes,
    onSaveStatusSchemes,
    outboundDrafts,
    setOutboundDrafts,
    outboundLoading,
    outboundSaving,
    outboundLoaded,
    outboundError,
    outboundSaveSuccess,
    outboundTestMessage,
    onLoadOutboundPostbacks,
    onSaveOutboundPostbacks,
    onTestOutboundPostback,
  } = workspace;

  return (
    <div className="grid gap-4">
      {statusIntegrationSchemaName ? (
        <p className={adminTypography.bodyMuted}>
          Linked status preset: <strong>{statusIntegrationSchemaName}</strong>
        </p>
      ) : null}
      {panelFetching && !panel ? (
        <p className={adminTypography.bodyMuted}>Loading panel...</p>
      ) : null}
      {loadError && !panel
        ? campaignPanelError(loadError, 'Could not load integration panel')
        : null}

      {panel ? (
        <div
          className={cn(adminChrome.panel, `grid ${adminSpacing.gap.lg} p-4`, adminTypography.body)}
        >
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
                  <li className="leading-relaxed" key={`integration-row-${index}`}>
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

      <div
        className={cn(adminChrome.panel, `grid ${adminSpacing.gap.lg} p-4`, adminTypography.body)}
      >
        <h3 className="font-medium">Outbound postbacks</h3>
        <p className={adminTypography.bodyMuted}>
          Multiple S2S URLs per campaign. When rows exist here, legacy single postback config is not
          used for conversion enqueue.
        </p>
        {outboundError ? campaignPanelError(outboundError, 'Outbound postbacks failed') : null}
        {outboundLoaded ? (
          <div className="grid gap-4">
            {outboundDrafts.map((draft, index) => (
              <DirectoryFilterForm
                key={`outbound-postback-${draft.id ?? index}`}
                layout="auto-fill"
                onSubmit={(event) => event.preventDefault()}
              >
                <FilterField htmlFor={`outbound-name-${index}`} label="Name">
                  <Input
                    id={`outbound-name-${index}`}
                    value={draft.name}
                    onChange={(event) => {
                      const next = [...outboundDrafts];
                      next[index] = { ...draft, name: event.target.value };
                      setOutboundDrafts(next);
                    }}
                  />
                </FilterField>
                <FilterField htmlFor={`outbound-priority-${index}`} label="Priority">
                  <Input
                    id={`outbound-priority-${index}`}
                    inputMode="numeric"
                    value={draft.priority}
                    onChange={(event) => {
                      const next = [...outboundDrafts];
                      next[index] = { ...draft, priority: event.target.value };
                      setOutboundDrafts(next);
                    }}
                  />
                </FilterField>
                <FilterField htmlFor={`outbound-provider-${index}`} label="Provider">
                  <Input
                    id={`outbound-provider-${index}`}
                    value={draft.provider}
                    onChange={(event) => {
                      const next = [...outboundDrafts];
                      next[index] = { ...draft, provider: event.target.value };
                      setOutboundDrafts(next);
                    }}
                  />
                </FilterField>
                <FilterField htmlFor={`outbound-url-${index}`} label="URL template">
                  <Input
                    id={`outbound-url-${index}`}
                    value={draft.url_template}
                    onChange={(event) => {
                      const next = [...outboundDrafts];
                      next[index] = { ...draft, url_template: event.target.value };
                      setOutboundDrafts(next);
                    }}
                  />
                </FilterField>
                <FilterField htmlFor={`outbound-trigger-kind-${index}`} label="Trigger kind">
                  <Select
                    value={draft.trigger_kind}
                    onValueChange={(value) => {
                      const next = [...outboundDrafts];
                      next[index] = { ...draft, trigger_kind: value };
                      setOutboundDrafts(next);
                    }}
                  >
                    <SelectTrigger id={`outbound-trigger-kind-${index}`}>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="conversion">conversion</SelectItem>
                      <SelectItem value="status">status</SelectItem>
                      <SelectItem value="goal">goal</SelectItem>
                    </SelectContent>
                  </Select>
                </FilterField>
                <FilterField htmlFor={`outbound-trigger-value-${index}`} label="Trigger value">
                  <Input
                    id={`outbound-trigger-value-${index}`}
                    value={draft.trigger_value}
                    onChange={(event) => {
                      const next = [...outboundDrafts];
                      next[index] = { ...draft, trigger_value: event.target.value };
                      setOutboundDrafts(next);
                    }}
                  />
                </FilterField>
                <FilterField htmlFor={`outbound-token-${index}`} label="API token">
                  <Input
                    id={`outbound-token-${index}`}
                    type="password"
                    value={draft.api_token}
                    onChange={(event) => {
                      const next = [...outboundDrafts];
                      next[index] = { ...draft, api_token: event.target.value };
                      setOutboundDrafts(next);
                    }}
                  />
                </FilterField>
                <FilterField htmlFor={`outbound-sample-${index}`} label="Sample %">
                  <Input
                    id={`outbound-sample-${index}`}
                    inputMode="numeric"
                    value={draft.sample_percent}
                    onChange={(event) => {
                      const next = [...outboundDrafts];
                      next[index] = { ...draft, sample_percent: event.target.value };
                      setOutboundDrafts(next);
                    }}
                  />
                </FilterField>
                <FilterField htmlFor={`outbound-delay-${index}`} label="Delay (sec)">
                  <Input
                    id={`outbound-delay-${index}`}
                    inputMode="numeric"
                    value={draft.delay_seconds}
                    onChange={(event) => {
                      const next = [...outboundDrafts];
                      next[index] = { ...draft, delay_seconds: event.target.value };
                      setOutboundDrafts(next);
                    }}
                  />
                </FilterField>
                <FilterField htmlFor={`outbound-enabled-${index}`} label="Enabled">
                  <Switch
                    id={`outbound-enabled-${index}`}
                    checked={draft.enabled}
                    onCheckedChange={(checked) => {
                      const next = [...outboundDrafts];
                      next[index] = { ...draft, enabled: checked };
                      setOutboundDrafts(next);
                    }}
                  />
                </FilterField>
                {draft.id ? (
                  <FilterFormActions>
                    <Button
                      onClick={() => onTestOutboundPostback(draft.id!)}
                      type="button"
                      variant="outline"
                    >
                      Test saved row
                    </Button>
                  </FilterFormActions>
                ) : null}
              </DirectoryFilterForm>
            ))}
            <div className="flex flex-wrap gap-2">
              <Button
                disabled={outboundSaving}
                onClick={() =>
                  setOutboundDrafts((rows) => [
                    ...rows,
                    {
                      name: '',
                      priority: String(rows.length),
                      enabled: true,
                      provider: 'webhook',
                      url_template: '',
                      api_token: '',
                      target_event: 'conversion',
                      trigger_kind: 'conversion',
                      trigger_value: '',
                      test_event_code: '',
                      sample_percent: '100',
                      delay_seconds: '0',
                    },
                  ])
                }
                type="button"
                variant="outline"
              >
                Add postback URL
              </Button>
              <Button disabled={outboundSaving} onClick={onSaveOutboundPostbacks} type="button">
                {outboundSaving ? 'Saving...' : 'Save outbound postbacks'}
              </Button>
            </div>
            {outboundSaveSuccess ? <p role="status">Outbound postbacks saved.</p> : null}
            {outboundTestMessage ? <p role="status">{outboundTestMessage}</p> : null}
          </div>
        ) : (
          <Button
            disabled={outboundLoading}
            onClick={onLoadOutboundPostbacks}
            type="button"
            variant="outline"
          >
            {outboundLoading ? 'Loading...' : 'Load outbound postbacks'}
          </Button>
        )}
      </div>

      <div
        className={cn(adminChrome.panel, `grid ${adminSpacing.gap.lg} p-4`, adminTypography.body)}
      >
        <h3 className="font-medium">Status scheme rules</h3>
        <p className={adminTypography.bodyMuted}>
          Ordered if/then rules applied on conversion ingest after status mapping. First matching
          rule wins.
        </p>
        {schemeError ? campaignPanelError(schemeError, 'Status scheme failed') : null}
        {schemeLoaded ? (
          <div className="grid gap-4">
            {schemeDrafts.map((draft, index) => (
              <DirectoryFilterForm
                key={`status-scheme-${index}`}
                layout="auto-fill"
                onSubmit={(event) => event.preventDefault()}
              >
                <FilterField htmlFor={`scheme-when-status-${index}`} label="When status">
                  <Input
                    id={`scheme-when-status-${index}`}
                    value={draft.when_status}
                    onChange={(event) => {
                      const next = [...schemeDrafts];
                      next[index] = { ...draft, when_status: event.target.value };
                      setSchemeDrafts(next);
                    }}
                  />
                </FilterField>
                <FilterField htmlFor={`scheme-when-goal-${index}`} label="When goal">
                  <Input
                    id={`scheme-when-goal-${index}`}
                    value={draft.when_goal}
                    onChange={(event) => {
                      const next = [...schemeDrafts];
                      next[index] = { ...draft, when_goal: event.target.value };
                      setSchemeDrafts(next);
                    }}
                  />
                </FilterField>
                <FilterField htmlFor={`scheme-internal-${index}`} label="Set internal status">
                  <Input
                    id={`scheme-internal-${index}`}
                    value={draft.set_internal_status}
                    onChange={(event) => {
                      const next = [...schemeDrafts];
                      next[index] = { ...draft, set_internal_status: event.target.value };
                      setSchemeDrafts(next);
                    }}
                  />
                </FilterField>
                <FilterField htmlFor={`scheme-goal-${index}`} label="Set goal name">
                  <Input
                    id={`scheme-goal-${index}`}
                    value={draft.set_goal_name}
                    onChange={(event) => {
                      const next = [...schemeDrafts];
                      next[index] = { ...draft, set_goal_name: event.target.value };
                      setSchemeDrafts(next);
                    }}
                  />
                </FilterField>
                <FilterField htmlFor={`scheme-payout-mode-${index}`} label="Payout mode">
                  <Select
                    value={draft.payout_mode}
                    onValueChange={(value) => {
                      const next = [...schemeDrafts];
                      next[index] = { ...draft, payout_mode: value };
                      setSchemeDrafts(next);
                    }}
                  >
                    <SelectTrigger id={`scheme-payout-mode-${index}`}>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="inherit">inherit</SelectItem>
                      <SelectItem value="fixed">fixed</SelectItem>
                      <SelectItem value="zero">zero</SelectItem>
                      <SelectItem value="pass_through">pass_through</SelectItem>
                      <SelectItem value="accumulate_payout">accumulate_payout</SelectItem>
                    </SelectContent>
                  </Select>
                </FilterField>
                <FilterField htmlFor={`scheme-payout-micro-${index}`} label="Payout micro">
                  <Input
                    id={`scheme-payout-micro-${index}`}
                    inputMode="numeric"
                    value={draft.payout_micro}
                    onChange={(event) => {
                      const next = [...schemeDrafts];
                      next[index] = { ...draft, payout_micro: event.target.value };
                      setSchemeDrafts(next);
                    }}
                  />
                </FilterField>
                <FilterField htmlFor={`scheme-fire-outbound-${index}`} label="Fire outbound">
                  <Switch
                    id={`scheme-fire-outbound-${index}`}
                    checked={draft.fire_outbound}
                    onCheckedChange={(checked) => {
                      const next = [...schemeDrafts];
                      next[index] = { ...draft, fire_outbound: checked };
                      setSchemeDrafts(next);
                    }}
                  />
                </FilterField>
                <FilterField htmlFor={`scheme-enabled-${index}`} label="Enabled">
                  <Switch
                    id={`scheme-enabled-${index}`}
                    checked={draft.enabled}
                    onCheckedChange={(checked) => {
                      const next = [...schemeDrafts];
                      next[index] = { ...draft, enabled: checked };
                      setSchemeDrafts(next);
                    }}
                  />
                </FilterField>
              </DirectoryFilterForm>
            ))}
            <div className="flex flex-wrap gap-2">
              <Button
                disabled={schemeSaving}
                onClick={() =>
                  setSchemeDrafts((rows) => [
                    ...rows,
                    {
                      when_status: '',
                      when_goal: '',
                      set_internal_status: '',
                      set_goal_name: '',
                      payout_mode: 'inherit',
                      payout_micro: '',
                      fire_outbound: true,
                      enabled: true,
                    },
                  ])
                }
                type="button"
                variant="outline"
              >
                Add rule
              </Button>
              <Button disabled={schemeSaving} onClick={onSaveStatusSchemes} type="button">
                {schemeSaving ? 'Saving...' : 'Save status scheme'}
              </Button>
            </div>
            {schemeSaveSuccess ? <p role="status">Status scheme saved.</p> : null}
          </div>
        ) : (
          <Button
            disabled={schemeLoading}
            onClick={onLoadStatusSchemes}
            type="button"
            variant="outline"
          >
            {schemeLoading ? 'Loading...' : 'Load status scheme'}
          </Button>
        )}
      </div>

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
        <div
          className={cn(adminChrome.panel, `grid ${adminSpacing.gap.lg} p-4`, adminTypography.body)}
        >
          {clickCopyURL ? (
            <div className="flex items-center gap-2">
              <span className="font-medium">Click URL</span>
              <code className={cn('min-w-0 flex-1 truncate', adminTypography.monoData)}>
                {clickCopyURL}
              </code>
              <CopyButton label="Click URL" value={clickCopyURL} />
            </div>
          ) : null}
          {postbackCopyURL ? (
            <div className="flex items-center gap-2">
              <span className="font-medium">Postback URL</span>
              <code className={cn('min-w-0 flex-1 truncate', adminTypography.monoData)}>
                {postbackCopyURL}
              </code>
              <CopyButton label="Postback URL" value={postbackCopyURL} />
            </div>
          ) : null}
          {panel?.browser_pixel_snippet ? (
            <div>
              <div>
                <span>Browser pixel (tag.js)</span>
                {panel.browser_pixel_first_party ? (
                  <Badge variant="secondary">First-party /_aed/tag.js</Badge>
                ) : (
                  <Badge variant="outline">Tracker /static/tag.js</Badge>
                )}
              </div>
              {panel.browser_pixel_script_url ? <p>{panel.browser_pixel_script_url}</p> : null}
              <pre>{panel.browser_pixel_snippet}</pre>
              <CopyButton label="Browser pixel snippet" value={panel.browser_pixel_snippet} />
              <p>
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
        <p role="status">
          Templates applied for campaign {applyResult.campaign_id}.
          {applyResult.affiliate_status?.mappings_applied_count != null
            ? ` Status mappings: ${applyResult.affiliate_status.mappings_applied_count}.`
            : null}
        </p>
      ) : null}
      {dryRunResult?.postback_dry_run ? (
        <p role="status">
          Dry-run {dryRunResult.postback_dry_run.ok ? 'succeeded' : 'failed'}
          {dryRunResult.postback_dry_run.rendered_url
            ? `: ${dryRunResult.postback_dry_run.rendered_url}`
            : dryRunResult.postback_dry_run.error
              ? `: ${dryRunResult.postback_dry_run.error}`
              : ''}
        </p>
      ) : null}
      {health ? (
        <div>
          {health.summary && health.summary.trim().toLowerCase() !== 'ok' ? (
            <StubBanner
              message="Resolve integration health checks before relying on delivery URLs and postbacks."
              title={`Integration health: ${formatIntegrationHealthStatus(health.summary)}`}
            />
          ) : null}
          <div>
            <span>Integration health</span>
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
                  <TableCell>{row.message ?? ''}</TableCell>
                  <TableCell>
                    {row.fix_route ? <Link to={row.fix_route}>Open fix</Link> : null}
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
