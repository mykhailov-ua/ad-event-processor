import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import {
  BUNDLED_AFFILIATE_TEMPLATES,
  BUNDLED_TRAFFIC_TEMPLATES,
  fetchBundledTemplates,
  fetchIntegrationSchemas,
  importBundledTemplates,
} from '../helpers/integration_api.js';
import type {
  IntegrationSchemaDTO,
  IntegrationTemplateCatalogEntry,
} from '../types/integration.js';
import { to } from '../lib/to.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';

export function IntegrationsTemplatesImportPage() {
  const canWrite = can(auth.getUser()?.permissions ?? [], 'campaigns:write');
  const [catalog, setCatalog] = useState<IntegrationTemplateCatalogEntry[]>([]);
  const [schemas, setSchemas] = useState<IntegrationSchemaDTO[]>([]);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(true);
  const [importing, setImporting] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);
    const [catRes, schRes] = await Promise.all([
      to(fetchBundledTemplates()),
      to(fetchIntegrationSchemas()),
    ]);
    setLoading(false);
    if (catRes[1]) {
      setError(catRes[1]);
      return;
    }
    if (schRes[1]) {
      setError(schRes[1]);
      return;
    }
    setCatalog(catRes[0] ?? []);
    setSchemas(schRes[0] ?? []);
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const toggle = (name: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  };

  const importSelected = async () => {
    if (!canWrite || selected.size === 0) return;
    setImporting(true);
    const [, err] = await to(importBundledTemplates([...selected]));
    setImporting(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Import failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({
      title: 'Templates imported',
      message: `${selected.size} schema(s) upserted`,
    });
    setSelected(new Set());
    void reload();
  };

  const importedNames = new Set(schemas.map((s) => s.name));

  const presetGroups = [
    { title: 'Traffic sources', items: BUNDLED_TRAFFIC_TEMPLATES },
    { title: 'Affiliate networks', items: BUNDLED_AFFILIATE_TEMPLATES },
  ];

  return (
    <div className="page stack" data-testid="integration-templates-import-page">
      <header className="page-header">
        <h1 className="page-title">Integration templates</h1>
        <p className="text-muted text-sm">
          Import bundled YAML presets into Postgres. Apply them per campaign on the{' '}
          <Link to="/campaigns">Integration</Link> tab.
        </p>
      </header>

      {error ? <ErrorBlock error={error} /> : null}

      <section className="section-card stack">
        <h2 className="subsection-title">Bundled catalog</h2>
        {loading ? <p className="text-muted">Loading...</p> : null}
        {!loading && catalog.length === 0 ? (
          <p className="text-muted">No bundled templates on this deployment.</p>
        ) : null}
        {!loading &&
          presetGroups.map((group) => (
            <div key={group.title} className="stack" data-testid={`template-group-${group.title}`}>
              <h3 className="text-sm font-medium">{group.title}</h3>
              <ul className="list-plain">
                {group.items.map((item) => {
                  const inCatalog = catalog.some((c) => c.name === item.value);
                  const alreadyImported = importedNames.has(item.value);
                  return (
                    <li key={item.value} className="toolbar-row">
                      <label className="form-field form-field--inline">
                        <input
                          type="checkbox"
                          disabled={!canWrite || !inCatalog}
                          checked={selected.has(item.value)}
                          data-testid={`template-import-${item.value}`}
                          onChange={() => toggle(item.value)}
                        />
                        {item.label}
                        <span className="text-muted text-sm font-mono">{item.value}</span>
                      </label>
                      {alreadyImported ? (
                        <span
                          className="text-muted text-sm"
                          data-testid={`template-imported-${item.value}`}
                        >
                          imported
                        </span>
                      ) : null}
                      {!inCatalog ? (
                        <span className="text-muted text-sm">not in bundle</span>
                      ) : null}
                    </li>
                  );
                })}
              </ul>
            </div>
          ))}
        {canWrite ? (
          <Button
            label={importing ? 'Importing...' : 'Import selected'}
            variant="primary"
            size="sm"
            loading={importing}
            disabled={importing || selected.size === 0}
            data-testid="template-import-submit"
            onClick={() => void importSelected()}
          />
        ) : null}
      </section>

      <section className="section-card stack" data-testid="integration-schemas-panel">
        <h2 className="subsection-title">Imported schemas</h2>
        <div className="table-wrapper">
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">Name</th>
                <th scope="col">Kind</th>
                <th scope="col">Version</th>
                <th scope="col">Updated</th>
              </tr>
            </thead>
            <tbody>
              {schemas.length === 0 ? (
                <tr>
                  <td colSpan={4} className="text-muted">
                    No schemas imported yet.
                  </td>
                </tr>
              ) : (
                schemas.map((row) => (
                  <tr key={row.id}>
                    <td>{row.name}</td>
                    <td>{row.kind}</td>
                    <td>{row.version}</td>
                    <td>{row.updated_at}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
