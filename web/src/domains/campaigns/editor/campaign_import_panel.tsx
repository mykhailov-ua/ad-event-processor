import {
  campaignEditorActionsRowClass,
  campaignEditorSectionClass,
  campaignPanelError,
} from '@/domains/campaigns/editor/campaign_editor_shared';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { JsonPayloadView } from '@/shell/json_payload_view';
import { formatCampaignJsonKey } from '@/domains/campaigns/editor/campaign_json_labels';
import {
  ImportField,
  PULL_SOURCE_KINDS,
  type PullSourceKind,
  type SourceKind,
} from '@/domains/campaigns/editor/campaign_import_panel_shared';
import type { CampaignImportPanelWorkspace } from '@/domains/campaigns/editor/use_campaign_import_panel_workspace';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export function CampaignImportPanel({ workspace }: { workspace: CampaignImportPanelWorkspace }) {
  const {
    load,
    sourceLabels,
    draftCustomerId,
    setDraftCustomerId,
    draftSourceKind,
    setDraftSourceKind,
    draftPayload,
    setDraftPayload,
    draftJobId,
    setDraftJobId,
    draftNamePrefix,
    setDraftNamePrefix,
    draftPullBaseUrl,
    setDraftPullBaseUrl,
    draftPullToken,
    setDraftPullToken,
    draftPullSourceKind,
    setDraftPullSourceKind,
    validating,
    enqueueing,
    importing,
    migrating,
    pullPreviewing,
    pullImporting,
    validateResult,
    pullPreview,
    actionError,
    importedCampaignIds,
    onValidateSync,
    onPreviewMigration,
    onImportMigration,
    onImportBundle,
    onPullPreview,
    onPullImport,
    onEnqueueJob,
    onPollJob,
  } = workspace;

  return (
    <div className="grid gap-4">
      <section className={cn(campaignEditorSectionClass, 'gap-3')}>
        <header className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="m-0 text-sm font-semibold">Import validate</h2>
          <p className="text-muted-foreground text-sm">
            Validate external tracker payloads before migration import.
          </p>
        </header>

        <div className="grid gap-3">
          <ImportField id="import-customer-id" label="Customer ID">
            <Input
              id="import-customer-id"
              value={draftCustomerId}
              onChange={(event) => setDraftCustomerId(event.target.value)}
            />
          </ImportField>

          <div className="grid gap-3 sm:grid-cols-2">
            <ImportField id="import-source-kind" label="Source kind">
              <Select
                value={draftSourceKind}
                onValueChange={(value) => setDraftSourceKind(value as SourceKind)}
              >
                <SelectTrigger id="import-source-kind" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {sourceLabels.map(({ kind, label }) => (
                    <SelectItem key={kind} value={kind}>
                      {label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </ImportField>

            <ImportField id="import-name-prefix" label="Name prefix">
              <Input
                id="import-name-prefix"
                value={draftNamePrefix}
                onChange={(event) => setDraftNamePrefix(event.target.value)}
              />
            </ImportField>
          </div>

          <ImportField id="import-payload" label="Payload JSON">
            <textarea
              id="import-payload"
              className={cn(
                'min-h-40 w-full border border-border bg-background px-3 py-2 font-mono text-sm text-foreground',
                adminKit.controlRadius
              )}
              value={draftPayload}
              onChange={(event) => setDraftPayload(event.target.value)}
            />
          </ImportField>

          <div className="flex flex-wrap gap-2">
            <div className="flex flex-wrap items-center gap-1" aria-label="Validate actions">
              <Button disabled={validating} onClick={onValidateSync} type="button">
                {validating ? 'Validating...' : 'Validate now'}
              </Button>
              <Button
                disabled={validating}
                onClick={onPreviewMigration}
                type="button"
                variant="outline"
              >
                {validating ? 'Previewing...' : 'Migrate preview'}
              </Button>
              <Button
                disabled={enqueueing}
                onClick={onEnqueueJob}
                type="button"
                variant="secondary"
              >
                {enqueueing ? 'Enqueueing...' : 'Enqueue validate job'}
              </Button>
            </div>
            <div className="flex flex-wrap items-center gap-1" aria-label="Import actions">
              <Button disabled={migrating} onClick={onImportMigration} type="button">
                {migrating ? 'Importing...' : 'Migrate import'}
              </Button>
              <Button
                disabled={importing}
                onClick={onImportBundle}
                type="button"
                variant="secondary"
              >
                {importing ? 'Importing...' : 'Import bundle'}
              </Button>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-2 text-sm">
            <ImportField id="import-job-id" label="Validate job ID">
              <Input
                id="import-job-id"
                value={draftJobId}
                onChange={(event) => setDraftJobId(event.target.value)}
              />
            </ImportField>
            <Button
              disabled={load.jobFetching || !draftJobId.trim()}
              onClick={onPollJob}
              type="button"
              variant="outline"
            >
              Poll job
            </Button>
          </div>

          {load.job?.status ? (
            <p className="text-sm text-muted-foreground text-sm">
              Job status: <strong>{load.job.status}</strong>
            </p>
          ) : null}

          {importedCampaignIds.length > 0 ? (
            <p className="text-sm text-muted-foreground text-sm" role="status">
              Imported campaign ID(s):{' '}
              <strong className="font-mono">{importedCampaignIds.join(', ')}</strong>
            </p>
          ) : null}

          {load.sourcesError
            ? campaignPanelError(load.sourcesError, 'Could not load migration sources')
            : null}
          {actionError ? campaignPanelError(actionError, 'Import action failed') : null}
          {load.jobError ? campaignPanelError(load.jobError, 'Could not poll validate job') : null}

          {validateResult ? (
            <JsonPayloadView
              formatColumn={formatCampaignJsonKey}
              formatKey={formatCampaignJsonKey}
              payload={validateResult}
            />
          ) : null}
        </div>
      </section>

      <section className={cn(campaignEditorSectionClass, 'gap-3')}>
        <header className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="m-0 text-sm font-semibold">Migrate pull</h2>
          <p className="text-muted-foreground text-sm">
            Pull campaigns from Keitaro or Binom admin APIs.
          </p>
        </header>

        <div className="grid gap-3">
          <ImportField id="pull-source-kind" label="Pull source">
            <Select
              value={draftPullSourceKind}
              onValueChange={(value) => setDraftPullSourceKind(value as PullSourceKind)}
            >
              <SelectTrigger id="pull-source-kind" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {PULL_SOURCE_KINDS.map((kind) => (
                  <SelectItem key={kind} value={kind}>
                    {kind}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </ImportField>

          <ImportField id="pull-base-url" label="Base URL">
            <Input
              id="pull-base-url"
              value={draftPullBaseUrl}
              onChange={(event) => setDraftPullBaseUrl(event.target.value)}
            />
          </ImportField>

          <ImportField id="pull-api-token" label="API token">
            <Input
              id="pull-api-token"
              autoComplete="off"
              type="password"
              value={draftPullToken}
              onChange={(event) => setDraftPullToken(event.target.value)}
            />
          </ImportField>

          <div className="flex flex-wrap items-center gap-1">
            <Button
              disabled={pullPreviewing}
              onClick={onPullPreview}
              type="button"
              variant="outline"
            >
              {pullPreviewing ? 'Previewing...' : 'Pull preview'}
            </Button>
            <Button disabled={pullImporting} onClick={onPullImport} type="button">
              {pullImporting ? 'Importing...' : 'Pull import'}
            </Button>
          </div>

          {pullPreview ? (
            <JsonPayloadView
              formatColumn={formatCampaignJsonKey}
              formatKey={formatCampaignJsonKey}
              payload={pullPreview}
            />
          ) : null}
        </div>
      </section>
    </div>
  );
}
