import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { isCustomerUuid } from '../helpers/customer_context.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { formatMicro } from '../helpers/money.js';
import {
  type CostSyncCredentialResponse,
  type CostSyncExtraField,
  type CostSyncNetworkSchema,
  type CostSyncSyncInterval,
  deleteCostSyncCredential,
  fetchCostSyncCredentials,
  fetchCostSyncHistory,
  fetchCostSyncNetworks,
  runCostSync,
  type RunCostSyncRequest,
  upsertCostSyncCredential,
} from '../helpers/cost_sync_api.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { StatusBadge } from '../components/status_badge.js';

type CostSyncHistoryRow = {
  cost_date?: string;
  network?: string;
  status?: string;
  rows_imported?: number;
  total_amount_usd_micro?: number;
  trigger_source?: string;
};

type CredentialFormState = {
  network: string;
  account_id: string;
  access_token: string;
  refresh_token: string;
  api_key: string;
  extra_config: Record<string, string>;
  sync_interval_minutes: CostSyncSyncInterval;
  token_mapping: {
    placement_field: string;
    network_object: string;
    attribution_mode: 'token' | 'spread';
  };
};

const EMPTY_CRED_FORM: CredentialFormState = {
  network: 'facebook',
  account_id: '',
  access_token: '',
  refresh_token: '',
  api_key: '',
  extra_config: {},
  sync_interval_minutes: 1440,
  token_mapping: {
    placement_field: 'placement_id',
    network_object: 'ad_id',
    attribution_mode: 'token',
  },
};

/** Skeleton rows while credential or history tables load. */
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

/** Revcontent uses client_credentials; access token is fetched by the worker. */
function usesClientCredentialsAuth(network: string): boolean {
  return network === 'revcontent';
}

/** Build initial extra_config map for a network schema. */
function emptyExtraForSchema(schema: CostSyncNetworkSchema | null): Record<string, string> {
  const out: Record<string, string> = {};
  for (const field of schema?.extra_fields ?? []) {
    out[field.key] = '';
  }
  return out;
}

/** Merge stored credential into the edit form (secrets stay blank). */
function formFromCredential(
  cred: CostSyncCredentialResponse,
  schema: CostSyncNetworkSchema | null
): CredentialFormState {
  const extra = emptyExtraForSchema(schema);
  for (const [key, value] of Object.entries(cred.extra_config ?? {})) {
    extra[key] = value;
  }
  return {
    network: cred.network,
    account_id: cred.account_id ?? '',
    access_token: '',
    refresh_token: '',
    api_key: '',
    extra_config: extra,
    sync_interval_minutes: cred.sync_interval_minutes ?? 1440,
    token_mapping: {
      placement_field: cred.token_mapping?.placement_field ?? 'placement_id',
      network_object: cred.token_mapping?.network_object ?? 'ad_id',
      attribution_mode: cred.token_mapping?.attribution_mode ?? 'token',
    },
  };
}

/** Validate required extra_config fields before PUT. */
function validateExtraConfig(
  schema: CostSyncNetworkSchema | null,
  extra: Record<string, string>,
  extraSet: Record<string, boolean> | undefined
): string | null {
  for (const field of schema?.extra_fields ?? []) {
    if (!field.required) continue;
    const value = (extra[field.key] ?? '').trim();
    if (value !== '') continue;
    if (field.secret && extraSet?.[field.key]) continue;
    return `${field.label} is required`;
  }
  return null;
}

/** Render one network-specific extra_config field. */
function ExtraConfigField({
  field,
  value,
  configured,
  disabled,
  onChange,
}: {
  field: CostSyncExtraField;
  value: string;
  configured: boolean;
  disabled: boolean;
  onChange: (key: string, next: string) => void;
}) {
  const placeholder =
    field.secret && configured ? 'Leave blank to keep existing value' : field.placeholder || '';
  return (
    <label className="form-field" key={field.key}>
      {field.label}
      {field.required ? ' *' : ''}
      <input
        className="form-input font-mono"
        type={field.secret ? 'password' : 'text'}
        autoComplete="off"
        placeholder={placeholder}
        value={value}
        disabled={disabled}
        onChange={(e) => onChange(field.key, e.target.value)}
      />
      {field.secret && configured && !value ? (
        <span className="text-muted text-sm">Configured (hidden)</span>
      ) : null}
      {field.hint ? <span className="text-muted text-sm">{field.hint}</span> : null}
    </label>
  );
}

