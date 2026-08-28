import type {
  MLEvalReport,
  MLManualLabel,
  MLModelStatus,
} from '../../helpers/ops_api.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import styles from './ops_shared.module.css';

export type OpsMlModelPanelProps = {
  status: MLModelStatus | null;
  evalReport: MLEvalReport | null;
  labels: MLManualLabel[];
  loading: boolean;
  error: unknown;
  ipHash: string;
  label: string;
  reason: string;
  formBusy: boolean;
  onIpHashChange: (value: string) => void;
  onLabelChange: (value: string) => void;
  onReasonChange: (value: string) => void;
  onSubmitLabel: () => void;
};

export function OpsMlModelPanel({
  status,
  evalReport,
  labels,
  loading,
  error,
  ipHash,
  label,
  reason,
  formBusy,
  onIpHashChange,
  onLabelChange,
  onReasonChange,
  onSubmitLabel,
}: OpsMlModelPanelProps) {
  if (error && !status) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load ML model status" />;
  }

  return (
    <div className={styles.root} data-testid="ops-ml-model-page">
      <PageChrome title="ML model ops" />
      <div className={styles.content}>
        {loading && !status ? (
          <PageSkeleton rows={4} columns={4} />
        ) : (
          <div className={styles.mlSection}>
            <section>
              <h2 className={styles.cardTitle}>Model status</h2>
              <div className={styles.mlMeta}>
                <span>Active: {status?.active_version?.id ?? '-'}</span>
                <span>Syncing: {status?.syncing_version?.id ?? '-'}</span>
                <span>Redis version: {status?.redis?.version_id ?? '-'}</span>
                <span>
                  Shards consistent:{' '}
                  {status?.redis?.shards_consistent == null
                    ? '-'
                    : status.redis.shards_consistent
                      ? 'yes'
                      : 'no'}
                </span>
                <span>Drift detected: {status?.drift_detected ? 'yes' : 'no'}</span>
              </div>
            </section>

            <section>
              <h2 className={styles.cardTitle}>Eval report</h2>
              <div className={styles.mlMeta}>
                <span>Status: {evalReport?.status ?? '-'}</span>
                <span>Generated: {evalReport?.generated_at ?? '-'}</span>
                <span>
                  Proxy precision: {evalReport?.proxy_metrics?.precision ?? '-'}
                </span>
                <span>
                  Audited precision: {evalReport?.audited_metrics?.precision ?? '-'}
                </span>
              </div>
            </section>

            <section>
              <h2 className={styles.cardTitle}>Add manual label</h2>
              <form
                className={styles.formStack}
                onSubmit={(event) => {
                  event.preventDefault();
                  onSubmitLabel();
                }}
              >
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
                    className={styles.select}
                    value={label}
                    onChange={(event) => onLabelChange(event.target.value)}
                  >
                    <option value="0">0</option>
                    <option value="1">1</option>
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
                <Button type="submit" disabled={formBusy || !ipHash.trim()}>
                  Save label
                </Button>
              </form>
            </section>

            <section>
              <h2 className={styles.cardTitle}>Manual labels</h2>
              {labels.length === 0 ? (
                <EmptyState message="No manual labels." />
              ) : (
                <div className={`${styles.gridTable} ${styles.consentCols}`} role="grid">
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
                        {row.ip_hash ?? '-'}
                      </span>
                      <span className={styles.gridCell} role="gridcell">
                        {row.label ?? '-'}
                      </span>
                      <span className={styles.gridCell} role="gridcell">
                        {row.reason ?? '-'}
                      </span>
                      <span className={styles.gridCell} role="gridcell">
                        {row.source ?? '-'}
                      </span>
                      <span className={styles.gridCell} role="gridcell">
                        {row.created_at ?? '-'}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </section>
          </div>
        )}
      </div>
    </div>
  );
}
