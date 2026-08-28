import { useState } from 'react';
import { Link } from 'react-router-dom';
import * as auth from '../../helpers/auth.js';
import type { SmartAlertEvent, SmartAlertRule } from '../../helpers/integrations_api.js';
import { can, canReadCampaigns } from '../../helpers/permissions.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { CustomerScopeBar } from '../integrations/customer_scope_bar.js';
import shared from '../integrations/integrations_shared.module.css';
import styles from './smart_alerts.module.css';

export type SmartAlertsPanelProps = {
  customerId: string;
  rules: SmartAlertRule[];
  history: SmartAlertEvent[];
  historyLimit: number;
  loading: boolean;
  error: unknown;
  busy: boolean;
  onCustomerApply: (customerId: string) => void;
  onHistoryLimitChange: (limit: number) => void;
  onCreateRule: (body: {
    customer_id: string;
    name: string;
    metric: string;
    operator: string;
    threshold: number;
    window_minutes: number;
    webhook_url: string;
    enabled: boolean;
    campaign_id?: string;
  }) => void;
  onDeleteRule: (id: string) => void;
  onAckEvent: (id: string) => void;
};

export function SmartAlertsPanel({
  customerId,
  rules,
  history,
  historyLimit,
  loading,
  error,
  busy,
  onCustomerApply,
  onHistoryLimitChange,
  onCreateRule,
  onDeleteRule,
  onAckEvent,
}: SmartAlertsPanelProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const canWrite = can(permissions, 'campaigns:write');
  const canList = canReadCampaigns(permissions);

  const [name, setName] = useState('');
  const [metric, setMetric] = useState('spend_velocity');
  const [operator, setOperator] = useState('gt');
  const [threshold, setThreshold] = useState('100');
  const [windowMinutes, setWindowMinutes] = useState('60');
  const [webhookUrl, setWebhookUrl] = useState('');

  if (!canList) {
    return <ErrorBlock error={new Error('forbidden')} fallbackTitle="Smart alerts access denied" />;
  }

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load smart alerts" />;
  }

  return (
    <div className={shared.panelRoot} data-testid="integrations-smart-alerts-page">
      <PageChrome
        title="Smart alerts"
        badge={
          <Link to="/integrations" className={shared.bannerLink}>
            All integrations
          </Link>
        }
      />
      <CustomerScopeBar customerId={customerId} onApply={onCustomerApply} />

      {!customerId ? (
        <div className={shared.hint}>Enter a customer ID and apply to load rules.</div>
      ) : (
        <>
          <section>
            <h2 className={shared.sectionTitle}>Rules</h2>
            <div className={shared.gridTable} role="grid">
              <div className={`${shared.gridHeader} ${styles.colsRules}`} role="row">
                <span className={shared.gridCell} role="columnheader">
                  Name
                </span>
                <span className={shared.gridCell} role="columnheader">
                  Metric
                </span>
                <span className={shared.gridCell} role="columnheader">
                  Op
                </span>
                <span className={shared.gridCell} role="columnheader">
                  Threshold
                </span>
                <span className={shared.gridCell} role="columnheader">
                  Window (min)
                </span>
                <span className={shared.gridCell} role="columnheader">
                  Enabled
                </span>
                <span className={shared.gridCell} role="columnheader">
                  Action
                </span>
              </div>
              {rules.length === 0 && !loading ? (
                <EmptyState message="No smart alert rules for this customer." />
              ) : (
                rules.map((row) => (
                  <div
                    key={row.id}
                    className={`${shared.gridRow} ${styles.colsRules}`}
                    role="row"
                  >
                    <span className={shared.gridCell} role="gridcell">
                      {row.name}
                    </span>
                    <span className={shared.gridCell} role="gridcell">
                      {row.metric}
                    </span>
                    <span className={shared.gridCell} role="gridcell">
                      {row.operator}
                    </span>
                    <span className={shared.gridCell} role="gridcell">
                      {row.threshold}
                    </span>
                    <span className={shared.gridCell} role="gridcell">
                      {row.window_minutes}
                    </span>
                    <span className={shared.gridCell} role="gridcell">
                      {row.enabled ? 'yes' : 'no'}
                    </span>
                    <span className={shared.gridCell} role="gridcell">
                      {canWrite && row.id ? (
                        <Button
                         
                          variant="secondary"
                          disabled={busy}
                          onClick={() => onDeleteRule(row.id as string)}
                        >
                          Delete
                        </Button>
                      ) : (
                        '-'
                      )}
                    </span>
                  </div>
                ))
              )}
            </div>
            {canWrite ? (
              <div className={shared.formStack}>
                <label className={shared.field}>
                  <span className={shared.fieldLabel}>Name</span>
                  <input
                    className={shared.textInput}
                    value={name}
                    onChange={(event) => setName(event.target.value)}
                  />
                </label>
                <label className={shared.field}>
                  <span className={shared.fieldLabel}>Metric</span>
                  <input
                    className={shared.textInput}
                    value={metric}
                    onChange={(event) => setMetric(event.target.value)}
                  />
                </label>
                <label className={shared.field}>
                  <span className={shared.fieldLabel}>Operator</span>
                  <input
                    className={shared.textInput}
                    value={operator}
                    onChange={(event) => setOperator(event.target.value)}
                  />
                </label>
                <label className={shared.field}>
                  <span className={shared.fieldLabel}>Threshold</span>
                  <input
                    className={shared.textInput}
                    type="number"
                    value={threshold}
                    onChange={(event) => setThreshold(event.target.value)}
                  />
                </label>
                <label className={shared.field}>
                  <span className={shared.fieldLabel}>Window minutes</span>
                  <input
                    className={shared.textInput}
                    type="number"
                    value={windowMinutes}
                    onChange={(event) => setWindowMinutes(event.target.value)}
                  />
                </label>
                <label className={shared.field}>
                  <span className={shared.fieldLabel}>Webhook URL</span>
                  <input
                    className={shared.textInput}
                    value={webhookUrl}
                    onChange={(event) => setWebhookUrl(event.target.value)}
                  />
                </label>
                <Button
                 
                  variant="primary"
                  disabled={busy || !name.trim()}
                  onClick={() =>
                    onCreateRule({
                      customer_id: customerId,
                      name: name.trim(),
                      metric: metric.trim(),
                      operator: operator.trim(),
                      threshold: Number.parseFloat(threshold) || 0,
                      window_minutes: Number.parseInt(windowMinutes, 10) || 60,
                      webhook_url: webhookUrl.trim(),
                      enabled: true,
                    })
                  }
                >
                  Create rule
                </Button>
              </div>
            ) : null}
          </section>

          <section>
            <h2 className={shared.sectionTitle}>History</h2>
            <div className={shared.toolbar}>
              <label className={shared.field}>
                <span className={shared.fieldLabel}>Limit</span>
                <select
                  className={shared.select}
                  value={historyLimit}
                  onChange={(event) => onHistoryLimitChange(Number.parseInt(event.target.value, 10))}
                >
                  <option value={25}>25</option>
                  <option value={50}>50</option>
                  <option value={100}>100</option>
                  <option value={200}>200</option>
                </select>
              </label>
            </div>
            <div className={shared.gridTable} role="grid">
              <div className={`${shared.gridHeader} ${styles.colsHistory}`} role="row">
                <span className={shared.gridCell} role="columnheader">
                  Fired
                </span>
                <span className={shared.gridCell} role="columnheader">
                  Rule
                </span>
                <span className={shared.gridCell} role="columnheader">
                  Metric
                </span>
                <span className={shared.gridCell} role="columnheader">
                  Observed
                </span>
                <span className={shared.gridCell} role="columnheader">
                  Webhook
                </span>
                <span className={shared.gridCell} role="columnheader">
                  Ack
                </span>
              </div>
              {history.length === 0 && !loading ? (
                <EmptyState message="No fired alert events." />
              ) : (
                history.map((row) => (
                  <div
                    key={row.id}
                    className={`${shared.gridRow} ${styles.colsHistory}`}
                    role="row"
                  >
                    <span className={shared.gridCell} role="gridcell">
                      {row.fired_at}
                    </span>
                    <span className={shared.gridCell} role="gridcell">
                      {row.rule_id}
                    </span>
                    <span className={shared.gridCell} role="gridcell">
                      {row.metric}
                    </span>
                    <span className={shared.gridCell} role="gridcell">
                      {row.observed_value}
                    </span>
                    <span className={shared.gridCell} role="gridcell">
                      {row.webhook_status}
                    </span>
                    <span className={shared.gridCell} role="gridcell">
                      {row.acked_at ? (
                        'acked'
                      ) : canWrite && row.id ? (
                        <Button
                         
                          variant="secondary"
                          disabled={busy}
                          onClick={() => onAckEvent(row.id as string)}
                        >
                          Ack
                        </Button>
                      ) : (
                        '-'
                      )}
                    </span>
                  </div>
                ))
              )}
            </div>
          </section>
        </>
      )}
    </div>
  );
}
