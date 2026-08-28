import { useState } from 'react';
import { Link } from 'react-router-dom';
import * as auth from '../../helpers/auth.js';
import type {
  AutomationDryRunResponse,
  AutomationPreset,
  AutomationRule,
} from '../../helpers/integrations_api.js';
import { can, canReadCampaigns } from '../../helpers/permissions.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { CustomerScopeBar } from '../integrations/customer_scope_bar.js';
import shared from '../integrations/integrations_shared.module.css';
import styles from './automation.module.css';

export type AutomationPanelProps = {
  customerId: string;
  presets: AutomationPreset[];
  rules: AutomationRule[];
  dryRun: AutomationDryRunResponse | null;
  loading: boolean;
  error: unknown;
  busy: boolean;
  dryRunRuleId: string | null;
  onCustomerApply: (customerId: string) => void;
  onCreateRule: (body: {
    customer_id: string;
    name: string;
    metric: string;
    operator: string;
    threshold: number;
    window_minutes: number;
    enabled: boolean;
    preset_key?: string;
  }) => void;
  onDeleteRule: (id: string) => void;
  onDryRun: (id: string) => void;
};

export function AutomationPanel({
  customerId,
  presets,
  rules,
  dryRun,
  loading,
  error,
  busy,
  dryRunRuleId,
  onCustomerApply,
  onCreateRule,
  onDeleteRule,
  onDryRun,
}: AutomationPanelProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const canWrite = can(permissions, 'campaigns:write');
  const canList = canReadCampaigns(permissions);

  const [name, setName] = useState('');
  const [metric, setMetric] = useState('ivt_rate');
  const [operator, setOperator] = useState('gt');
  const [threshold, setThreshold] = useState('0.1');
  const [windowMinutes, setWindowMinutes] = useState('60');
  const [presetKey, setPresetKey] = useState('');

  if (!canList) {
    return <ErrorBlock error={new Error('forbidden')} fallbackTitle="Automation access denied" />;
  }

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load automation" />;
  }

  return (
    <div className={shared.panelRoot} data-testid="integrations-automation-page">
      <PageChrome
        title="Automation"
        badge={
          <Link to="/integrations" className={shared.bannerLink}>
            All integrations
          </Link>
        }
      />
      <CustomerScopeBar customerId={customerId} onApply={onCustomerApply} />

      <section>
        <h2 className={shared.sectionTitle}>Presets</h2>
        <div className={shared.gridTable} role="grid">
          <div className={`${shared.gridHeader} ${styles.colsPresets}`} role="row">
            <span className={shared.gridCell} role="columnheader">
              Key
            </span>
            <span className={shared.gridCell} role="columnheader">
              Description
            </span>
            <span className={shared.gridCell} role="columnheader">
              Title
            </span>
          </div>
          {presets.length === 0 && !loading ? (
            <EmptyState message="No automation presets returned." />
          ) : (
            presets.map((row) => (
              <div
                key={row.key}
                className={`${shared.gridRow} ${styles.colsPresets}`}
                role="row"
              >
                <span className={shared.gridCell} role="gridcell">
                  {row.key}
                </span>
                <span className={shared.gridCell} role="gridcell">
                  {row.description ?? row.title}
                </span>
                <span className={shared.gridCell} role="gridcell">
                  {row.title}
                </span>
              </div>
            ))
          )}
        </div>
      </section>

      {!customerId ? (
        <div className={shared.hint}>Enter a customer ID and apply to load rules.</div>
      ) : (
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
                Window
              </span>
              <span className={shared.gridCell} role="columnheader">
                Enabled
              </span>
              <span className={shared.gridCell} role="columnheader">
                Actions
              </span>
            </div>
            {rules.length === 0 && !loading ? (
              <EmptyState message="No automation rules for this customer." />
            ) : (
              rules.map((row) => (
                <div key={row.id} className={`${shared.gridRow} ${styles.colsRules}`} role="row">
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
                    <div className={shared.actions}>
                      {row.id ? (
                        <Button
                         
                          variant="secondary"
                          disabled={busy}
                          onClick={() => onDryRun(row.id as string)}
                        >
                          {dryRunRuleId === row.id ? 'Running...' : 'Dry-run'}
                        </Button>
                      ) : null}
                      {canWrite && row.id ? (
                        <Button
                         
                          variant="secondary"
                          disabled={busy}
                          onClick={() => onDeleteRule(row.id as string)}
                        >
                          Delete
                        </Button>
                      ) : null}
                    </div>
                  </span>
                </div>
              ))
            )}
          </div>
          {canWrite ? (
            <div className={shared.formStack}>
              <label className={shared.field}>
                <span className={shared.fieldLabel}>Preset (optional)</span>
                <select
                  className={shared.select}
                  value={presetKey}
                  onChange={(event) => setPresetKey(event.target.value)}
                >
                  <option value="">Custom rule</option>
                  {presets.map((preset) => (
                    <option key={preset.key} value={preset.key}>
                      {preset.title ?? preset.key}
                    </option>
                  ))}
                </select>
              </label>
              <label className={shared.field}>
                <span className={shared.fieldLabel}>Name</span>
                <input
                  className={shared.textInput}
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                />
              </label>
              {!presetKey ? (
                <>
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
                </>
              ) : null}
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
                    enabled: true,
                    preset_key: presetKey || undefined,
                  })
                }
              >
                Create rule
              </Button>
            </div>
          ) : null}
          {dryRun ? (
            <pre className={shared.codePreview} data-testid="automation-dry-run-result">
              {JSON.stringify(dryRun, null, 2)}
            </pre>
          ) : null}
        </section>
      )}
    </div>
  );
}
