import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { isCustomerUuid } from '../helpers/customer_context.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import {
  AUTOMATION_ACTION_TYPES,
  AUTOMATION_METRICS,
  AUTOMATION_METRIC_LABELS,
  AUTOMATION_OPERATORS,
  type AutomationAction,
  type AutomationRule,
  createAutomationRule,
  deleteAutomationRule,
  dryRunAutomationRule,
  fetchAutomationRules,
  updateAutomationRule,
} from '../helpers/automation_api.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { StatusBadge } from '../components/status_badge.js';

const DEFAULT_FORM = {
  name: '',
  metric: 'roi_pct' as const,
  operator: 'lt' as const,
  threshold: 0,
  window_minutes: 60,
  group_by: 'placement_id' as const,
  cooldown_minutes: 60,
  campaign_id: '',
  enabled: true,
  action_type: 'pause_campaign' as AutomationAction['type'],
  webhook_url: '',
  network: 'facebook',
};

/** Campaign automation rules admin page. */
export function IntegrationsAutomationPage() {
  const [searchParams] = useSearchParams();
  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'campaigns:write');
  const sessionScoped = hasBoundCustomer(user?.role);
  const tenantCustomerId = boundCustomerId(user);

  const qsCustomer = searchParams.get('customer_id') || '';
  const [customerId, setCustomerId] = useState(sessionScoped ? tenantCustomerId : qsCustomer);
  const [rules, setRules] = useState<AutomationRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState({ ...DEFAULT_FORM });
  const [dryRunResult, setDryRunResult] = useState<unknown[] | null>(null);

  const reload = useCallback(async () => {
    if (!isCustomerUuid(customerId)) {
      setRules([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    const [data, err] = await to(fetchAutomationRules(customerId));
    setLoading(false);
    if (err) {
      setError(err);
      return;
    }
    setRules(data ?? []);
  }, [customerId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  const buildActions = (): AutomationAction[] => {
    if (form.action_type === 'notify') {
      return [{ type: 'notify', webhook_url: form.webhook_url.trim() }];
    }
    if (form.action_type === 'platform_pause') {
      return [{ type: 'platform_pause', network: form.network.trim() }];
    }
    return [{ type: form.action_type }];
  };

  const saveRule = async () => {
    if (!canWrite || !isCustomerUuid(customerId)) return;
    setBusy(true);
    const body = {
      customer_id: customerId,
      campaign_id: form.campaign_id.trim() || undefined,
      name: form.name.trim(),
      metric: form.metric,
      operator: form.operator,
      threshold: form.threshold,
      window_minutes: form.window_minutes,
      group_by: form.group_by,
      cooldown_minutes: form.cooldown_minutes,
      enabled: form.enabled,
      actions: buildActions(),
    };
    const [, err] = await to(
      editingId ? updateAutomationRule(editingId, body) : createAutomationRule(body)
    );
    setBusy(false);
    if (err) {
      pushToastMessage({ title: 'Save failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({ title: 'Rule saved', message: form.name });
    setEditingId(null);
    setForm({ ...DEFAULT_FORM });
    void reload();
  };

  const runDryRun = async (ruleId: string) => {
    setBusy(true);
    const [data, err] = await to(dryRunAutomationRule(ruleId));
    setBusy(false);
    if (err) {
      pushToastMessage({ title: 'Dry run failed', message: mapServiceError(err).message });
      return;
    }
    setDryRunResult(data?.would_fire ?? []);
  };

  if (!isCustomerUuid(customerId)) {
    return (
      <div className="page stack">
        <h1 className="page-title">Automation rules</h1>
        <label className="form-field">
          Customer ID
          <input
            className="form-input font-mono"
            value={customerId}
            onChange={(e) => setCustomerId(e.target.value.trim())}
          />
        </label>
      </div>
    );
  }

  return (
    <div className="page stack">
      <h1 className="page-title">Automation rules</h1>
      <p className="text-muted text-sm">
        Evaluate ClickHouse placement rollups on a schedule and pause campaigns, blacklist
        placements, or notify webhooks when thresholds breach.
      </p>
      {error ? <ErrorBlock error={error} /> : null}

      {canWrite ? (
        <section className="card stack">
          <h2 className="subsection-title">{editingId ? 'Edit rule' : 'New rule'}</h2>
          <div className="form-row">
            <label className="form-field">
              Name
              <input
                className="form-input"
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              />
            </label>
            <label className="form-field">
              Campaign ID (optional)
              <input
                className="form-input font-mono"
                value={form.campaign_id}
                onChange={(e) => setForm((f) => ({ ...f, campaign_id: e.target.value }))}
              />
            </label>
          </div>
          <div className="form-row">
            <label className="form-field">
              Metric
              <select
                className="form-select"
                value={form.metric}
                onChange={(e) =>
                  setForm((f) => ({ ...f, metric: e.target.value as typeof f.metric }))
                }
              >
                {AUTOMATION_METRICS.map((m) => (
                  <option key={m} value={m}>
                    {AUTOMATION_METRIC_LABELS[m]}
                  </option>
                ))}
              </select>
            </label>
            <label className="form-field">
              Operator
              <select
                className="form-select"
                value={form.operator}
                onChange={(e) =>
                  setForm((f) => ({ ...f, operator: e.target.value as typeof f.operator }))
                }
              >
                {AUTOMATION_OPERATORS.map((o) => (
                  <option key={o} value={o}>
                    {o}
                  </option>
                ))}
              </select>
            </label>
            <label className="form-field">
              Threshold
              <input
                className="form-input"
                type="number"
                value={form.threshold}
                onChange={(e) => setForm((f) => ({ ...f, threshold: Number(e.target.value) }))}
              />
            </label>
          </div>
          <div className="form-row">
            <label className="form-field">
              Window (minutes)
              <input
                className="form-input"
                type="number"
                value={form.window_minutes}
                onChange={(e) => setForm((f) => ({ ...f, window_minutes: Number(e.target.value) }))}
              />
            </label>
            <label className="form-field">
              Group by
              <select
                className="form-select"
                value={form.group_by}
                onChange={(e) =>
                  setForm((f) => ({ ...f, group_by: e.target.value as typeof f.group_by }))
                }
              >
                <option value="placement_id">placement_id</option>
                <option value="campaign">campaign</option>
              </select>
            </label>
            <label className="form-field">
              Action
              <select
                className="form-select"
                value={form.action_type}
                onChange={(e) =>
                  setForm((f) => ({
                    ...f,
                    action_type: e.target.value as AutomationAction['type'],
                  }))
                }
              >
                {AUTOMATION_ACTION_TYPES.map((a) => (
                  <option key={a} value={a}>
                    {a}
                  </option>
                ))}
              </select>
            </label>
          </div>
          {form.action_type === 'notify' ? (
            <label className="form-field">
              Webhook URL
              <input
                className="form-input font-mono"
                value={form.webhook_url}
                onChange={(e) => setForm((f) => ({ ...f, webhook_url: e.target.value }))}
              />
            </label>
          ) : null}
          {form.action_type === 'platform_pause' ? (
            <label className="form-field">
              Network
              <input
                className="form-input"
                value={form.network}
                onChange={(e) => setForm((f) => ({ ...f, network: e.target.value }))}
              />
            </label>
          ) : null}
          <Button
            label={busy ? 'Saving...' : 'Save rule'}
            variant="primary"
            size="sm"
            loading={busy}
            disabled={busy}
            onClick={() => void saveRule()}
          />
        </section>
      ) : null}

      <section className="card stack">
        <h2 className="subsection-title">Rules</h2>
        {loading ? <p className="text-muted">Loading...</p> : null}
        {!loading && rules.length === 0 ? (
          <p className="text-muted">No automation rules configured.</p>
        ) : null}
        {rules.map((rule) => (
          <div key={rule.id} className="stack border-b pb-3">
            <div className="flex gap-2 items-center">
              <strong>{rule.name}</strong>
              <StatusBadge status={rule.enabled ? 'ACTIVE' : 'PAUSED'} />
            </div>
            <p className="text-sm text-muted">
              {rule.metric} {rule.operator} {rule.threshold} / {rule.window_minutes}m /{' '}
              {rule.group_by}
            </p>
            {canWrite ? (
              <div className="flex gap-2">
                <Button
                  label="Dry run"
                  variant="secondary"
                  size="sm"
                  onClick={() => void runDryRun(rule.id)}
                />
                <Button
                  label="Delete"
                  variant="secondary"
                  size="sm"
                  onClick={() => void deleteAutomationRule(rule.id).then(() => reload())}
                />
              </div>
            ) : null}
          </div>
        ))}
      </section>

      {dryRunResult ? (
        <section className="card stack">
          <h2 className="subsection-title">Dry run result</h2>
          <pre className="text-sm">{JSON.stringify(dryRunResult, null, 2)}</pre>
        </section>
      ) : null}
    </div>
  );
}
