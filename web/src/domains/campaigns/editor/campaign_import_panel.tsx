import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react';
import { useSearchParams } from 'react-router-dom';

import {
  createCampaignImportValidateJob,
  getCampaignImportValidateJob,
  importCampaign,
  importCampaignMigration,
  importCampaignMigrationPull,
  listMigrationSources,
  previewCampaignMigration,
  previewCampaignMigrationPull,
  validateCampaignImport,
} from '@/api/campaigns_api';
import { ApiError } from '@/api/client';
import type {
  ImportCampaignRequest,
  ImportCampaignResult,
  ImportMigrationResult,
  ImportValidateJobRequest,
  MigratePreviewRequest,
  MigratePullRequest,
  MigrationPreviewResult,
} from '@/api/types';
import { ErrorBlock } from '@/shell/error_block';
import { StubBanner } from '@/shell/stub_banner';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';
import { cn } from '@/lib/utils';

type SourceKind = ImportValidateJobRequest['source_kind'];
type PullSourceKind = MigratePullRequest['source_kind'];

const SOURCE_KINDS: SourceKind[] = [
  'keitaro_json',
  'keitaro_admin_api',
  'binom_json',
  'binom_report_api',
  'native_v1',
];

const PULL_SOURCE_KINDS: PullSourceKind[] = ['keitaro_admin_api', 'binom_report_api'];

function parsePayloadJson(raw: string): Record<string, unknown> | unknown[] {
  const parsed: unknown = JSON.parse(raw);
  if (parsed == null || (typeof parsed !== 'object' && !Array.isArray(parsed))) {
    throw new Error('Payload must be a JSON object or array.');
  }
  return parsed as Record<string, unknown> | unknown[];
}

function panelError(error: Error, title: string) {
  if (error instanceof ApiError && error.status === 501) {
    return <StubBanner title={`${title} unavailable`} message={error.message} />;
  }
  return <ErrorBlock title={title} message={error.message} />;
}

function ImportField({
  id,
  label,
  children,
  className,
}: {
  id: string;
  label: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <label className={cn('text-sm font-medium text-zinc-700 dark:text-zinc-300 flex flex-col gap-1 text-sm font-medium', className)} htmlFor={id}>
      {label}
      {children}
    </label>
  );
}

