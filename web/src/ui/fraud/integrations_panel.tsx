import type { FraudIntegrationDTO } from '../../helpers/fraud_api.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { LoadingCountBadge } from '../system/loading_count_badge.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { CustomerScopeBar } from '../integrations/customer_scope_bar.js';
import { FraudSubNav } from './fraud_sub_nav.js';
import styles from './fraud_shared.module.css';

export type IntegrationsPanelProps = {
  customerId: string;
  integrations: FraudIntegrationDTO[];
  loading: boolean;
  error: unknown;
  onCustomerApply: (customerId: string) => void;
};

function formatBool(value: boolean): string {
  return value ? 'yes' : 'no';
}

export function IntegrationsPanel({
  customerId,
  integrations,
  loading,
  error,
  onCustomerApply,
}: IntegrationsPanelProps) {
  if (error && integrations.length === 0 && !loading && customerId) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load fraud integrations" />;
  }

  return (
    <div className={styles.root} data-testid="fraud-integrations-page">
      <PageChrome
        title="Fraud integrations"
        badge={<LoadingCountBadge loading={loading} label={`${integrations.length} campaigns`} />}
      />
      <FraudSubNav customerId={customerId} />
      <p className={styles.intro}>
        Per-campaign fraud integration health: provider wiring, last success, and DLQ depth.
      </p>
      <CustomerScopeBar customerId={customerId} onApply={onCustomerApply} />
      <div className={styles.content}>
        {!customerId ? (
          <p className={styles.hint}>Set customer scope to list fraud integrations.</p>
        ) : loading && integrations.length === 0 ? (
          <PageSkeleton rows={4} columns={6} />
        ) : integrations.length === 0 ? (
          <EmptyState message="No fraud integrations for this customer." />
        ) : (
          <div className={`${styles.gridTable} ${styles.integrationsCols}`} role="grid">
            <div className={styles.gridHeader} role="row">
              <span className={styles.gridCell} role="columnheader">
                Campaign
              </span>
              <span className={styles.gridCell} role="columnheader">
                Name
              </span>
              <span className={styles.gridCell} role="columnheader">
                Provider
              </span>
              <span className={styles.gridCell} role="columnheader">
                Configured
              </span>
              <span className={styles.gridCell} role="columnheader">
                Health
              </span>
              <span className={styles.gridCell} role="columnheader">
                Last success
              </span>
              <span className={styles.gridCell} role="columnheader">
                DLQ
              </span>
              <span className={styles.gridCell} role="columnheader">
                Last error
              </span>
            </div>
            {integrations.map((row) => (
              <div key={row.campaign_id} className={styles.gridRow} role="row">
                <span className={styles.gridCell} role="gridcell">
                  {row.campaign_id}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.name || '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.provider || '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {formatBool(row.configured)}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.health_status || '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.last_success_at || '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.dlq_count}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.last_error || '-'}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
