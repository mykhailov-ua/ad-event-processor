import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { to } from '../../lib/to.js';
import * as auth from '../../helpers/auth.js';
import { can } from '../../helpers/permissions.js';
import { isCustomerUuid } from '../../helpers/customer_context.js';
import { hasBoundCustomer, boundCustomerId } from '../../helpers/buyer_session.js';
import { mapServiceError } from '../../helpers/service_error.js';
import { pushToastMessage } from '../../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../../helpers/confirm_ui.js';
import { formatMicro } from '../../helpers/money.js';
import {
  COST_SYNC_NETWORKS,
  type CostSyncNetwork,
  deleteCostSyncCredential,
  fetchCostSyncCredentials,
  fetchCostSyncHistory,
  runCostSync,
  upsertCostSyncCredential,
} from '../../helpers/cost_sync_api.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { StatusBadge } from '../components/status_badge.js';

type CostSyncCredential = {
  network: string;
  account_id?: string;
  updated_at?: string;
};

type CostSyncHistoryRow = {
  cost_date?: string;
  network?: string;
  status?: string;
  rows_imported?: number;
  total_amount_usd_micro?: number;
  trigger_source?: string;
};

function TableSkeleton({ cols, rows = 4 }: { cols: number; rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }, (_, i) => (
        <tr key={`sk-${i}`} className="data-table__row--skeleton" aria-hidden="true">
          {Array.from({ length: cols }, (__, j) => (
            <td key={`sk-${i}-${j}`}><span className="skeleton-bar" /></td>
          ))}
        </tr>
      ))}
    </>
  );
}

/**
 * Cost Sync integration admin view.
 */
export function IntegrationsCostSyncPage() {
  const [searchParams] = useSearchParams();
  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'campaigns:write');
  const sessionScoped = hasBoundCustomer(user?.role);
  const tenantCustomerId = boundCustomerId(user);

  const qsCustomer = searchParams.get('customer_id') || '';
  const [customerId, setCustomerId] = useState(sessionScoped ? tenantCustomerId : qsCustomer);
  const [credentials, setCredentials] = useState<CostSyncCredential[]>([]);
  const [history, setHistory] = useState<CostSyncHistoryRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);

  const [credForm, setCredForm] = useState({
    network: 'facebook',
    account_id: '',
    access_token: '',
    refresh_token: '',
    api_key: '',
  });
  const [runForm, setRunForm] = useState({
    network: 'facebook',
    from: '',
    to: '',
  });

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
    setCredentials((creds[0] ?? []) as CostSyncCredential[]);
    setHistory(hist[1] ? [] : ((hist[0] ?? []) as CostSyncHistoryRow[]));
  }, [customerId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  const saveCredential = async () => {
    if (!canWrite || !isCustomerUuid(customerId)) return;
    setBusy(true);
    const [, err] = await to(upsertCostSyncCredential(credForm.network, {
      customer_id: customerId,
      account_id: credForm.account_id.trim(),
      access_token: credForm.access_token,
      refresh_token: credForm.refresh_token,
      api_key: credForm.api_key,
    }));
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Credential save failed', message: mapServiceError(err).message });
      return;
    }
    setCredForm((f) => ({ ...f, access_token: '', refresh_token: '', api_key: '' }));
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
    const body: {
      customer_id: string;
      network: string;
      from?: string;
      to?: string;
    } = { customer_id: customerId, network: runForm.network };
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

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Cost Sync unavailable" />;
  }

  return (
    <section className="stack" data-testid="cost-sync-view">
      <div className="page-header">
        <h1 className="page-header__title">Cost Sync</h1>
        <p className="page-header__desc">
          Import network spend for reconciliation. Credentials are encrypted at rest. After sync, open{' '}
          <a href="/reports/true-roi">True ROI</a> for Ad Spend / True Profit / True ROI / True CPA.
        </p>
      </div>

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
          Customer: <span className="font-mono">{customerId || '—'}</span>
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
                  <th>Updated</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {loading ? <TableSkeleton cols={4} /> : null}
                {!loading && credentials.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="data-table__empty">
                      <div className="empty-state">
                        <div className="empty-state__title">No credentials configured</div>
                        <div className="empty-state__desc text-muted text-sm">
                          Add network credentials below to sync spend.
                        </div>
                      </div>
                    </td>
                  </tr>
                ) : null}
                {credentials.map((c) => (
                  <tr key={c.network}>
                    <td>{c.network}</td>
                    <td className="font-mono text-hint">{c.account_id || '—'}</td>
                    <td>{c.updated_at ? new Date(c.updated_at).toLocaleString() : '—'}</td>
                    <td>
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
                ))}
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
                    onChange={(e) => setCredForm((f) => ({ ...f, network: e.target.value }))}
                  >
                    {COST_SYNC_NETWORKS.map((n: CostSyncNetwork) => (
                      <option key={n.id} value={n.id}>{n.label}</option>
                    ))}
                  </select>
                </label>
                <label className="form-field">
                  Account ID
                  <input
                    className="form-input"
                    value={credForm.account_id}
                    onChange={(e) => setCredForm((f) => ({ ...f, account_id: e.target.value }))}
                  />
                </label>
              </div>
              <label className="form-field">
                Access token
                <input
                  className="form-input font-mono"
                  type="password"
                  autoComplete="off"
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
                  value={credForm.refresh_token}
                  onChange={(e) => setCredForm((f) => ({ ...f, refresh_token: e.target.value }))}
                />
              </label>
              <label className="form-field">
                API key (optional)
                <input
                  className="form-input font-mono"
                  type="password"
                  autoComplete="off"
                  value={credForm.api_key}
                  onChange={(e) => setCredForm((f) => ({ ...f, api_key: e.target.value }))}
                />
              </label>
              <Button
                label={busy ? 'Saving…' : 'Save credential'}
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
                {COST_SYNC_NETWORKS.map((n: CostSyncNetwork) => (
                  <option key={n.id} value={n.id}>{n.label}</option>
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
            label={busy ? 'Running…' : 'Run sync'}
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
                  <tr><td colSpan={6}>No runs yet.</td></tr>
                ) : null}
                {history.map((row, i) => (
                  <tr key={`${row.cost_date}-${row.network}-${i}`}>
                    <td>{row.cost_date ?? '—'}</td>
                    <td>{row.network ?? '—'}</td>
                    <td>
                      <StatusBadge
                        status={row.status === 'success' ? 'ACTIVE' : row.status === 'failed' ? 'ARCHIVED' : 'PAUSED'}
                        label={row.status}
                      />
                    </td>
                    <td>{String(row.rows_imported ?? 0)}</td>
                    <td className="font-mono">${formatMicro(row.total_amount_usd_micro ?? 0)}</td>
                    <td>{row.trigger_source ?? '—'}</td>
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
