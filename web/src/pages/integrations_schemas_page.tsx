import { useCallback, useMemo, useState } from 'react';
import { toast } from 'sonner';

import {
  applyIntegrationSchema,
  createIntegrationSchema,
  fetchIntegrationSnapshot,
  getIntegrationSchema,
  importIntegrationTemplates,
} from '@/api/integrations_api';
import type { ApplyIntegrationSchemaResponse, IntegrationSchema } from '@/api/types';
import {
  IntegrationsSchemas,
  type IntegrationsSchemasTab,
} from '@/domains/integrations/integrations_schemas';
import { useResource } from '@/hooks/use_resource';

export function IntegrationsSchemasPage() {
  const [tab, setTab] = useState<IntegrationsSchemasTab>('schemas');
  const [refreshToken, setRefreshToken] = useState(0);

  const { data, error, fetching } = useResource(
    (signal) => fetchIntegrationSnapshot(signal),
    [refreshToken],
  );

  const [draftName, setDraftName] = useState('');
  const [draftVersion, setDraftVersion] = useState('1');
  const [draftSchemaJson, setDraftSchemaJson] = useState('{}');
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<Error | undefined>();
  const [createSuccess, setCreateSuccess] = useState(false);

  const [draftSchemaId, setDraftSchemaId] = useState('');
  const [draftCampaignId, setDraftCampaignId] = useState('');
  const [applying, setApplying] = useState(false);
  const [applyError, setApplyError] = useState<Error | undefined>();
  const [applySuccess, setApplySuccess] = useState(false);
  const [applyResult, setApplyResult] = useState<ApplyIntegrationSchemaResponse | undefined>();

  const [draftTemplateNames, setDraftTemplateNames] = useState('');
  const [importing, setImporting] = useState(false);
  const [importError, setImportError] = useState<Error | undefined>();
  const [importSuccess, setImportSuccess] = useState(false);
  const [importedCount, setImportedCount] = useState<number | undefined>();

  const [viewedSchema, setViewedSchema] = useState<IntegrationSchema | undefined>();
  const [viewingSchema, setViewingSchema] = useState(false);
  const [viewSchemaError, setViewSchemaError] = useState<Error | undefined>();

  const schemas = useMemo(() => data?.schemas ?? [], [data?.schemas]);
  const templates = useMemo(() => data?.templates ?? [], [data?.templates]);

  const onPrefillFromSchema = useCallback((row: IntegrationSchema) => {
    setDraftSchemaId(row.id);
    setApplyError(undefined);
    setApplySuccess(false);
    setApplyResult(undefined);
  }, []);

  const onViewSchema = useCallback((row: IntegrationSchema) => {
    setViewedSchema(undefined);
    setViewSchemaError(undefined);
    setViewingSchema(true);
    void getIntegrationSchema(row.id)
      .then((schema) => {
        setViewedSchema(schema);
      })
      .catch((err: unknown) => {
        setViewSchemaError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setViewingSchema(false);
      });
  }, []);

  const onCloseViewedSchema = useCallback(() => {
    setViewedSchema(undefined);
    setViewSchemaError(undefined);
    setViewingSchema(false);
  }, []);

  const onCreate = useCallback(async () => {
    const name = draftName.trim();
    const version = Number.parseInt(draftVersion.trim(), 10);
    if (!name || !Number.isFinite(version)) {
      return;
    }
    let schema: Record<string, unknown>;
    try {
      schema = JSON.parse(draftSchemaJson) as Record<string, unknown>;
    } catch (err) {
      setCreateError(err instanceof Error ? err : new Error('Schema JSON is invalid'));
      return;
    }
    setCreating(true);
    setCreateError(undefined);
    setCreateSuccess(false);
    try {
      await createIntegrationSchema({ name, version, schema });
      setCreateSuccess(true);
      toast.success('Integration schema created');
      setRefreshToken((value) => value + 1);
    } catch (err) {
      setCreateError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [draftName, draftSchemaJson, draftVersion]);

  const onApply = useCallback(async () => {
    const schemaId = draftSchemaId.trim();
    const campaignId = draftCampaignId.trim();
    if (!schemaId || !campaignId) {
      return;
    }
    setApplying(true);
    setApplyError(undefined);
    setApplySuccess(false);
    setApplyResult(undefined);
    try {
      const result = await applyIntegrationSchema(schemaId, { campaign_id: campaignId });
      setApplyResult(result);
      setApplySuccess(true);
    } catch (err) {
      setApplyError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setApplying(false);
    }
  }, [draftCampaignId, draftSchemaId]);

  const onImport = useCallback(async () => {
    const names = draftTemplateNames
      .split(',')
      .map((value) => value.trim())
      .filter((value) => value.length > 0);
    setImporting(true);
    setImportError(undefined);
    setImportSuccess(false);
    setImportedCount(undefined);
    try {
      const result = await importIntegrationTemplates(names.length > 0 ? { names } : {});
      setImportedCount(result.length);
      setImportSuccess(true);
      setRefreshToken((value) => value + 1);
    } catch (err) {
      setImportError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setImporting(false);
    }
  }, [draftTemplateNames]);

  return (
    <IntegrationsSchemas
      tab={tab}
      onTabChange={setTab}
      schemas={schemas}
      templates={templates}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
      createForm={{
        draftName,
        draftVersion,
        draftSchemaJson,
        creating,
        createError,
        createSuccess,
        onDraftNameChange: setDraftName,
        onDraftVersionChange: setDraftVersion,
        onDraftSchemaJsonChange: setDraftSchemaJson,
        onCreate: () => {
          void onCreate();
        },
      }}
      applyForm={{
        draftSchemaId,
        draftCampaignId,
        applying,
        applyError,
        applySuccess,
        applyResult,
        onDraftSchemaIdChange: setDraftSchemaId,
        onDraftCampaignIdChange: setDraftCampaignId,
        onApply: () => {
          void onApply();
        },
        onPrefillFromSchema,
      }}
      importForm={{
        draftTemplateNames,
        importing,
        importError,
        importSuccess,
        importedCount,
        onDraftTemplateNamesChange: setDraftTemplateNames,
        onImport: () => {
          void onImport();
        },
      }}
      viewSchema={{
        schema: viewedSchema,
        fetching: viewingSchema,
        error: viewSchemaError,
        onView: onViewSchema,
        onClose: onCloseViewedSchema,
      }}
    />
  );
}
