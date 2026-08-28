import { Link } from 'react-router-dom';
import type { DisputeRow } from '../../helpers/disputes_api.js';
import { formatAmountMicro } from '../../helpers/money.js';
import { CustomerScopeBar } from '../integrations/customer_scope_bar.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { SettingsSubnav } from './settings_hub.js';
import styles from './settings_shared.module.css';

export type DisputesPanelProps = {
  customerId: string;
  disputes: DisputeRow[];
  total: number;
  limit: number;
  offset: number;
  loading: boolean;
  error: unknown;
  onCustomerApply: (customerId: string) => void;
  onPageChange: (offset: number) => void;
};

export function DisputesPanel({
  customerId,
  disputes,
  total,
  limit,
  offset,
  loading,
  error,
  onCustomerApply,
  onPageChange,
}: DisputesPanelProps) {
  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load disputes" />;
  }

  const hasPrev = offset > 0;
  const hasNext = offset + limit < total;

  return (
    <div className={styles.root} data-testid="settings-disputes-page">
      <PageChrome
        title="Payment disputes"
        badge={
          <Link to="/settings" className={styles.bannerLink}>
            Platform
          </Link>
        }
      />
      <SettingsSubnav />
      <CustomerScopeBar customerId={customerId} onApply={onCustomerApply} label="Customer ID (optional)" />

      <div className={styles.content}>
        {loading && disputes.length === 0 ? (
          <PageSkeleton rows={4} columns={5} />
        ) : disputes.length === 0 ? (
          <EmptyState message="No disputes returned for the current filters." />
        ) : (
          <div className={`${styles.gridTable} ${styles.colsDisputes}`} role="grid">
            <div className={styles.gridHeader} role="row">
              <span className={styles.gridCell} role="columnheader">
                Intent
              </span>
              <span className={styles.gridCell} role="columnheader">
                Customer
              </span>
              <span className={styles.gridCell} role="columnheader">
                Amount
              </span>
              <span className={styles.gridCell} role="columnheader">
                Provider ID
              </span>
              <span className={styles.gridCell} role="columnheader">
                Updated
              </span>
            </div>
            {disputes.map((row) => (
              <div
                key={`${row.intent_id ?? ''}-${row.provider_dispute_id ?? ''}-${row.updated_at ?? ''}`}
                className={styles.gridRow}
                role="row"
              >
                <span className={styles.gridCell} role="gridcell">
                  {row.intent_id ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.customer_id ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {formatAmountMicro(row.amount_micro, row.currency ?? 'USD')}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.provider_dispute_id ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.updated_at ?? '-'}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className={styles.toolbar}>
        <Button type="button" disabled={!hasPrev} onClick={() => onPageChange(Math.max(0, offset - limit))}>
          Previous
        </Button>
        <span className={styles.hint}>
          {total > 0 ? `${offset + 1}-${Math.min(offset + limit, total)} of ${total}` : '0 rows'}
        </span>
        <Button type="button" disabled={!hasNext} onClick={() => onPageChange(offset + limit)}>
          Next
        </Button>
      </div>
    </div>
  );
}