export function IntegrationsCostSyncPage() {
  const [searchParams] = useSearchParams();
  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'campaigns:write');
  const sessionScoped = hasBoundCustomer(user?.role);
  const tenantCustomerId = boundCustomerId(user);

  const qsCustomer = searchParams.get('customer_id') || '';
  const drillCampaignId = searchParams.get('campaign_id') || '';
  const [customerId, setCustomerId] = useState(sessionScoped ? tenantCustomerId : qsCustomer);
  const [networkSchemas, setNetworkSchemas] = useState<CostSyncNetworkSchema[]>([]);
  const [credentials, setCredentials] = useState<CostSyncCredentialResponse[]>([]);
  const [history, setHistory] = useState<CostSyncHistoryRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);

  const [credForm, setCredForm] = useState<CredentialFormState>(EMPTY_CRED_FORM);
  const [runForm, setRunForm] = useState({
    network: 'facebook',
    from: '',
    to: '',
  });

  const schemaByNetwork = useMemo(() => {
    const map = new Map<string, CostSyncNetworkSchema>();
    for (const schema of networkSchemas) {
      map.set(schema.network, schema);
    }
    return map;
  }, [networkSchemas]);

  const selectedSchema = schemaByNetwork.get(credForm.network) ?? null;
  const selectedCredential = credentials.find((c) => c.network === credForm.network);

  useEffect(() => {
    void (async () => {
      const [schemas, schemaErr] = await to(fetchCostSyncNetworks());
      if (schemaErr) {
        setError(schemaErr);
        return;
      }
      setNetworkSchemas(schemas ?? []);
    })();
  }, []);

  const reload = useCallback(async () => {
    if (!isCustomerUuid(customerId)) {
      setCredentials([]);
      setHistory([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    const [creds, hist] = await Promise.all([
      to(fetchCostSyncCredentials(customerId)),
      to(fetchCostSyncHistory(customerId)),
    ]);
    setLoading(false);
    if (creds[1]) {
      setError(creds[1]);
      return;
    }
    setCredentials((creds[0] ?? []) as CostSyncCredentialResponse[]);
    setHistory(hist[1] ? [] : ((hist[0] ?? []) as CostSyncHistoryRow[]));
  }, [customerId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  const selectNetworkForEdit = (network: string) => {
    const schema = schemaByNetwork.get(network) ?? null;
    const existing = credentials.find((c) => c.network === network);
    if (existing) {
      setCredForm(formFromCredential(existing, schema));
      return;
    }
    setCredForm({
      ...EMPTY_CRED_FORM,
      network,
      extra_config: emptyExtraForSchema(schema),
    });
  };

  const saveCredential = async () => {
    if (!canWrite || !isCustomerUuid(customerId)) return;
    const validationErr = validateExtraConfig(
      selectedSchema,
      credForm.extra_config,
      selectedCredential?.extra_config_set
    );
    if (validationErr) {
      pushToastMessage({ title: 'Validation failed', message: validationErr });
      return;
    }
    setBusy(true);
    const extraPayload: Record<string, string> = {};
    for (const [key, value] of Object.entries(credForm.extra_config)) {
      if (value.trim() !== '') {
        extraPayload[key] = value.trim();
      }
    }
    const [, err] = await to(
      upsertCostSyncCredential(credForm.network, {
        customer_id: customerId,
        account_id: credForm.account_id.trim(),
        access_token: credForm.access_token,
        refresh_token: credForm.refresh_token,
        api_key: credForm.api_key,
        extra_config: extraPayload,
        sync_interval_minutes: credForm.sync_interval_minutes,
        token_mapping: credForm.token_mapping,
      })
    );
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Credential save failed', message: mapServiceError(err).message });
      return;
    }
    setCredForm((f) => ({
      ...f,
      access_token: '',
      refresh_token: '',
      api_key: '',
      extra_config: Object.fromEntries(
        Object.entries(f.extra_config).map(([key, value]) => [
          key,
          selectedSchema?.extra_fields?.find((field) => field.key === key && field.secret)
            ? ''
            : value,
        ])
      ),
    }));
    pushToastMessage({ title: 'Credential saved', message: credForm.network });
    void reload();
  };

  const removeCredential = async (network: string) => {
    if (!canWrite) return;
    const [, err] = await to(deleteCostSyncCredential(network, customerId));
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Delete failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({ title: 'Credential removed', message: network });
    void reload();
  };

  const triggerRun = async () => {
    if (!canWrite || !isCustomerUuid(customerId)) return;
    setBusy(true);
    const body: RunCostSyncRequest = { customer_id: customerId, network: runForm.network };
    if (runForm.from) body.from = runForm.from;
    if (runForm.to) body.to = runForm.to;
    const [, err] = await to(runCostSync(body));
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Sync failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({ title: 'Sync queued', message: 'Cost sync run accepted' });
    setTimeout(() => void reload(), 1500);
  };

  const accountIdLabel =
    selectedSchema?.account_id_label ||
    (usesClientCredentialsAuth(credForm.network) ? 'Client ID' : 'Account ID');

  const networkOptions =
    networkSchemas.length > 0
      ? networkSchemas
      : [{ network: 'facebook', label: 'Facebook', extra_fields: [] }];

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Cost Sync unavailable" />;
  }

  return (
    <section className="stack" data-testid="cost-sync-view">
      <div className="page-header">
        <h1 className="page-header__title">Cost Sync</h1>
        <p className="page-header__desc">
          Import network spend for reconciliation. Credentials are encrypted at rest. After sync,
          open <a href="/reports/true-roi">True ROI</a> for Ad Spend / True Profit / True ROI / True
          CPA.
        </p>
      </div>

      {drillCampaignId ? (
        <p className="text-muted text-sm" data-testid="cost-sync-drill-campaign">
          Discrepancy drill-down for campaign <span className="font-mono">{drillCampaignId}</span> -
          review import history below.
        </p>
      ) : null}

      {!sessionScoped ? (
        <label className="form-field" htmlFor="cost-sync-customer">
          Customer UUID
          <input
            id="cost-sync-customer"
            className="form-input form-input--sm font-mono"
            value={customerId}
            onChange={(e) => setCustomerId(e.target.value.trim())}
            onBlur={() => void reload()}
          />
        </label>
      ) : (
        <p className="text-muted text-sm">
          Customer: <span className="font-mono">{customerId || '-'}</span>
        </p>
      )}

      {!isCustomerUuid(customerId) ? (
        <p className="text-muted">Enter a customer UUID to manage credentials.</p>
      ) : null}

      {isCustomerUuid(customerId) ? (
        <div className="section-card stack">
          <h2 className="subsection-title">Credentials</h2>
          <div className="table-wrapper">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Network</th>
                  <th>Account</th>
                  <th>Extra config</th>
                  <th>Updated</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {loading ? <TableSkeleton cols={5} /> : null}
                {!loading && credentials.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="data-table__empty">
                      <div className="empty-state">
                        <div className="empty-state__title">No credentials configured</div>
                        <div className="empty-state__desc text-muted text-sm">
                          Add network credentials below to sync spend.
                        </div>
                      </div>
                    </td>
                  </tr>
                ) : null}
                {credentials.map((c) => {
                  const extraKeys = [
                    ...Object.keys(c.extra_config ?? {}),
                    ...Object.keys(c.extra_config_set ?? {}),
                  ];
                  const uniqueExtra = [...new Set(extraKeys)];
                  return (
                    <tr key={c.network}>
                      <td>{c.network}</td>
                      <td className="font-mono text-hint">{c.account_id || '-'}</td>
                      <td className="text-hint text-sm">
                        {uniqueExtra.length > 0 ? uniqueExtra.join(', ') : '-'}
                      </td>
                      <td>{c.updated_at ? new Date(c.updated_at).toLocaleString() : '-'}</td>
                      <td>
                        {canWrite ? (
                          <Button
                            label="Edit"
                            variant="secondary"
                            size="sm"
                            onClick={() => selectNetworkForEdit(c.network)}
                          />
                        ) : null}
                        {canWrite ? (
                          <Button
                            label="Remove"
                            variant="secondary"
                            size="sm"
                            onClick={() => void removeCredential(c.network)}
                          />
                        ) : null}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>

          {canWrite ? (
            <div className="stack mt-4">
              <h3 className="subsection-title">Add / update credential</h3>
              <div className="form-row">
                <label className="form-field">
                  Network
                  <select
                    className="form-select"
                    value={credForm.network}
                    onChange={(e) => selectNetworkForEdit(e.target.value)}
                  >
                    {networkOptions.map((n) => (
                      <option key={n.network} value={n.network}>
                        {n.label}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="form-field">
                  {accountIdLabel}
                  <input
                    className="form-input"
                    value={credForm.account_id}
                    onChange={(e) => setCredForm((f) => ({ ...f, account_id: e.target.value }))}
                  />
                </label>
              </div>
              {usesClientCredentialsAuth(credForm.network) ? (
                <>
                  <p className="text-muted text-sm">
                    Stats API credentials from Revcontent Account Settings. Access token is fetched
                    automatically before each sync (~24h TTL).
                  </p>
                  <label className="form-field">
                    Client secret
                    <input
                      className="form-input font-mono"
                      type="password"
                      autoComplete="off"
                      placeholder="Leave blank to keep existing value"
                      value={credForm.api_key}
                      onChange={(e) => setCredForm((f) => ({ ...f, api_key: e.target.value }))}
                    />
                  </label>
                </>
              ) : (
                <>
                  <label className="form-field">
                    Access token
                    <input
                      className="form-input font-mono"
                      type="password"
                      autoComplete="off"
                      placeholder="Leave blank to keep existing value"
                      value={credForm.access_token}
                      onChange={(e) => setCredForm((f) => ({ ...f, access_token: e.target.value }))}
                    />
                  </label>
                  <label className="form-field">
                    Refresh token (optional)
                    <input
                      className="form-input font-mono"
                      type="password"
                      autoComplete="off"
                      placeholder="Leave blank to keep existing value"
                      value={credForm.refresh_token}
                      onChange={(e) =>
                        setCredForm((f) => ({ ...f, refresh_token: e.target.value }))
                      }
                    />
                  </label>
                  <label className="form-field">
                    API key (optional)
                    <input
                      className="form-input font-mono"
                      type="password"
                      autoComplete="off"
                      placeholder="Leave blank to keep existing value"
                      value={credForm.api_key}
                      onChange={(e) => setCredForm((f) => ({ ...f, api_key: e.target.value }))}
                    />
                  </label>
                </>
              )}
              {(selectedSchema?.extra_fields?.length ?? 0) > 0 ? (
                <div className="stack" data-testid="cost-sync-extra-fields">
                  <h4 className="subsection-title">Network settings</h4>
                  {selectedSchema?.extra_fields?.map((field) => (
                    <ExtraConfigField
                      key={field.key}
                      field={field}
                      value={credForm.extra_config[field.key] ?? ''}
                      configured={Boolean(selectedCredential?.extra_config_set?.[field.key])}
                      disabled={busy}
                      onChange={(key, next) =>
                        setCredForm((f) => ({
                          ...f,
                          extra_config: { ...f.extra_config, [key]: next },
                        }))
                      }
                    />
                  ))}
                </div>
              ) : null}
              <div className="form-row">
                <label className="form-field">
                  Sync interval
                  <select
                    className="form-select"
                    value={String(credForm.sync_interval_minutes)}
                    onChange={(e) =>
                      setCredForm((f) => ({
                        ...f,
                        sync_interval_minutes: Number(e.target.value) as CostSyncSyncInterval,
                      }))
                    }
                  >
                    <option value="1440">Daily (24h)</option>
                    <option value="60">Hourly</option>
                    <option value="30">Every 30 minutes</option>
                    <option value="15">Every 15 minutes</option>
                  </select>
                </label>
                <label className="form-field">
                  Attribution mode
                  <select
                    className="form-select"
                    value={credForm.token_mapping.attribution_mode}
                    onChange={(e) =>
                      setCredForm((f) => ({
                        ...f,
                        token_mapping: {
                          ...f.token_mapping,
                          attribution_mode: e.target.value as 'token' | 'spread',
                        },
                      }))
                    }
                  >
                    <option value="token">Token match</option>
                    <option value="spread">Spread across clicks</option>
                  </select>
                </label>
              </div>
              <div className="form-row">
                <label className="form-field">
                  Click field to match
                  <select
                    className="form-select"
                    value={credForm.token_mapping.placement_field}
                    onChange={(e) =>
                      setCredForm((f) => ({
                        ...f,
                        token_mapping: { ...f.token_mapping, placement_field: e.target.value },
                      }))
                    }
                  >
                    <option value="placement_id">placement_id</option>
                    <option value="sub1">sub1</option>
                    <option value="sub2">sub2</option>
                  </select>
                </label>
                <label className="form-field">
                  Network object ID
                  <select
                    className="form-select"
                    value={credForm.token_mapping.network_object}
                    onChange={(e) =>
                      setCredForm((f) => ({
                        ...f,
                        token_mapping: { ...f.token_mapping, network_object: e.target.value },
                      }))
                    }
                  >
                    <option value="ad_id">ad_id</option>
                    <option value="adset_id">adset_id</option>
                    <option value="placement_id">placement_id</option>
                  </select>
                </label>
              </div>
              <Button
                label={busy ? 'Saving...' : 'Save credential'}
                variant="primary"
                size="sm"
                loading={busy}
                disabled={busy}
                onClick={() => void saveCredential()}
              />
            </div>
          ) : null}
        </div>
      ) : null}

      {isCustomerUuid(customerId) && canWrite ? (
        <div className="section-card stack">
          <h2 className="subsection-title">Manual run</h2>
          <div className="form-row">
            <label className="form-field">
              Network
              <select
                className="form-select"
                value={runForm.network}
                onChange={(e) => setRunForm((f) => ({ ...f, network: e.target.value }))}
              >
                {networkOptions.map((n) => (
                  <option key={n.network} value={n.network}>
                    {n.label}
                  </option>
                ))}
              </select>
            </label>
            <label className="form-field">
              From (YYYY-MM-DD)
              <input
                className="form-input"
                placeholder="yesterday default"
                value={runForm.from}
                onChange={(e) => setRunForm((f) => ({ ...f, from: e.target.value }))}
              />
            </label>
            <label className="form-field">
              To (YYYY-MM-DD)
              <input
                className="form-input"
                placeholder="same as from"
                value={runForm.to}
                onChange={(e) => setRunForm((f) => ({ ...f, to: e.target.value }))}
              />
            </label>
          </div>
          <Button
            label={busy ? 'Running...' : 'Run sync'}
            variant="primary"
            size="sm"
            loading={busy}
            disabled={busy}
            onClick={() => void triggerRun()}
          />
        </div>
      ) : null}

      {isCustomerUuid(customerId) ? (
        <div className="section-card stack">
          <h2 className="subsection-title">History</h2>
          <div className="table-wrapper">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Date</th>
                  <th>Network</th>
                  <th>Status</th>
                  <th>Rows</th>
                  <th>Amount</th>
                  <th>Trigger</th>
                </tr>
              </thead>
              <tbody>
                {loading ? <TableSkeleton cols={6} /> : null}
                {!loading && history.length === 0 ? (
                  <tr>
                    <td colSpan={6}>No runs yet.</td>
                  </tr>
                ) : null}
                {history.map((row, i) => (
                  <tr key={`${row.cost_date}-${row.network}-${i}`}>
                    <td>{row.cost_date ?? '-'}</td>
                    <td>{row.network ?? '-'}</td>
                    <td>
                      <StatusBadge
                        status={
                          row.status === 'success'
                            ? 'ACTIVE'
                            : row.status === 'failed'
                              ? 'ARCHIVED'
                              : 'PAUSED'
                        }
                        label={row.status}
                      />
                    </td>
                    <td>{String(row.rows_imported ?? 0)}</td>
                    <td className="font-mono">${formatMicro(row.total_amount_usd_micro ?? 0)}</td>
                    <td>{row.trigger_source ?? '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : null}
    </section>
  );
}
