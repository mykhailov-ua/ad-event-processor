import { useCallback, useEffect, useState } from 'react';
import { ApiError } from '../helpers/api_client.js';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import {
  fetchIntegrationTemplates,
  importIntegrationTemplates,
  type IntegrationSchema,
  type IntegrationTemplateCatalogEntry,
} from '../helpers/integrations_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { TemplatesPanel } from '../ui/templates/templates_panel.js';

export function IntegrationsTemplatesImportPage() {
  const [catalog, setCatalog] = useState<IntegrationTemplateCatalogEntry[]>([]);
  const [imported, setImported] = useState<IntegrationSchema[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [stubUnavailable, setStubUnavailable] = useState(false);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    setStubUnavailable(false);
    void (async () => {
      const [result, err] = await to(fetchIntegrationTemplates(ctrl.signal));
      if (cancelled) return;
      if (err) {
        if (err.name === 'AbortError') return;
        if (err instanceof ApiError && (err.status === 501 || err.status === 503)) {
          setStubUnavailable(true);
        } else {
          setError(err);
        }
        setLoading(false);
        return;
      }
      setCatalog(result ?? []);
      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, []);

  const onImport = useCallback(async (names: string[]) => {
    setBusy(true);
    try {
      const rows = await importIntegrationTemplates({ names });
      setImported(rows);
      pushToastMessage({
        title: 'Templates imported',
        message: `${rows.length} schema(s)`,
      });
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({
        title: 'Import failed',
        message: err instanceof Error ? err.message : 'Import failed',
      });
    } finally {
      setBusy(false);
    }
  }, []);

  return (
    <TemplatesPanel
      catalog={catalog}
      imported={imported}
      loading={loading}
      error={error}
      stubUnavailable={stubUnavailable}
      busy={busy}
      onImport={(names) => {
        void onImport(names);
      }}
    />
  );
}
