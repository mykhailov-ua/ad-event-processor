import type { FormEvent } from 'react';
import * as auth from '../../helpers/auth.js';
import type { MLManualLabelDTO } from '../../helpers/fraud_api.js';
import { canWriteFraudLabels } from '../../helpers/fraud_api.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { LoadingCountBadge } from '../system/loading_count_badge.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { CustomerScopeBar } from '../integrations/customer_scope_bar.js';
import { FraudSubNav } from './fraud_sub_nav.js';
import styles from './fraud_shared.module.css';

export type LabelsPanelProps = {
  customerId: string;
  limit: number;
  labels: MLManualLabelDTO[];
  loading: boolean;
  error: unknown;
  formBusy: boolean;
  ipHash: string;
  label: string;
  reason: string;
  bulkJson: string;
  onCustomerApply: (customerId: string) => void;
  onLimitChange: (limit: number) => void;
  onIpHashChange: (value: string) => void;
  onLabelChange: (value: string) => void;
  onReasonChange: (value: string) => void;
  onBulkJsonChange: (value: string) => void;
  onCreateLabel: () => void;
  onBulkImport: () => void;
};

export function LabelsPanel({
  customerId,
  limit,
  labels,
  loading,
  error,
  formBusy,
  ipHash,
  label,
  reason,
  bulkJson,
  onCustomerApply,
  onLimitChange,
  onIpHashChange,
  onLabelChange,
  onReasonChange,
  onBulkJsonChange,
  onCreateLabel,
  onBulkImport,
}: LabelsPanelProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const canWrite = canWriteFraudLabels(permissions);

  if (error && labels.length === 0 && !loading && customerId) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load fraud labels" />;
  }

  return (
    <div className={styles.root} data-testid="fraud-labels-page">
      <PageChrome
        title="Fraud labels"
        badge={<LoadingCountBadge loading={loading} label={`${labels.length} rows`} />}
      />
      <FraudSubNav customerId={customerId} />
      <p className={styles.intro}>
        Manual ML training labels per customer. Label 0 = legitimate, 1 = fraud. Source and reason
        are stored with each row.
      </p>
      <CustomerScopeBar customerId={customerId} onApply={onCustomerApply} />
      <div className={styles.actions}>
        <label className={styles.field}>
          <span className={styles.fieldLabel}>List limit</span>
          <input
            className={styles.textInput}
            type="number"
            min={1}
            max={100}
            value={limit}
            onChange={(event) => onLimitChange(Number.parseInt(event.target.value, 10) || 50)}
            aria-label="List limit"
          />
        </label>
      </div>
      {canWrite ? (
        <>
          <form
            className={styles.formStack}
            onSubmit={(event: FormEvent) => {
              event.preventDefault();
              onCreateLabel();
            }}
          >
            <h3 className={styles.sectionTitle}>Create label</h3>
            <label className={styles.field}>
              <span className={styles.fieldLabel}>IP hash (32 hex)</span>
              <input
                className={styles.textInput}
                value={ipHash}
                onChange={(event) => onIpHashChange(event.target.value)}
                required
              />
            </label>
            <label className={styles.field}>
              <span className={styles.fieldLabel}>Label (0 or 1)</span>
              <select
                className={styles.textInput}
                value={label}
                onChange={(event) => onLabelChange(event.target.value)}
              >
                <option value="0">0 - legitimate</option>
                <option value="1">1 - fraud</option>
              </select>
            </label>
            <label className={styles.field}>
              <span className={styles.fieldLabel}>Reason</span>
              <input
                className={styles.textInput}
                value={reason}
                onChange={(event) => onReasonChange(event.target.value)}
              />
            </label>
            <div className={styles.actions}>
              <Button type="submit" disabled={formBusy || !customerId || !ipHash.trim()}>
                Save label
              </Button>
            </div>
          </form>
          <div className={styles.formStack}>
            <h3 className={styles.sectionTitle}>Bulk import (JSON rows)</h3>
            <textarea
              className={styles.textarea}
              value={bulkJson}
              onChange={(event) => onBulkJsonChange(event.target.value)}
              placeholder={'{"rows":[{"ip_hash":"...","label":1,"reason":"manual"}]}'}
              aria-label="Bulk label JSON"
            />
            <div className={styles.actions}>
              <Button
                type="button"
               
                variant="secondary"
                disabled={formBusy || !customerId || !bulkJson.trim()}
                onClick={onBulkImport}
              >
                Import bulk
              </Button>
            </div>
          </div>
        </>
      ) : (
        <p className={styles.helpText}>Label mutations require campaigns:write or shards:write.</p>
      )}
      <div className={styles.content}>
        {!customerId ? (
          <p className={styles.hint}>Set customer scope to list fraud labels.</p>
        ) : loading && labels.length === 0 ? (
          <PageSkeleton rows={5} columns={5} />
        ) : labels.length === 0 ? (
          <EmptyState message="No manual fraud labels for this customer." />
        ) : (
          <div className={`${styles.gridTable} ${styles.labelsCols}`} role="grid">
            <div className={styles.gridHeader} role="row">
              <span className={styles.gridCell} role="columnheader">
                IP hash
              </span>
              <span className={styles.gridCell} role="columnheader">
                Label
              </span>
              <span className={styles.gridCell} role="columnheader">
                Reason
              </span>
              <span className={styles.gridCell} role="columnheader">
                Source
              </span>
              <span className={styles.gridCell} role="columnheader">
                Created
              </span>
            </div>
            {labels.map((row) => (
              <div key={`${row.ip_hash}-${row.created_at}`} className={styles.gridRow} role="row">
                <span className={styles.gridCell} role="gridcell">
                  {row.ip_hash}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.label}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.reason || '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.source || '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.created_at || '-'}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
