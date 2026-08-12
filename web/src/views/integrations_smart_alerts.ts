import type { ViewHandle } from '../lib/router_types.js';
import { el, replaceChildren, eventTargetValue } from '../lib/dom.js';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { isCustomerUuid } from '../helpers/customer_context.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderStatusBadge } from '../ui/status_badge.js';
import { tableSkeletonRows, renderEmptyState } from '../ui/data_table.js';
import { renderButton } from '../ui/button.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
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

/**
 * Smart Alerts integration — metric threshold rules with webhook delivery.
 */
export function mount(container: HTMLElement): ViewHandle {
  let destroyed = false;
  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'campaigns:write');
  const sessionScoped = hasBoundCustomer(user?.role);
  const tenantCustomerId = boundCustomerId(user);

  const qsCustomer = new URLSearchParams(window.location.search).get('customer_id') || '';
  let customerId = sessionScoped ? tenantCustomerId : qsCustomer;
  let rules: SmartAlertRule[] = [];
  let history: SmartAlertEvent[] = [];
  let loading = true;
  let error: Error | string | null = null;
  let busy = false;
  let editingId: string | null = null;

  const ruleForm: {
    name: string;
    metric: SmartAlertMetric;
    operator: SmartAlertOperator;
    threshold: number;
    window_minutes: number;
    webhook_url: string;
    campaign_id: string;
    enabled: boolean;
  } = {
    name: '',
    metric: 'clicks',
    operator: 'gt',
    threshold: 100,
    window_minutes: 60,
    webhook_url: '',
    campaign_id: '',
    enabled: true,
  };

  function resetForm() {
    editingId = null;
    ruleForm.name = '';
    ruleForm.metric = 'clicks';
    ruleForm.operator = 'gt';
    ruleForm.threshold = 100;
    ruleForm.window_minutes = 60;
    ruleForm.webhook_url = '';
    ruleForm.campaign_id = '';
    ruleForm.enabled = true;
  }

  function fillForm(rule: SmartAlertRule) {
    editingId = rule.id;
    ruleForm.name = rule.name;
    ruleForm.metric = rule.metric;
    ruleForm.operator = rule.operator;
    ruleForm.threshold = rule.threshold;
    ruleForm.window_minutes = rule.window_minutes;
    ruleForm.webhook_url = rule.webhook_url;
    ruleForm.campaign_id = rule.campaign_id ?? '';
    ruleForm.enabled = rule.enabled;
  }

  async function reload() {
    if (!isCustomerUuid(customerId)) {
      rules = [];
      history = [];
      loading = false;
      render();
      return;
    }
    loading = true;
    error = null;
    render();
    try {
      const [r, h] = await Promise.all([
        fetchSmartAlertRules(customerId),
        fetchSmartAlertHistory(customerId),
      ]);
      if (destroyed) return;
      rules = r;
      history = h;
    } catch (e) {
      if (destroyed) return;
      error = e instanceof Error ? e : String(e);
    } finally {
      if (!destroyed) {
        loading = false;
        render();
      }
    }
  }

  async function saveRule() {
    if (!canWrite || !isCustomerUuid(customerId)) return;
    busy = true;
    render();
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
      busy = false;
      if (!destroyed) render();
    }
  }

  async function removeRule(id: string) {
    if (!canWrite) return;
    busy = true;
    render();
    try {
      await deleteSmartAlertRule(id);
      pushToastMessage({ title: 'Rule deleted', message: id });
      if (editingId === id) resetForm();
      await reload();
    } catch (e) {
      if (e instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Save failed', message: mapServiceError(e).message });
    } finally {
      busy = false;
      if (!destroyed) render();
    }
  }

  async function ackEvent(id: string) {
    if (!canWrite) return;
    busy = true;
    render();
    try {
      await ackSmartAlertEvent(id);
      pushToastMessage({ title: 'Alert acknowledged', message: id });
      await reload();
    } catch (e) {
      if (e instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Save failed', message: mapServiceError(e).message });
    } finally {
      busy = false;
      if (!destroyed) render();
    }
  }

  function renderRuleForm() {
    return el('section', { className: 'stack card', 'data-testid': 'smart-alerts-form' },
      el('h2', { className: 'h3' }, editingId ? 'Edit rule' : 'New rule'),
      el('label', { className: 'stack gap-xs' },
        el('span', {}, 'Name'),
        el('input', {
          type: 'text',
          value: ruleForm.name,
          disabled: !canWrite || busy,
          'data-testid': 'smart-alerts-name',
          oninput: (e: Event) => { ruleForm.name = eventTargetValue(e); },
        }),
      ),
      el('div', { className: 'grid-2' },
        el('label', { className: 'stack gap-xs' },
          el('span', {}, 'Metric'),
          el('select', {
            disabled: !canWrite || busy,
            onchange: (e: Event) => {
              ruleForm.metric = eventTargetValue(e) as typeof ruleForm.metric;
            },
          }, ...SMART_ALERT_METRICS.map((m) =>
            el('option', { value: m.value, selected: ruleForm.metric === m.value }, m.label),
          )),
        ),
        el('label', { className: 'stack gap-xs' },
          el('span', {}, 'Operator'),
          el('select', {
            disabled: !canWrite || busy,
            onchange: (e: Event) => {
              ruleForm.operator = eventTargetValue(e) as typeof ruleForm.operator;
            },
          }, ...SMART_ALERT_OPERATORS.map((o) =>
            el('option', { value: o.value, selected: ruleForm.operator === o.value }, o.label),
          )),
        ),
      ),
      el('div', { className: 'grid-2' },
        el('label', { className: 'stack gap-xs' },
          el('span', {}, 'Threshold'),
          el('input', {
            type: 'number',
            step: 'any',
            value: String(ruleForm.threshold),
            disabled: !canWrite || busy,
            oninput: (e: Event) => { ruleForm.threshold = Number(eventTargetValue(e)); },
          }),
        ),
        el('label', { className: 'stack gap-xs' },
          el('span', {}, 'Window (minutes)'),
          el('input', {
            type: 'number',
            min: '5',
            max: '1440',
            value: String(ruleForm.window_minutes),
            disabled: !canWrite || busy,
            oninput: (e: Event) => { ruleForm.window_minutes = Number(eventTargetValue(e)); },
          }),
        ),
      ),
      el('label', { className: 'stack gap-xs' },
        el('span', {}, 'Webhook URL (Slack / Discord / custom)'),
        el('input', {
          type: 'url',
          value: ruleForm.webhook_url,
          placeholder: 'https://hooks.slack.com/...',
          disabled: !canWrite || busy,
          'data-testid': 'smart-alerts-webhook',
          oninput: (e: Event) => { ruleForm.webhook_url = eventTargetValue(e); },
        }),
      ),
      el('label', { className: 'stack gap-xs' },
        el('span', {}, 'Campaign ID (optional — all campaigns when empty)'),
        el('input', {
          type: 'text',
          value: ruleForm.campaign_id,
          disabled: !canWrite || busy,
          oninput: (e: Event) => { ruleForm.campaign_id = eventTargetValue(e); },
        }),
      ),
      el('label', { className: 'row gap-sm align-center' },
        el('input', {
          type: 'checkbox',
          checked: ruleForm.enabled,
          disabled: !canWrite || busy,
          onchange: (e: Event) => {
            ruleForm.enabled = (e.target as HTMLInputElement).checked;
          },
        }),
        el('span', {}, 'Enabled'),
      ),
      el('div', { className: 'cluster--actions' },
        canWrite
          ? renderButton({
            label: editingId ? 'Update rule' : 'Create rule',
            variant: 'primary',
            loading: busy,
            disabled: busy || !ruleForm.name.trim() || !ruleForm.webhook_url.trim(),
            testId: 'smart-alerts-save',
            onClick: () => { void saveRule(); },
          })
          : null,
        editingId
          ? renderButton({
            label: 'Cancel edit',
            variant: 'ghost',
            disabled: busy,
            onClick: () => { resetForm(); render(); },
          })
          : null,
      ),
    );
  }

  function renderRulesTable() {
    if (loading) {
      return el('section', { className: 'card', 'data-testid': 'smart-alerts-rules' },
        el('h2', { className: 'h3' }, 'Rules'),
        tableSkeletonRows(3),
      );
    }
    return el('section', { className: 'card stack', 'data-testid': 'smart-alerts-rules' },
      el('h2', { className: 'h3' }, 'Rules'),
      rules.length === 0
        ? renderEmptyState({
          title: 'No alert rules yet',
          description: 'Create a rule to receive webhook notifications when metrics cross thresholds.',
          icon: 'bell',
        })
        : el('table', { className: 'data-table' },
          el('thead', {},
            el('tr', {},
              el('th', {}, 'Name'),
              el('th', {}, 'Condition'),
              el('th', {}, 'Window'),
              el('th', {}, 'Status'),
              el('th', {}, ''),
            ),
          ),
          el('tbody', {},
            ...rules.map((rule) =>
              el('tr', { 'data-testid': `smart-alert-rule-${rule.id}` },
                el('td', {}, rule.name),
                el('td', {},
                  `${rule.metric} ${rule.operator} ${rule.threshold}`,
                  rule.campaign_id ? el('div', { className: 'text-muted text-sm' }, rule.campaign_id) : null,
                ),
                el('td', {}, `${rule.window_minutes} min`),
                el('td', {}, renderStatusBadge(rule.enabled ? 'ACTIVE' : 'PAUSED')),
                el('td', { className: 'row gap-xs' },
                  canWrite
                    ? renderButton({
                      label: 'Edit',
                      variant: 'ghost',
                      size: 'sm',
                      disabled: busy,
                      onClick: () => { fillForm(rule); render(); },
                    })
                    : null,
                  canWrite
                    ? renderButton({
                      label: 'Delete',
                      variant: 'ghost',
                      size: 'sm',
                      disabled: busy,
                      onClick: () => { void removeRule(rule.id); },
                    })
                    : null,
                ),
              ),
            ),
          ),
        ),
    );
  }

  function renderHistoryTable() {
    return el('section', { className: 'card stack', 'data-testid': 'smart-alerts-history' },
      el('h2', { className: 'h3' }, 'History'),
      loading
        ? tableSkeletonRows(3)
        : history.length === 0
          ? renderEmptyState({
            title: 'No fired alerts yet',
            description: 'History appears when a rule threshold is crossed.',
            icon: 'bell',
          })
          : el('table', { className: 'data-table' },
            el('thead', {},
              el('tr', {},
                el('th', {}, 'Fired'),
                el('th', {}, 'Metric'),
                el('th', {}, 'Observed'),
                el('th', {}, 'Webhook'),
                el('th', {}, 'Ack'),
                el('th', {}, ''),
              ),
            ),
            el('tbody', {},
              ...history.map((ev) =>
                el('tr', { 'data-testid': `smart-alert-event-${ev.id}` },
                  el('td', {}, new Date(ev.fired_at).toLocaleString()),
                  el('td', {}, `${ev.metric} ${ev.operator} ${ev.threshold}`),
                  el('td', {}, String(ev.observed_value)),
                  el('td', {}, renderStatusBadge(ev.webhook_status.toUpperCase())),
                  el('td', {}, ev.acked_at ? 'Acked' : '—'),
                  el('td', {},
                    canWrite && !ev.acked_at
                      ? renderButton({
                        label: 'Ack',
                        variant: 'ghost',
                        size: 'sm',
                        disabled: busy,
                        testId: 'smart-alerts-ack',
                        onClick: () => { void ackEvent(ev.id); },
                      })
                      : null,
                  ),
                ),
              ),
            ),
          ),
    );
  }

  function render() {
    if (destroyed) return;
    const needsCustomer = !sessionScoped && !isCustomerUuid(customerId);
    replaceChildren(container,
      el('header', { className: 'page-header' },
        el('h1', { className: 'h2' }, 'Smart Alerts'),
        el('p', { className: 'text-muted' },
          'Metric thresholds on ClickHouse data → JSON webhook (Slack, Discord, or custom).',
        ),
      ),
      needsCustomer
        ? el('section', { className: 'card stack' },
          el('label', { className: 'stack gap-xs' },
            el('span', {}, 'Customer ID'),
            el('input', {
              type: 'text',
              value: customerId,
              placeholder: 'UUID',
              'data-testid': 'smart-alerts-customer',
              onchange: (e: Event) => {
                customerId = eventTargetValue(e).trim();
                const url = new URL(window.location.href);
                if (customerId) url.searchParams.set('customer_id', customerId);
                else url.searchParams.delete('customer_id');
                window.history.replaceState({}, '', url);
                void reload();
              },
            }),
          ),
          el('p', { className: 'text-muted' }, 'Select a customer to manage alert rules.'),
        )
        : null,
      error ? renderErrorBlock(error) : null,
      !needsCustomer && isCustomerUuid(customerId) ? renderRuleForm() : null,
      !needsCustomer && isCustomerUuid(customerId) ? renderRulesTable() : null,
      !needsCustomer && isCustomerUuid(customerId) ? renderHistoryTable() : null,
    );
  }

  void reload();

  return {
    destroy() {
      destroyed = true;
    },
  };
}
