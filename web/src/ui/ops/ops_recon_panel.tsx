import type { ReconRun } from '../../helpers/ops_api.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { PaginationBar } from '../system/pagination_bar.js';
import styles from './ops_shared.module.css';

export type OpsReconPanelProps = {
  items: ReconRun[];
  total: number;
  limit: number;
  offset: number;
  serviceFilter: string;
  loading: boolean;
  error: unknown;
  onServiceFilterChange: (value: string) => void;
  onApplyService: () => void;
  onOffsetChange: (offset: number) => void;
};

export function OpsReconPanel({
  items,
  total,
  limit,
  offset,
  serviceFilter,
  loading,
  error,
  onServiceFilterChange,
  onApplyService,
  onOffsetChange,
}: OpsReconPanelProps) {
  if (error && items.length === 0 && !loading) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load reconciliation runs" />;
  }

  return (
    <div className={styles.root} data-testid="ops-recon-page">
      <PageChrome title="Reconciliation runs" badge={loading ? null : <span>{total} runs</span>} />
      <div className={styles.toolbar}>
        <label className={styles.field}>
          <span className={styles.fieldLabel}>Service</span>
          <input
            className={styles.textInput}
            value={serviceFilter}
            onChange={(event) => onServiceFilterChange(event.target.value)}
            placeholder="optional filter"
          />
        </label>
        <Button type="button" size="sm" onClick={onApplyService}>
          Apply
        </Button>
      </div>
      <div className={styles.content}>
        {loading && items.length === 0 ? (
          <PageSkeleton rows={4} columns={6} />
        ) : items.length === 0 ? (
          <EmptyState message="No reconciliation runs." />
        ) : (
          <div className={`${styles.gridTable} ${styles.reconCols}`} role="grid">
            <div className={styles.gridHeader} role="row">
              <span className={styles.gridCell} role="columnheader">
                Service
              </span>
              <span className={styles.gridCell} role="columnheader">
                Status
              </span>
              <span className={styles.gridCell} role="columnheader">
                Period start
              </span>
              <span className={styles.gridCell} role="columnheader">
                Period end
              </span>
              <span className={styles.gridCell} role="columnheader">
                Delta
              </span>
              <span className={styles.gridCell} role="columnheader">
                Created
              </span>
            </div>
            {items.map((row) => (
              <div key={`${row.service}-${row.id}`} className={styles.gridRow} role="row">
                <span className={styles.gridCell} role="gridcell">
                  {row.service ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.status ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.period_start ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.period_end ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.total_delta ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.created_at ?? '-'}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
      <div className={styles.footer}>
        <PaginationBar limit={limit} offset={offset} total={total} onOffsetChange={onOffsetChange} />
      </div>
    </div>
  );
}
