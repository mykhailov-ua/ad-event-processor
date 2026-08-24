import { useCallback, useEffect, useState, Fragment } from 'react';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { to } from '../lib/to.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import {
  applyIntegrationSchema,
  fetchBundledTemplates,
  fetchIntegrationSchema,
  fetchIntegrationSchemas,
  importBundledTemplates,
} from '../helpers/integration_api.js';
import type {
  IntegrationSchemaDTO,
  IntegrationTemplateCatalogEntry,
} from '../types/integration.js';
import { IntegrationSchemaAuthorPanel } from '../components/integration_schema_author_panel.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { StatusBadge } from '../components/status_badge.js';

function TableSkeleton({ cols, rows = 4 }: { cols: number; rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }, (_, i) => (
        <tr key={`sk-${i}`} className="data-table__row--skeleton" aria-hidden="true">
          {Array.from({ length: cols }, (__, j) => (
            <td key={`sk-${i}-${j}`}>
              <span className="skeleton-bar" />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}

function SchemaDetailRow({ schemaId, onClose }: { schemaId: string; onClose: () => void }) {
  const [detail, setDetail] = useState<IntegrationSchemaDTO | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);

  useEffect(() => {
    let active = true;
    setLoading(true);
    void (async () => {
      const [row, err] = await to(fetchIntegrationSchema(schemaId));
      if (!active) return;
      setLoading(false);
      if (err) setError(err);
      else setDetail(row);
    })();
    return () => {
      active = false;
    };
  }, [schemaId]);

  return (
    <tr data-testid={`schema-detail-${schemaId}`}>
      <td colSpan={5}>
        <div className="stack gap-sm">
          <div className="row gap-sm items-center">
            <strong className="text-sm">Schema document</strong>
            <Button label="Close" variant="ghost" size="sm" onClick={onClose} />
          </div>
          {loading ? <p className="text-muted text-sm">Loading...</p> : null}
          {error ? <ErrorBlock error={error} /> : null}
          {detail ? (
            <pre className="code-block text-sm" data-testid="schema-detail-json">
              {JSON.stringify(detail.schema, null, 2)}
            </pre>
          ) : null}
        </div>
      </td>
    </tr>
  );
}

export function IntegrationsSchemasPage() {
  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'campaigns:write');

  const [schemas, setSchemas] = useState<IntegrationSchemaDTO[]>([]);
  const [catalog, setCatalog] = useState<IntegrationTemplateCatalogEntry[]>([]);
  const [selectedImport, setSelectedImport] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [importBusy, setImportBusy] = useState(false);
  const [applyBusy, setApplyBusy] = useState(false);
  const [applyCampaignId, setApplyCampaignId] = useState('');
  const [applySchemaId, setApplySchemaId] = useState('');
  const [expandedSchemaId, setExpandedSchemaId] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    const [schemaResult, catalogResult] = await Promise.all([
      to(fetchIntegrationSchemas()),
      to(fetchBundledTemplates()),
    ]);
    setLoading(false);
    const [schemaRows, schemaErr] = schemaResult;
    const [catalogRows, catalogErr] = catalogResult;
    if (schemaErr || catalogErr) {
      setError(schemaErr ?? catalogErr);
      return;
    }
    setSchemas(schemaRows ?? []);
    setCatalog(catalogRows ?? []);
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const onSchemaCreated = (created: IntegrationSchemaDTO) => {
    setSchemas((prev) => {
      const exists = prev.some((s) => s.id === created.id);
      if (exists) return prev.map((s) => (s.id === created.id ? created : s));
      return [created, ...prev];
    });
    setApplySchemaId(created.id);
  };

  const toggleImport = (name: string) => {
    setSelectedImport((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  };

  const runImport = async () => {
    if (!canWrite || importBusy || selectedImport.size === 0) return;
    setImportBusy(true);
    const [, err] = await to(importBundledTemplates([...selectedImport]));
    setImportBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Import failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({
      title: 'Templates imported',
      message: 'Schemas are ready to apply to campaigns.',
    });
    setSelectedImport(new Set());
    void load();
  };

  const runApply = async () => {
    if (!canWrite || applyBusy) return;
    const campaignId = applyCampaignId.trim();
    if (!campaignId || !applySchemaId) {
      pushToastMessage({
        title: 'Missing fields',
        message: 'Campaign ID and schema are required.',
      });
      return;
    }
    setApplyBusy(true);
    const [applied, err] = await to(applyIntegrationSchema(applySchemaId, campaignId));
    setApplyBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Apply failed', message: mapServiceError(err).message });
      return;
    }
    const hint = applied.target_url || applied.url_template || applied.kind || 'ok';
    pushToastMessage({ title: 'Schema applied', message: String(hint) });
  };

  return (
    <>
      <div className="page-header">
        <h1 className="page-header__title">Integration schemas</h1>
        <p className="text-muted text-sm">
          Bundled YAML presets for traffic sources and affiliate postbacks. Import templates
          or author JSON.
        </p>
      </div>

      {error ? <ErrorBlock error={error} /> : null}

      <IntegrationSchemaAuthorPanel canWrite={canWrite} onCreated={onSchemaCreated} />

      {canWrite ? (
        <div className="section-block stack">
          <h2 className="subsection-title">Import bundled templates</h2>
          <div className="stack" data-testid="integration-template-import">
            {catalog.map((entry) => (
              <label key={entry.name} className="form-field checkbox-field">
                <input
                  type="checkbox"
                  checked={selectedImport.has(entry.name)}
                  onChange={() => toggleImport(entry.name)}
                />{' '}
                <span className="font-mono">{entry.name}</span>
                <span className="text-muted text-sm">
                  {' '}
                  v{entry.version} , {entry.category} , {entry.kind}
                </span>
              </label>
            ))}
            {catalog.length === 0 && !loading ? (
              <p className="text-muted text-sm">No bundled templates on this deployment.</p>
            ) : null}
          </div>
          <Button
            label={importBusy ? 'Importing...' : 'Import selected'}
            variant="secondary"
            size="sm"
            loading={importBusy}
            disabled={importBusy || selectedImport.size === 0}
            onClick={() => void runImport()}
          />
        </div>
      ) : null}

      {canWrite ? (
        <div className="section-block stack">
          <h2 className="subsection-title">Apply schema to campaign</h2>
          <div className="form-row" data-testid="integration-schema-apply">
            <label className="form-field">
              Campaign ID
              <input
                className="form-input"
                value={applyCampaignId}
                data-testid="apply-schema-campaign-id"
                onChange={(e) => setApplyCampaignId(e.target.value)}
                placeholder="UUID"
              />
            </label>
            <label className="form-field">
              Schema
              <select
                className="form-input"
                value={applySchemaId}
                data-testid="apply-schema-select"
                onChange={(e) => setApplySchemaId(e.target.value)}
              >
                <option value="">Select schema...</option>
                {schemas.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.name} v{s.version} ({s.kind})
                  </option>
                ))}
              </select>
            </label>
          </div>
          <Button
            label={applyBusy ? 'Applying...' : 'Apply schema'}
            variant="primary"
            size="sm"
            loading={applyBusy}
            disabled={applyBusy}
            onClick={() => void runApply()}
          />
        </div>
      ) : null}

      <div className="section-block">
        <h2 className="subsection-title">Stored schemas</h2>
        <div className="table-wrapper">
          <table className="data-table" aria-label="Integration schemas">
            <thead>
              <tr>
                <th>Name</th>
                <th>Version</th>
                <th>Kind</th>
                <th>Updated</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {loading ? <TableSkeleton cols={5} /> : null}
              {!loading && schemas.length === 0 ? (
                <tr>
                  <td colSpan={5} className="data-table__empty">
                    No schemas yet - import or author one.
                  </td>
                </tr>
              ) : null}
              {schemas.map((row) => (
                <Fragment key={row.id}>
                  <tr data-testid={`schema-row-${row.id}`}>
                    <td className="font-mono">{row.name}</td>
                    <td>{String(row.version)}</td>
                    <td>
                      <StatusBadge status={row.kind} kind="service" />
                    </td>
                    <td>{row.updated_at ? new Date(row.updated_at).toLocaleString() : '-'}</td>
                    <td>
                      <Button
                        label={expandedSchemaId === row.id ? 'Hide JSON' : 'View JSON'}
                        variant="ghost"
                        size="sm"
                        data-testid={`schema-view-${row.id}`}
                        onClick={() =>
                          setExpandedSchemaId(expandedSchemaId === row.id ? '' : row.id)
                        }
                      />
                    </td>
                  </tr>
                  {expandedSchemaId === row.id ? (
                    <SchemaDetailRow schemaId={row.id} onClose={() => setExpandedSchemaId('')} />
                  ) : null}
                </Fragment>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </>
  );
}
