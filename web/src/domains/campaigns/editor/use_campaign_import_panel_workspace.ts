// import/migration panel: validate job, direct import, and pull/preview migration lanes (enabled when overlay open).
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { toast } from 'sonner';

import {
  createCampaignImportValidateJob,
  importCampaign,
  importCampaignMigration,
  importCampaignMigrationPull,
  previewCampaignMigration,
  previewCampaignMigrationPull,
  validateCampaignImport,
} from '@/api/campaigns_api';
import type {
  ImportCampaignRequest,
  ImportCampaignResult,
  ImportMigrationResult,
  ImportValidateJobRequest,
  MigratePreviewRequest,
  MigratePullRequest,
  MigrationPreviewResult,
} from '@/api/types';
import {
  parsePayloadJson,
  SOURCE_KINDS,
  type SourceKind,
  type PullSourceKind,
} from '@/domains/campaigns/editor/campaign_import_panel_shared';
import { useCampaignImportPanelLoad } from '@/domains/campaigns/editor/use_campaign_import_panel_load';
import { newRandomUuid } from '@/lib/uuid';
import { useSession } from '@/hooks/use_session';
import { userErrorMessage } from '@/lib/admin_error';
import { requireNonEmpty, validationError } from '@/lib/admin_validation_error';
import { mutationError } from '@/lib/mutation_audit';

