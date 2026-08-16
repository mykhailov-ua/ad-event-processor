import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { isCustomerUuid } from '../helpers/customer_context.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import {
  SMART_ALERT_METRICS,
  SMART_ALERT_OPERATORS,
  type SmartAlertEvent,
  type SmartAlertRule,
  type SmartAlertMetric,
  type SmartAlertOperator,
  ackSmartAlertEvent,
  createSmartAlertRule,
  deleteSmartAlertRule,
  fetchSmartAlertHistory,
  fetchSmartAlertRules,
  updateSmartAlertRule,
} from '../helpers/smart_alerts_api.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { StatusBadge } from '../components/status_badge.js';

const DEFAULT_RULE_FORM = {
  name: '',
  metric: 'clicks' as SmartAlertMetric,
  operator: 'gt' as SmartAlertOperator,
  threshold: 100,
  window_minutes: 60,
  webhook_url: '',
  campaign_id: '',
  enabled: true,
};

function TableSkeleton({ cols, rows = 3 }: { cols: number; rows?: number }) {
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
 * Smart Alerts integration — metric threshold rules with webhook delivery.
 */
export function IntegrationsSmartAlertsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'campaigns:write');
  const sessionScoped = hasBoundCustomer(user?.role);
  const tenantCustomerId = boundCustomerId(user);

  const qsCustomer = searchParams.get('customer_id') || '';
  const [customerId, setCustomerId] = useState(sessionScoped ? tenantCustomerId : qsCustomer);
  const [rules, setRules] = useState<SmartAlertRule[]>([]);
  const [history, setHistory] = useState<SmartAlertEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [ruleForm, setRuleForm] = useState({ ...DEFAULT_RULE_FORM });

  const resetForm = () => {
    setEditingId(null);
    setRuleForm({ ...DEFAULT_RULE_FORM });
  };

  const fillForm = (rule: SmartAlertRule) => {
    setEditingId(rule.id);
    setRuleForm({
      name: rule.name,
      metric: rule.metric,
      operator: rule.operator,
      threshold: rule.threshold,
      window_minutes: rule.window_minutes,
      webhook_url: rule.webhook_url,
      campaign_id: rule.campaign_id ?? '',
      enabled: rule.enabled,
    });
  };

  const reload = useCallback(async () => {
    if (!isCustomerUuid(customerId)) {
      setRules([]);
      setHistory([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const [r, h] = await Promise.all([
        fetchSmartAlertRules(customerId),
        fetchSmartAlertHistory(customerId),
      ]);
      setRules(r);
      setHistory(h);
    } catch (e) {
      setError(e instanceof Error ? e : String(e));
    } finally {
      setLoading(false);
    }
  }, [customerId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  const saveRule = async () => {
    if (!canWrite || !isCustomerUuid(customerId)) return;
    setBusy(true);
    try {
      const body = {
        customer_id: customerId,
        name: ruleForm.name.trim(),
        metric: ruleForm.metric,
        operator: ruleForm.operator,
        threshold: Number(ruleForm.threshold),
        window_minutes: Number(ruleForm.window_minutes),
        webhook_url: ruleForm.webhook_url.trim(),
        campaign_id: ruleForm.campaign_id.trim() || undefined,
        enabled: ruleForm.enabled,
      };
      if (editingId) {
        await updateSmartAlertRule(editingId, body);
        pushToastMessage({ title: 'Rule updated', message: body.name });
      } else {
        await createSmartAlertRule(body);
        pushToastMessage({ title: 'Rule created', message: body.name });
      }
      resetForm();
      await reload();
    } catch (e) {
      if (e instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Save failed', message: mapServiceError(e).message });
    } finally {
      setBusy(false);
    }
  };

  const removeRule = async (id: string) => {
    if (!canWrite) return;
    setBusy(true);
    try {
      await deleteSmartAlertRule(id);
      pushToastMessage({ title: 'Rule deleted', message: id });
      if (editingId === id) resetForm();
      await reload();
    } catch (e) {
      if (e instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Save failed', message: mapServiceError(e).message });
    } finally {
      setBusy(false);
    }
  };

  const ackEvent = async (id: string) => {
    if (!canWrite) return;
    setBusy(true);
    try {
      await ackSmartAlertEvent(id);
      pushToastMessage({ title: 'Alert acknowledged', message: id });
      await reload();
    } catch (e) {
      if (e instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Save failed', message: mapServiceError(e).message });
    } finally {
      setBusy(false);
    }
  };

  const needsCustomer = !sessionScoped && !isCustomerUuid(customerId);
  const ready = !needsCustomer && isCustomerUuid(customerId);

  return (
    <>
      <header className="page-header">
        <h1 className="h2">Smart Alerts</h1>
        <p className="text-muted">
          Metric thresholds on ClickHouse data → JSON webhook (Slack, Discord, or custom).
        </p>
      </header>

      {needsCustomer ? (
        <section className="card stack">
          <label className="stack gap-xs">
            <span>Customer ID</span>
            <input
              type="text"
              value={customerId}
              placeholder="UUID"
              data-testid="smart-alerts-customer"
              onChange={(e) => {
                const next = e.target.value.trim();
                setCustomerId(next);
                const params = new URLSearchParams(searchParams);
                if (next) params.set('customer_id', next);
                else params.delete('customer_id');
                setSearchParams(params, { replace: true });
              }}
              onBlur={() => void reload()}
            />
          </label>
          <p className="text-muted">Select a customer to manage alert rules.</p>
        </section>
      ) : null}

      {error ? <ErrorBlock error={error} /> : null}

      {ready ? (
        <section className="stack card" data-testid="smart-alerts-form">
          <h2 className="h3">{editingId ? 'Edit rule' : 'New rule'}</h2>
          <label className="stack gap-xs">
            <span>Name</span>
            <input
              type="text"
              value={ruleForm.name}
              disabled={!canWrite || busy}
              data-testid="smart-alerts-name"
              onChange={(e) => setRuleForm((f) => ({ ...f, name: e.target.value }))}
            />
          </label>
          <div className="grid-2">
            <label className="stack gap-xs">
              <span>Metric</span>
              <select
                disabled={!canWrite || busy}
                value={ruleForm.metric}
                onChange={(e) => setRuleForm((f) => ({ ...f, metric: e.target.value as SmartAlertMetric }))}
              >
                {SMART_ALERT_METRICS.map((m) => (
                  <option key={m.value} value={m.value}>{m.label}</option>
                ))}
              </select>
            </label>
            <label className="stack gap-xs">
              <span>Operator</span>
              <select
                disabled={!canWrite || busy}
                value={ruleForm.operator}
                onChange={(e) => setRuleForm((f) => ({ ...f, operator: e.target.value as SmartAlertOperator }))}
              >
                {SMART_ALERT_OPERATORS.map((o) => (
                  <option key={o.value} value={o.value}>{o.label}</option>
                ))}
              </select>
            </label>
          </div>
          <div className="grid-2">
            <label className="stack gap-xs">
              <span>Threshold</span>
              <input
                type="number"
                step="any"
                value={String(ruleForm.threshold)}
                disabled={!canWrite || busy}
                onChange={(e) => setRuleForm((f) => ({ ...f, threshold: Number(e.target.value) }))}
              />
            </label>
            <label className="stack gap-xs">
              <span>Window (minutes)</span>
              <input
                type="number"
                min={5}
                max={1440}
                value={String(ruleForm.window_minutes)}
                disabled={!canWrite || busy}
                onChange={(e) => setRuleForm((f) => ({ ...f, window_minutes: Number(e.target.value) }))}
              />
            </label>
          </div>
          <label className="stack gap-xs">
            <span>Webhook URL (Slack / Discord / custom)</span>
            <input
              type="url"
              value={ruleForm.webhook_url}
              placeholder="https://hooks.slack.com/..."
              disabled={!canWrite || busy}
              data-testid="smart-alerts-webhook"
              onChange={(e) => setRuleForm((f) => ({ ...f, webhook_url: e.target.value }))}
            />
          </label>
          <label className="stack gap-xs">
            <span>Campaign ID (optional — all campaigns when empty)</span>
            <input
              type="text"
              value={ruleForm.campaign_id}
              disabled={!canWrite || busy}
              onChange={(e) => setRuleForm((f) => ({ ...f, campaign_id: e.target.value }))}
            />
          </label>
          <label className="row gap-sm align-center">
            <input
              type="checkbox"
              checked={ruleForm.enabled}
              disabled={!canWrite || busy}
              onChange={(e) => setRuleForm((f) => ({ ...f, enabled: e.target.checked }))}
            />
            <span>Enabled</span>
          </label>
          <div className="cluster--actions">
            {canWrite ? (
              <Button
                label={editingId ? 'Update rule' : 'Create rule'}
                variant="primary"
                loading={busy}
                disabled={busy || !ruleForm.name.trim() || !ruleForm.webhook_url.trim()}
                data-testid="smart-alerts-save"
                onClick={() => void saveRule()}
              />
            ) : null}
            {editingId ? (
              <Button
                label="Cancel edit"
                variant="ghost"
                disabled={busy}
                onClick={resetForm}
              />
            ) : null}
          </div>
        </section>
      ) : null}

      {ready ? (
        <section className="card stack" data-testid="smart-alerts-rules">
          <h2 className="h3">Rules</h2>
          {loading ? <table className="data-table"><tbody><TableSkeleton cols={5} /></tbody></table> : null}
          {!loading && rules.length === 0 ? (
            <div className="empty-state">
              <div className="empty-state__title">No alert rules yet</div>
              <div className="empty-state__desc text-muted text-sm">
                Create a rule to receive webhook notifications when metrics cross thresholds.
              </div>
            </div>
          ) : null}
          {!loading && rules.length > 0 ? (
            <table className="data-table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Condition</th>
                  <th>Window</th>
                  <th>Status</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {rules.map((rule) => (
                  <tr key={rule.id} data-testid={`smart-alert-rule-${rule.id}`}>
                    <td>{rule.name}</td>
                    <td>
                      {`${rule.metric} ${rule.operator} ${rule.threshold}`}
                      {rule.campaign_id ? (
                        <div className="text-muted text-sm">{rule.campaign_id}</div>
                      ) : null}
                    </td>
                    <td>{`${rule.window_minutes} min`}</td>
                    <td><StatusBadge status={rule.enabled ? 'ACTIVE' : 'PAUSED'} /></td>
                    <td className="row gap-xs">
                      {canWrite ? (
                        <Button
                          label="Edit"
                          variant="ghost"
                          size="sm"
                          disabled={busy}
                          onClick={() => fillForm(rule)}
                        />
                      ) : null}
                      {canWrite ? (
                        <Button
                          label="Delete"
                          variant="ghost"
                          size="sm"
                          disabled={busy}
                          onClick={() => void removeRule(rule.id)}
                        />
                      ) : null}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : null}
        </section>
      ) : null}

      {ready ? (
        <section className="card stack" data-testid="smart-alerts-history">
          <h2 className="h3">History</h2>
          {loading ? <table className="data-table"><tbody><TableSkeleton cols={6} /></tbody></table> : null}
          {!loading && history.length === 0 ? (
            <div className="empty-state">
              <div className="empty-state__title">No fired alerts yet</div>
              <div className="empty-state__desc text-muted text-sm">
                History appears when a rule threshold is crossed.
              </div>
            </div>
          ) : null}
          {!loading && history.length > 0 ? (
            <table className="data-table">
              <thead>
                <tr>
                  <th>Fired</th>
                  <th>Metric</th>
                  <th>Observed</th>
                  <th>Webhook</th>
                  <th>Ack</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {history.map((ev) => (
                  <tr key={ev.id} data-testid={`smart-alert-event-${ev.id}`}>
                    <td>{new Date(ev.fired_at).toLocaleString()}</td>
                    <td>{`${ev.metric} ${ev.operator} ${ev.threshold}`}</td>
                    <td>{String(ev.observed_value)}</td>
                    <td><StatusBadge status={ev.webhook_status.toUpperCase()} /></td>
                    <td>{ev.acked_at ? 'Acked' : '—'}</td>
                    <td>
                      {canWrite && !ev.acked_at ? (
                        <Button
                          label="Ack"
                          variant="ghost"
                          size="sm"
                          disabled={busy}
                          data-testid="smart-alerts-ack"
                          onClick={() => void ackEvent(ev.id)}
                        />
                      ) : null}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : null}
        </section>
      ) : null}
    </>
  );
}
