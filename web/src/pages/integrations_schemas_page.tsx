import { useCallback, useEffect, useState } from 'react';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import {
  applyIntegrationSchema,
  createIntegrationSchema,
  fetchIntegrationSchema,
  fetchIntegrationSchemas,
  type IntegrationSchema,
} from '../helpers/integrations_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { SchemasPanel } from '../ui/schemas/schemas_panel.js';

export function IntegrationsSchemasPage() {
  const [items, setItems] = useState<IntegrationSchema[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [selected, setSelected] = useState<IntegrationSchema | null>(null);
  const [loading, setLoading] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [detailError, setDetailError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [reloadToken, setReloadToken] = useState(0);

  const reload = useCallback(() => {
    setReloadToken((token) => token + 1);
  }, []);

  useEffect(() => {
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      const [result, err] = await to(fetchIntegrationSchemas(ctrl.signal));
      if (cancelled) return;
      if (err && err.name !== 'AbortError') setError(err);
      else setItems(result ?? []);
      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [reloadToken]);

  useEffect(() => {
    if (!selectedId) {
      setSelected(null);
      setDetailError(null);
      return undefined;
    }
    const ctrl = new AbortController();
    let cancelled = false;
    setDetailLoading(true);
    setDetailError(null);
    void (async () => {
      const [result, err] = await to(fetchIntegrationSchema(selectedId, ctrl.signal));
      if (cancelled) return;
      if (err && err.name !== 'AbortError') setDetailError(err);
      else setSelected(result ?? null);
      setDetailLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [selectedId]);

  const onSelect = useCallback((id: string) => {
    setSelectedId(id);
  }, []);

  const onCreate = useCallback(
    async (name: string, version: number, schemaText: string) => {
      let schema: unknown;
      try {
        schema = JSON.parse(schemaText) as unknown;
      } catch {
        pushToastMessage({ title: 'Invalid JSON', message: 'Schema body must be valid JSON.' });
        return;
      }
      setBusy(true);
      try {
        const created = await createIntegrationSchema({ name, version, schema });
        pushToastMessage({ title: 'Schema created', message: `${name} v${version}` });
        if (created.id) setSelectedId(created.id);
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Create failed',
          message: err instanceof Error ? err.message : 'Create failed',
        });
      } finally {
        setBusy(false);
      }
    },
    [reload]
  );

  const onApply = useCallback(
    async (schemaId: string, campaignId: string) => {
      setBusy(true);
      try {
        await applyIntegrationSchema(schemaId, { campaign_id: campaignId });
        pushToastMessage({ title: 'Schema applied', message: campaignId });
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Apply failed',
          message: err instanceof Error ? err.message : 'Apply failed',
        });
      } finally {
        setBusy(false);
      }
    },
    []
  );

  return (
    <SchemasPanel
      items={items}
      selected={selected}
      loading={loading}
      detailLoading={detailLoading}
      error={error}
      detailError={detailError}
      busy={busy}
      onSelect={onSelect}
      onCreate={(name, version, schemaText) => {
        void onCreate(name, version, schemaText);
      }}
      onApply={(schemaId, campaignId) => {
        void onApply(schemaId, campaignId);
      }}
    />
  );
}
