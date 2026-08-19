import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { to } from '../lib/to.js';
import {
  fetchPostbackConfig,
  fetchPostbackDlq,
  retryPostbackDlq,
  savePostbackConfig,
  testPostbackConfig,
  type PostbackDlqRow,
  type PostbackDryRunResult,
} from '../helpers/postback_api.js';
import {
  normalizePostbackProvider,
  postbackProviderIds,
  POSTBACK_PROVIDER_UI,
  type PostbackProvider,
} from '../helpers/postback_provider_ui.js';
import {
  AFFILIATE_POSTBACK_PRESETS,
  affiliatePostbackById,
} from '../models/affiliate_postback_presets.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { Button } from './button.js';

export type CampaignPostbackSectionProps = {
  campaignId: string;
  canWrite: boolean;
};

function copyText(label: string, text: string): void {
  navigator.clipboard
    ?.writeText(text)
    .then(() => {
      pushToastMessage({ title: 'Copied', message: `${label} copied to clipboard` });
    })
    .catch(() => {
      pushToastMessage({ title: 'Copy failed', message: text || '(empty)' });
    });
}

export function CampaignPostbackSection({ campaignId, canWrite }: CampaignPostbackSectionProps) {
  const [provider, setProvider] = useState<PostbackProvider>('webhook');
  const [urlTemplate, setUrlTemplate] = useState('');
  const [apiToken, setApiToken] = useState('');
  const [targetEvent, setTargetEvent] = useState('conversion');
  const [testEventCode, setTestEventCode] = useState('');
  const [affiliatePresetId, setAffiliatePresetId] = useState('');
  const [dlq, setDlq] = useState<PostbackDlqRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [dryRunning, setDryRunning] = useState(false);
  const [dryRunResult, setDryRunResult] = useState<PostbackDryRunResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    const [cfg, dlqRes] = await Promise.all([
      fetchPostbackConfig(campaignId),
      fetchPostbackDlq(campaignId),
    ]);
    if (cfg) {
      setProvider(normalizePostbackProvider(String(cfg.provider || 'webhook')));
      setUrlTemplate(String(cfg.url_template || ''));
      setTargetEvent(String(cfg.target_event || 'conversion'));
      setTestEventCode(String(cfg.test_event_code || ''));
    }
    setDlq(dlqRes);
    setLoading(false);
  }, [campaignId]);

  useEffect(() => {
    void load();
  }, [load]);

  const save = async () => {
    if (!canWrite) return;
    setSaving(true);
    setError(null);
    const [, err] = await to(
      savePostbackConfig(campaignId, {
        provider,
        url_template: urlTemplate,
        api_token: apiToken || undefined,
        target_event: targetEvent,
        test_event_code: testEventCode || undefined,
      })
    );
    setSaving(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      setError(mapServiceError(err).message);
      return;
    }
    setApiToken('');
    pushToastMessage({ title: 'Postback saved', message: 'Configuration updated' });
    void load();
  };

  const retry = async (rowId: string | number) => {
    const [, err] = await to(retryPostbackDlq(rowId));
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Retry failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({ title: 'Retry queued', message: `DLQ #${rowId}` });
    void load();
  };

  const dryRun = async () => {
    if (!canWrite) return;
    setDryRunning(true);
    setDryRunResult(null);
    const [res, err] = await to(testPostbackConfig(campaignId));
    setDryRunning(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Dry-run failed', message: mapServiceError(err).message });
      return;
    }
    setDryRunResult(res ?? null);
    pushToastMessage({
      title: res?.ok ? 'Dry-run passed' : 'Dry-run failed',
      message: res?.error || res?.rendered_url || res?.provider || '',
    });
  };

  const applyAffiliatePreset = (id: string) => {
    setAffiliatePresetId(id);
    if (!id) return;
    const preset = affiliatePostbackById(id);
    if (!preset) return;
    setProvider('webhook');
    setUrlTemplate(preset.url_template);
  };

  const ui = POSTBACK_PROVIDER_UI[provider];
  const selectedPreset = affiliatePresetId ? affiliatePostbackById(affiliatePresetId) : null;

  return (
    <div className="section-card stack" data-testid="campaign-capi-panel">
      <h3 className="subsection-title">CAPI & Postbacks</h3>
      <p className="text-muted text-sm">
        When a tracked event matches the target type, the postback worker dispatches to the provider
        below. CAPI adapters use click IDs captured on redirect (/click) or zero-redirect /track
        (fbclid, gclid, ttclid). Inbound affiliate S2S (partner → ad-event-processor) is configured on the{' '}
        <Link to={`/campaigns/${campaignId}?tab=tracking`} className="text-sm">
          Integration
        </Link>{' '}
        tab.
      </p>
      {loading ? <p className="text-muted">Loading…</p> : null}
      {error ? <p className="text-danger text-sm">{error}</p> : null}

      <label className="form-field" htmlFor="pb-provider">
        Provider
        <select
          id="pb-provider"
          className="form-input form-input--sm"
          disabled={!canWrite}
          value={provider}
          onChange={(e) => setProvider(normalizePostbackProvider(e.target.value))}
        >
          {postbackProviderIds().map((id) => (
            <option key={id} value={id}>
              {POSTBACK_PROVIDER_UI[id].label}
            </option>
          ))}
        </select>
      </label>
      <p className="text-muted text-sm">{ui.blurb}</p>

      {provider === 'webhook' ? (
        <label
          className="form-field"
          htmlFor="pb-affiliate-preset"
          data-testid="affiliate-postback-preset-field"
        >
          Affiliate network preset
          <select
            id="pb-affiliate-preset"
            className="form-input form-input--sm"
            data-testid="affiliate-postback-preset"
            disabled={!canWrite}
            value={affiliatePresetId}
            onChange={(e) => applyAffiliatePreset(e.target.value)}
          >
            <option value="">— Select network —</option>
            {AFFILIATE_POSTBACK_PRESETS.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
        </label>
      ) : null}

      {selectedPreset ? (
        <p className="text-muted text-sm" data-testid="affiliate-postback-notes">
          {selectedPreset.notes ||
            `Network tokens: click=${selectedPreset.network_click_token}, payout=${selectedPreset.network_payout_token}. ` +
              'Replace REPLACE_HOST with the partner postback host. Outbound macros are ad-event-processor {click_id}/{payout}/{tx_id}.'}
        </p>
      ) : null}

      <label className="form-field" htmlFor="pb-url">
        {ui.primaryLabel}
        <div className="toolbar-row">
          <input
            id="pb-url"
            className="form-input"
            data-testid="postback-url-template"
            disabled={!canWrite}
            placeholder={ui.primaryPlaceholder}
            value={urlTemplate}
            onChange={(e) => setUrlTemplate(e.target.value)}
          />
          <Button
            label="Copy"
            variant="secondary"
            size="sm"
            data-testid="postback-url-copy"
            disabled={!urlTemplate}
            onClick={() => copyText(ui.primaryLabel, urlTemplate)}
          />
        </div>
        <span className="form-hint text-muted text-sm">{ui.primaryHelp}</span>
      </label>

      <label className="form-field" htmlFor="pb-token">
        {ui.tokenLabel}
        <input
          id="pb-token"
          type="password"
          className="form-input"
          disabled={!canWrite}
          placeholder={ui.tokenPlaceholder}
          value={apiToken}
          onChange={(e) => setApiToken(e.target.value)}
        />
        <span className="form-hint text-muted text-sm">{ui.tokenHelp}</span>
      </label>

      <label className="form-field" htmlFor="pb-event">
        Target event
        <select
          id="pb-event"
          className="form-input form-input--sm"
          disabled={!canWrite}
          value={targetEvent}
          onChange={(e) => setTargetEvent(e.target.value)}
        >
          <option value="conversion">Conversion</option>
          <option value="click">Click</option>
          <option value="install">Install</option>
          <option value="lead">Lead</option>
          <option value="purchase">Purchase</option>
        </select>
        <span className="form-hint text-muted text-sm">{ui.eventMappingHint}</span>
      </label>

      {ui.supportsTestEventCode ? (
        <label
          className="form-field"
          htmlFor="pb-test-event-code"
          data-testid="postback-test-event-code-field"
        >
          Test event code (Meta / TikTok staging)
          <input
            id="pb-test-event-code"
            className="form-input form-input--sm"
            data-testid="postback-test-event-code"
            disabled={!canWrite}
            placeholder="TEST12345"
            value={testEventCode}
            onChange={(e) => setTestEventCode(e.target.value)}
          />
          <span className="form-hint text-muted text-sm">
            Routes events to the provider test stream. Use with scripts/test/capi_meta_staging.sh on
            staging.
          </span>
        </label>
      ) : null}

      {canWrite ? (
        <div className="toolbar-row">
          <Button
            label={saving ? 'Saving…' : 'Save postback'}
            variant="primary"
            size="sm"
            loading={saving}
            disabled={saving}
            onClick={() => void save()}
          />
          <Button
            label={dryRunning ? 'Testing…' : 'Dry-run postback'}
            variant="secondary"
            size="sm"
            loading={dryRunning}
            disabled={dryRunning || !urlTemplate}
            data-testid="postback-dry-run"
            onClick={() => void dryRun()}
          />
        </div>
      ) : null}

      {dryRunResult ? (
        <pre className="code-block text-sm" data-testid="postback-dry-run-result">
          {JSON.stringify(dryRunResult, null, 2)}
        </pre>
      ) : null}

      <h4 className="subsection-title mt-4">DLQ</h4>
      {dlq.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state__title">No failed postbacks</div>
          <div className="empty-state__desc text-muted text-sm">
            DLQ entries appear when outbound postbacks fail.
          </div>
        </div>
      ) : (
        <div className="table-wrapper">
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">ID</th>
                <th scope="col">Event</th>
                <th scope="col">Failures</th>
                <th scope="col">Status</th>
                <th scope="col" />
              </tr>
            </thead>
            <tbody>
              {dlq.map((row) => {
                const rowId = row.id;
                const status = typeof row.status === 'string' ? row.status : '';
                return (
                  <tr key={String(rowId)}>
                    <td>{String(rowId ?? '')}</td>
                    <td>{typeof row.event_type === 'string' ? row.event_type : '—'}</td>
                    <td>{String(row.failures_count ?? 0)}</td>
                    <td>{status || '—'}</td>
                    <td>
                      {canWrite &&
                      status !== 'RETRIED' &&
                      (typeof rowId === 'string' || typeof rowId === 'number') ? (
                        <Button
                          label="Retry"
                          variant="secondary"
                          size="sm"
                          data-testid={`postback-dlq-retry-${rowId}`}
                          onClick={() => void retry(rowId)}
                        />
                      ) : null}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