export function CampaignImportPanel() {
  const [searchParams] = useSearchParams();
  const { session } = useSession();
  const defaultCustomerId =
    searchParams.get('customer_id') ?? session?.default_customer_id ?? '';

  const [draftCustomerId, setDraftCustomerId] = useState(defaultCustomerId);
  const [draftSourceKind, setDraftSourceKind] = useState<SourceKind>('keitaro_json');
  const [draftPayload, setDraftPayload] = useState('{\n  \n}');
  const [draftJobId, setDraftJobId] = useState('');
  const [draftNamePrefix, setDraftNamePrefix] = useState('');
  const [draftPullBaseUrl, setDraftPullBaseUrl] = useState('');
  const [draftPullToken, setDraftPullToken] = useState('');
  const [draftPullSourceKind, setDraftPullSourceKind] = useState<PullSourceKind>('keitaro_admin_api');
  const [validating, setValidating] = useState(false);
  const [enqueueing, setEnqueueing] = useState(false);
  const [importing, setImporting] = useState(false);
  const [migrating, setMigrating] = useState(false);
  const [pullPreviewing, setPullPreviewing] = useState(false);
  const [pullImporting, setPullImporting] = useState(false);
  const [polling, setPolling] = useState(false);
  const [validateResult, setValidateResult] = useState<MigrationPreviewResult | undefined>();
  const [importResult, setImportResult] = useState<
    ImportCampaignResult | ImportMigrationResult | undefined
  >();
  const [pullPreview, setPullPreview] = useState<MigrationPreviewResult | undefined>();
  const [actionError, setActionError] = useState<Error | undefined>();
  const [pollToken, setPollToken] = useState(0);

  const sourcesResource = useResource(listMigrationSources, []);

  const jobResource = useResource(
    (signal) => {
      if (!draftJobId.trim()) {
        return Promise.resolve(undefined);
      }
      return getCampaignImportValidateJob(draftJobId.trim(), signal);
    },
    [draftJobId, pollToken],
  );

  const sourceLabels = useMemo(() => {
    const items = sourcesResource.data?.sources ?? [];
    const labels = new Map(items.map((item) => [item.kind, item.label]));
    return SOURCE_KINDS.map((kind) => ({
      kind,
      label: labels.get(kind) ?? kind,
    }));
  }, [sourcesResource.data?.sources]);

  const buildJobRequest = useCallback((): ImportValidateJobRequest => {
    const customerId = draftCustomerId.trim();
    if (!customerId) {
      throw new Error('Customer ID is required.');
    }
    return {
      customer_id: customerId,
      source_kind: draftSourceKind,
      payload: parsePayloadJson(draftPayload),
    };
  }, [draftCustomerId, draftPayload, draftSourceKind]);

  const buildSyncRequest = useCallback((): MigratePreviewRequest => {
    return {
      source_kind: draftSourceKind,
      payload: parsePayloadJson(draftPayload),
    };
  }, [draftPayload, draftSourceKind]);

  const buildPullRequest = useCallback((): MigratePullRequest => {
    const customerId = draftCustomerId.trim();
    if (!customerId) {
      throw new Error('Customer ID is required.');
    }
    const baseUrl = draftPullBaseUrl.trim();
    const apiToken = draftPullToken.trim();
    if (!baseUrl || !apiToken) {
      throw new Error('Pull base URL and API token are required.');
    }
    const body: MigratePullRequest = {
      customer_id: customerId,
      source_kind: draftPullSourceKind,
      base_url: baseUrl,
      api_token: apiToken,
    };
    const prefix = draftNamePrefix.trim();
    if (prefix) {
      body.name_prefix = prefix;
    }
    return body;
  }, [
    draftCustomerId,
    draftNamePrefix,
    draftPullBaseUrl,
    draftPullSourceKind,
    draftPullToken,
  ]);

  const onValidateSync = useCallback(async () => {
    setValidating(true);
    setActionError(undefined);
    setValidateResult(undefined);
    setImportResult(undefined);
    try {
      const result = await validateCampaignImport(buildSyncRequest());
      setValidateResult(result);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setValidating(false);
    }
  }, [buildSyncRequest]);

  const onPreviewMigration = useCallback(async () => {
    setValidating(true);
    setActionError(undefined);
    setValidateResult(undefined);
    setImportResult(undefined);
    try {
      const result = await previewCampaignMigration(buildSyncRequest());
      setValidateResult(result);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setValidating(false);
    }
  }, [buildSyncRequest]);

  const onImportMigration = useCallback(async () => {
    const customerId = draftCustomerId.trim();
    if (!customerId) {
      setActionError(new Error('Customer ID is required.'));
      return;
    }
    setMigrating(true);
    setActionError(undefined);
    setImportResult(undefined);
    try {
      const result = await importCampaignMigration(
        {
          customer_id: customerId,
          source_kind: draftSourceKind,
          payload: parsePayloadJson(draftPayload),
          name_prefix: draftNamePrefix.trim() || undefined,
        },
        crypto.randomUUID(),
      );
      setImportResult(result);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setMigrating(false);
    }
  }, [draftCustomerId, draftNamePrefix, draftPayload, draftSourceKind]);

  const onImportBundle = useCallback(async () => {
    const customerId = draftCustomerId.trim();
    if (!customerId) {
      setActionError(new Error('Customer ID is required.'));
      return;
    }
    setImporting(true);
    setActionError(undefined);
    setImportResult(undefined);
    try {
      const bundle = parsePayloadJson(draftPayload);
      if (Array.isArray(bundle)) {
        throw new Error('Import bundle must be a JSON object.');
      }
      const body: ImportCampaignRequest = {
        ...(bundle as ImportCampaignRequest),
        customer_id: customerId,
      };
      const result = await importCampaign(body, crypto.randomUUID());
      setImportResult(result);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setImporting(false);
    }
  }, [draftCustomerId, draftPayload]);

  const onPullPreview = useCallback(async () => {
    setPullPreviewing(true);
    setActionError(undefined);
    setPullPreview(undefined);
    setImportResult(undefined);
    try {
      const result = await previewCampaignMigrationPull(buildPullRequest());
      setPullPreview(result);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setPullPreviewing(false);
    }
  }, [buildPullRequest]);

  const onPullImport = useCallback(async () => {
    setPullImporting(true);
    setActionError(undefined);
    setImportResult(undefined);
    try {
      const result = await importCampaignMigrationPull(buildPullRequest(), crypto.randomUUID());
      setImportResult(result);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setPullImporting(false);
    }
  }, [buildPullRequest]);

  const onEnqueueJob = useCallback(async () => {
    setEnqueueing(true);
    setActionError(undefined);
    try {
      const request = buildJobRequest();
      const created = await createCampaignImportValidateJob(
        request,
        crypto.randomUUID(),
      );
      const nextId = created.job_id ?? created.id;
      if (!nextId) {
        throw new Error('job_id missing in create response');
      }
      setDraftJobId(nextId);
      setPollToken((value) => value + 1);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setEnqueueing(false);
    }
  }, [buildJobRequest]);

  const onPollJob = useCallback(() => {
    if (!draftJobId.trim()) {
      return;
    }
    setPolling(true);
    setPollToken((value) => value + 1);
    setPolling(false);
  }, [draftJobId]);

  const importedCampaignIds = useMemo(() => {
    if (!importResult) {
      return [];
    }
    if ('id' in importResult && importResult.id) {
      return [importResult.id];
    }
    if ('imported' in importResult && Array.isArray(importResult.imported)) {
      return importResult.imported
        .map((row) => row.id)
        .filter((value): value is string => Boolean(value));
    }
    return [];
  }, [importResult]);

  useEffect(() => {
    if (!defaultCustomerId || draftCustomerId.trim()) {
      return;
    }
    setDraftCustomerId(defaultCustomerId);
  }, [defaultCustomerId, draftCustomerId]);

  return (
    <div className="flex flex-col gap-4">
      <section className="rounded-md border border-zinc-200 p-3 dark:border-zinc-800">
        <header className="flex items-center justify-between gap-2">
          <h2 className="text-sm font-semibold">Import validate</h2>
          <p className="text-zinc-500 dark:text-zinc-400 text-sm">
            Validate external tracker payloads before migration import.
          </p>
        </header>

        <div className="mt-2 flex flex-col gap-3">
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
              className="min-h-[5rem] w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm dark:border-zinc-700 dark:bg-zinc-950 min-h-40 font-mono text-sm"
              value={draftPayload}
              onChange={(event) => setDraftPayload(event.target.value)}
            />
          </ImportField>

          <div className="flex flex-wrap gap-2">
            <div className="flex flex-wrap items-center gap-1" aria-label="Validate actions">
              <Button disabled={validating} onClick={onValidateSync} type="button">
                {validating ? 'Validating...' : 'Validate now'}
              </Button>
              <Button disabled={validating} onClick={onPreviewMigration} type="button" variant="outline">
                {validating ? 'Previewing...' : 'Migrate preview'}
              </Button>
              <Button disabled={enqueueing} onClick={onEnqueueJob} type="button" variant="secondary">
                {enqueueing ? 'Enqueueing...' : 'Enqueue validate job'}
              </Button>
            </div>
            <div className="flex flex-wrap items-center gap-1" aria-label="Import actions">
              <Button disabled={migrating} onClick={onImportMigration} type="button">
                {migrating ? 'Importing...' : 'Migrate import'}
              </Button>
              <Button disabled={importing} onClick={onImportBundle} type="button" variant="secondary">
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
              disabled={polling || !draftJobId.trim()}
              onClick={onPollJob}
              type="button"
              variant="outline"
            >
              Poll job
            </Button>
          </div>

          {jobResource.data?.status ? (
            <p className="text-sm text-zinc-500 dark:text-zinc-400 text-sm">
              Job status: <strong>{jobResource.data.status}</strong>
            </p>
          ) : null}

          {importedCampaignIds.length > 0 ? (
            <p className="text-sm text-zinc-500 dark:text-zinc-400 text-sm" role="status">
              Imported campaign ID(s):{' '}
              <strong className="font-mono">{importedCampaignIds.join(', ')}</strong>
            </p>
          ) : null}

          {sourcesResource.error ? panelError(sourcesResource.error, 'Could not load migration sources') : null}
          {actionError ? panelError(actionError, 'Import action failed') : null}
          {jobResource.error ? panelError(jobResource.error, 'Could not poll validate job') : null}

          {validateResult ? (
            <JsonDashboardView payload={validateResult as unknown as Record<string, unknown>} />
          ) : null}
        </div>
      </section>

      <section className="rounded-md border border-zinc-200 p-3 dark:border-zinc-800">
        <header className="flex items-center justify-between gap-2">
          <h2 className="text-sm font-semibold">Migrate pull</h2>
          <p className="text-zinc-500 dark:text-zinc-400 text-sm">
            Pull campaigns from Keitaro or Binom admin APIs.
          </p>
        </header>

        <div className="mt-2 flex flex-col gap-3">
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
            <Button disabled={pullPreviewing} onClick={onPullPreview} type="button" variant="outline">
              {pullPreviewing ? 'Previewing...' : 'Pull preview'}
            </Button>
            <Button disabled={pullImporting} onClick={onPullImport} type="button">
              {pullImporting ? 'Importing...' : 'Pull import'}
            </Button>
          </div>

          {pullPreview ? (
            <JsonDashboardView payload={pullPreview as unknown as Record<string, unknown>} />
          ) : null}
        </div>
      </section>
    </div>
  );
}