export function useCampaignImportPanelWorkspace(enabled: boolean) {
  const load = useCampaignImportPanelLoad(enabled);
  const [searchParams] = useSearchParams();
  const { session } = useSession();
  const defaultCustomerId = searchParams.get('customer_id') ?? session?.default_customer_id ?? '';

  const [draftCustomerId, setDraftCustomerId] = useState(defaultCustomerId);
  const [draftSourceKind, setDraftSourceKind] = useState<SourceKind>('keitaro_json');
  const [draftPayload, setDraftPayload] = useState('{\n  \n}');
  const { draftJobId, setDraftJobId, pollJob, onJobEnqueued } = load;
  const [draftNamePrefix, setDraftNamePrefix] = useState('');
  const [draftPullBaseUrl, setDraftPullBaseUrl] = useState('');
  const [draftPullToken, setDraftPullToken] = useState('');
  const [draftPullSourceKind, setDraftPullSourceKind] =
    useState<PullSourceKind>('keitaro_admin_api');
  const [validating, setValidating] = useState(false);
  const [enqueueing, setEnqueueing] = useState(false);
  const [importing, setImporting] = useState(false);
  const [migrating, setMigrating] = useState(false);
  const [pullPreviewing, setPullPreviewing] = useState(false);
  const [pullImporting, setPullImporting] = useState(false);
  const [validateResult, setValidateResult] = useState<MigrationPreviewResult | undefined>();
  const [importResult, setImportResult] = useState<
    ImportCampaignResult | ImportMigrationResult | { status: 'accepted' } | undefined
  >();
  const [pullPreview, setPullPreview] = useState<MigrationPreviewResult | undefined>();
  const [actionError, setActionError] = useState<Error | undefined>();

  const sourceLabels = useMemo(() => {
    const items = load.sources?.sources ?? [];
    const labels = new Map(items.map((item) => [item.kind, item.label]));
    return SOURCE_KINDS.map((kind) => ({
      kind,
      label: labels.get(kind) ?? kind,
    }));
  }, [load.sources?.sources]);

  const buildJobRequest = useCallback((): ImportValidateJobRequest => {
    const customerId = requireNonEmpty(draftCustomerId, 'Customer ID', 'customer_id');
    if (!customerId.ok) {
      throw customerId.error;
    }
    return {
      customer_id: customerId.value,
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
    const customerId = requireNonEmpty(draftCustomerId, 'Customer ID', 'customer_id');
    if (!customerId.ok) {
      throw customerId.error;
    }
    const baseUrl = draftPullBaseUrl.trim();
    const apiToken = draftPullToken.trim();
    if (!baseUrl || !apiToken) {
      throw validationError('Pull base URL and API token are required.');
    }
    const body: MigratePullRequest = {
      customer_id: customerId.value,
      source_kind: draftPullSourceKind,
      base_url: baseUrl,
      api_token: apiToken,
    };
    const prefix = draftNamePrefix.trim();
    if (prefix) {
      body.name_prefix = prefix;
    }
    return body;
  }, [draftCustomerId, draftNamePrefix, draftPullBaseUrl, draftPullSourceKind, draftPullToken]);

  const onValidateSync = useCallback(async () => {
    setValidating(true);
    setActionError(undefined);
    setValidateResult(undefined);
    setImportResult(undefined);
    try {
      const result = await validateCampaignImport(buildSyncRequest());
      setValidateResult(result);
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setActionError(nextError);
      toast.error(userErrorMessage(nextError));
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
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setActionError(nextError);
      toast.error(userErrorMessage(nextError));
    } finally {
      setValidating(false);
    }
  }, [buildSyncRequest]);

  const onImportMigration = useCallback(async () => {
    const customerId = requireNonEmpty(draftCustomerId, 'Customer ID', 'customer_id');
    if (!customerId.ok) {
      setActionError(customerId.error);
      return;
    }
    setMigrating(true);
    setActionError(undefined);
    setImportResult(undefined);
    try {
      const result = await importCampaignMigration(
        {
          customer_id: customerId.value,
          source_kind: draftSourceKind,
          payload: parsePayloadJson(draftPayload),
          name_prefix: draftNamePrefix.trim() || undefined,
        },
        newRandomUuid()
      );
      setImportResult(result);
      toast.success('Migration import accepted');
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setActionError(nextError);
      toast.error(userErrorMessage(nextError));
    } finally {
      setMigrating(false);
    }
  }, [draftCustomerId, draftNamePrefix, draftPayload, draftSourceKind]);

  const onImportBundle = useCallback(async () => {
    const customerId = requireNonEmpty(draftCustomerId, 'Customer ID', 'customer_id');
    if (!customerId.ok) {
      setActionError(customerId.error);
      return;
    }
    setImporting(true);
    setActionError(undefined);
    setImportResult(undefined);
    try {
      const bundle = parsePayloadJson(draftPayload);
      if (Array.isArray(bundle)) {
        throw validationError('Import bundle must be a JSON object.', { kind: 'format' });
      }
      const body: ImportCampaignRequest = {
        ...(bundle as ImportCampaignRequest),
        customer_id: customerId.value,
      };
      const result = await importCampaign(body, newRandomUuid());
      setImportResult(result);
      toast.success('Campaign import accepted');
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setActionError(nextError);
      toast.error(userErrorMessage(nextError));
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
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setActionError(nextError);
      toast.error(userErrorMessage(nextError));
    } finally {
      setPullPreviewing(false);
    }
  }, [buildPullRequest]);

  const onPullImport = useCallback(async () => {
    setPullImporting(true);
    setActionError(undefined);
    setImportResult(undefined);
    try {
      const result = await importCampaignMigrationPull(buildPullRequest(), newRandomUuid());
      setImportResult(result);
      toast.success('Pull import accepted');
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setActionError(nextError);
      toast.error(userErrorMessage(nextError));
    } finally {
      setPullImporting(false);
    }
  }, [buildPullRequest]);

  const onEnqueueJob = useCallback(async () => {
    setEnqueueing(true);
    setActionError(undefined);
    try {
      const request = buildJobRequest();
      const created = await createCampaignImportValidateJob(request, newRandomUuid());
      const nextId = created.job_id ?? created.id;
      if (!nextId) {
        throw validationError('job_id missing in create response');
      }
      onJobEnqueued(nextId);
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setActionError(nextError);
      toast.error(userErrorMessage(nextError));
    } finally {
      setEnqueueing(false);
    }
  }, [buildJobRequest, onJobEnqueued]);

  const onPollJob = useCallback(() => {
    pollJob();
  }, [pollJob]);

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

  useEffect(() => {
    if (enabled) {
      return;
    }
    setActionError(undefined);
    setDraftCustomerId(defaultCustomerId);
    setDraftSourceKind('keitaro_json');
    setDraftPayload('{\n  \n}');
    setDraftNamePrefix('');
    setDraftPullBaseUrl('');
    setDraftPullToken('');
    setDraftPullSourceKind('keitaro_admin_api');
    setValidateResult(undefined);
    setImportResult(undefined);
    setPullPreview(undefined);
    setValidating(false);
    setEnqueueing(false);
    setImporting(false);
    setMigrating(false);
    setPullPreviewing(false);
    setPullImporting(false);
    load.resetJobLane();
  }, [defaultCustomerId, enabled, load]);

  return {
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
    importResult,
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
  };
}

export type CampaignImportPanelWorkspace = ReturnType<typeof useCampaignImportPanelWorkspace>;
