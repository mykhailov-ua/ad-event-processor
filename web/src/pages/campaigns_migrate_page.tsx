import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { isCustomerUuid } from '../helpers/customer_context.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ParseDecimal } from '../helpers/money.js';
import {
  fetchMigrationSources,
  importMigration,
  previewMigration,
  type MigrationPreviewResult,
  type MigrationSourceKind,
} from '../helpers/migration_api.js';
import { Breadcrumbs } from '../components/breadcrumbs.js';
import { Button, ButtonLink } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { PageHeader } from '../components/page_header.js';

const SOURCE_LABELS: Record<MigrationSourceKind, string> = {
  keitaro_json: 'Keitaro JSON export',
  binom_json: 'Binom JSON export',
  native_v1: 'Native campaign export v1',
};

/**
 * Operator wizard to preview and import campaigns from Keitaro/Binom JSON exports.
 */
export function CampaignsMigratePage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const queryCustomerId = searchParams.get('customer_id')?.trim() ?? '';

  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'campaigns:write');

  const [customerId, setCustomerId] = useState(queryCustomerId);
  const [sourceKind, setSourceKind] = useState<MigrationSourceKind>('keitaro_json');
  const [namePrefix, setNamePrefix] = useState('');
  const [budgetInput, setBudgetInput] = useState('100');
  const [payloadText, setPayloadText] = useState('');
  const [maxBytes, setMaxBytes] = useState(0);
  const [preview, setPreview] = useState<MigrationPreviewResult | null>(null);
  const [loadError, setLoadError] = useState<unknown>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    void (async () => {
      const [sources, err] = await to(fetchMigrationSources());
      if (err) {
        setLoadError(err);
        return;
      }
      setMaxBytes(sources?.max_payload_bytes ?? 0);
    })();
  }, []);

  const parsedPayload = useMemo(() => {
    const text = payloadText.trim();
    if (!text) return null;
    try {
      return JSON.parse(text) as unknown;
    } catch {
      return null;
    }
  }, [payloadText]);

  const runPreview = useCallback(async () => {
    if (!parsedPayload) {
      pushToastMessage({ title: 'Invalid JSON', message: 'Paste or upload valid export JSON.' });
      return;
    }
    setBusy(true);
    setActionError(null);
    const [result, err] = await to(previewMigration(sourceKind, parsedPayload));
    setBusy(false);
    if (err) {
      setActionError(mapServiceError(err).message);
      return;
    }
    setPreview(result);
  }, [parsedPayload, sourceKind]);

  const runImport = async () => {
    if (!canWrite || !isCustomerUuid(customerId) || !parsedPayload) return;
    const budgetMicro = ParseDecimal(budgetInput.trim());
    if (budgetMicro == null || budgetMicro <= 0) {
      pushToastMessage({ title: 'Invalid budget', message: 'Enter a positive default budget in USD.' });
      return;
    }
    setBusy(true);
    setActionError(null);
    const [result, err] = await to(
      importMigration(customerId, sourceKind, parsedPayload, {
        namePrefix: namePrefix.trim(),
        budgetLimitMicro: Math.round(budgetMicro * 1_000_000),
      })
    );
    setBusy(false);
    if (err) {
      setActionError(mapServiceError(err).message);
      return;
    }
    const count = result?.imported?.length ?? 0;
    pushToastMessage({
      title: 'Migration complete',
      message: `${count} campaign(s) imported.`,
    });
    if (count === 1 && result?.imported?.[0]?.id) {
      navigate(`/campaigns/${result.imported[0].id}`);
      return;
    }
    const params = new URLSearchParams({ customer_id: customerId });
    navigate(`/campaigns?${params.toString()}`);
  };

  const onFilePick = async (file: File | null) => {
    if (!file) return;
    const [text, err] = await to(file.text());
    if (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to read file');
      return;
    }
    setPayloadText(text);
    setPreview(null);
  };

  if (loadError) {
    return <ErrorBlock error={loadError} fallbackTitle="Failed to load migration sources" />;
  }

  return (
    <div className="page-stack" data-testid="campaigns-migrate-page">
      <PageHeader
        title="Migrate campaigns"
        desc="Upload a Keitaro or Binom JSON export, preview macro mapping, then import campaigns with traffic presets."
      />
      <Breadcrumbs
        items={[
          { label: 'Campaigns', href: '/campaigns' },
          { label: 'Migrate' },
        ]}
      />

      {!canWrite ? (
        <p className="text-muted text-sm">campaigns:write permission required to import.</p>
      ) : null}

      <section className="section-card stack">
        <label className="form-field" htmlFor="migrate-customer-id">
          Customer ID
          <input
            id="migrate-customer-id"
            className="form-input form-input--sm"
            data-testid="migrate-customer-id"
            value={customerId}
            onChange={(e) => setCustomerId(e.target.value.trim())}
            placeholder="UUID"
          />
        </label>
        <label className="form-field" htmlFor="migrate-source-kind">
          Source
          <select
            id="migrate-source-kind"
            className="form-input form-input--sm"
            data-testid="migrate-source-kind"
            value={sourceKind}
            onChange={(e) => {
              setSourceKind(e.target.value as MigrationSourceKind);
              setPreview(null);
            }}
          >
            {(Object.keys(SOURCE_LABELS) as MigrationSourceKind[]).map((kind) => (
              <option key={kind} value={kind}>
                {SOURCE_LABELS[kind]}
              </option>
            ))}
          </select>
        </label>
        <label className="form-field" htmlFor="migrate-json">
          Export JSON
          <textarea
            id="migrate-json"
            className="form-input"
            data-testid="migrate-json"
            rows={8}
            value={payloadText}
            onChange={(e) => {
              setPayloadText(e.target.value);
              setPreview(null);
            }}
            placeholder='{"campaigns":[...]}'
          />
        </label>
        {maxBytes > 0 ? (
          <p className="text-muted text-sm">Max payload: {maxBytes.toLocaleString()} bytes.</p>
        ) : null}
        <div className="button-row">
          <label className="button button--secondary button--sm">
            Upload file
            <input
              type="file"
              accept="application/json,.json"
              className="sr-only"
              data-testid="migrate-file-input"
              onChange={(e) => void onFilePick(e.target.files?.[0] ?? null)}
            />
          </label>
          <Button
            label={busy ? 'Previewing...' : 'Preview mapping'}
            variant="secondary"
            size="sm"
            loading={busy}
            disabled={busy || !parsedPayload}
            data-testid="migrate-preview-button"
            onClick={() => void runPreview()}
          />
        </div>
      </section>

      {preview ? (
        <section className="section-card stack" data-testid="migrate-preview-panel">
          <h2 className="text-heading-20">Preview</h2>
          <p className="text-muted text-sm">
            {preview.mapped_campaigns.length} campaign(s); {preview.warnings?.length ?? 0} warning(s).
          </p>
          {preview.warnings && preview.warnings.length > 0 ? (
            <ul className="list-plain text-sm" data-testid="migrate-warnings">
              {preview.warnings.slice(0, 20).map((w, i) => (
                <li key={`${w.slug}-${i}`}>
                  <code className="code-inline">{w.slug}</code>: {w.message}
                  {w.campaign_ref ? ` (${w.campaign_ref})` : ''}
                </li>
              ))}
            </ul>
          ) : null}
          <table className="data-table data-table--compact">
            <thead>
              <tr>
                <th>Name</th>
                <th>Network</th>
                <th>Template</th>
                <th>sub2</th>
              </tr>
            </thead>
            <tbody>
              {preview.mapped_campaigns.map((row) => (
                <tr key={row.ref}>
                  <td>{row.name}</td>
                  <td>{row.traffic_source_name ?? '-'}</td>
                  <td>
                    <code className="code-inline">{row.bundled_slug ?? '-'}</code>
                  </td>
                  <td>
                    <code className="code-inline">{row.click_query_params?.sub2 ?? '-'}</code>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <label className="form-field" htmlFor="migrate-name-prefix">
            Name prefix
            <input
              id="migrate-name-prefix"
              className="form-input form-input--sm"
              value={namePrefix}
              onChange={(e) => setNamePrefix(e.target.value)}
              placeholder="Optional prefix for imported names"
            />
          </label>
          <label className="form-field" htmlFor="migrate-default-budget">
            Default budget (USD) when export omits spend
            <input
              id="migrate-default-budget"
              className="form-input form-input--sm"
              value={budgetInput}
              onChange={(e) => setBudgetInput(e.target.value)}
            />
          </label>
          <div className="button-row">
            <Button
              label={busy ? 'Importing...' : 'Import campaigns'}
              variant="primary"
              size="sm"
              loading={busy}
              disabled={busy || !isCustomerUuid(customerId)}
              data-testid="migrate-import-button"
              onClick={() => void runImport()}
            />
            <ButtonLink href="/campaigns" label="Back to list" variant="secondary" size="sm" />
          </div>
        </section>
      ) : null}

      {actionError ? <p className="text-danger text-sm" data-testid="migrate-error">{actionError}</p> : null}

      <p className="text-muted text-sm">
        Hosted lander ZIPs are not imported in v1; re-upload under{' '}
        <Link to="/campaigns/flows">Campaign flows</Link>. Postback secrets are not copied; configure
        on <Link to="/integrations/postbacks">CAPI & Postbacks</Link>.
      </p>
    </div>
  );
}
