import type { ConsentProofList } from '../../helpers/ops_api.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { LoadingCountBadge } from '../system/loading_count_badge.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import styles from './ops_shared.module.css';

export type OpsConsentPanelProps = {
  data: ConsentProofList | null;
  loading: boolean;
  error: unknown;
  userIdFilter: string;
  onUserIdFilterChange: (value: string) => void;
  onApplyFilter: () => void;
  cursor: string;
  nextCursor: string | null;
  onPrevCursor: () => void;
  onNextCursor: () => void;
};

export function OpsConsentPanel({
  data,
  loading,
  error,
  userIdFilter,
  onUserIdFilterChange,
  onApplyFilter,
  cursor,
  nextCursor,
  onPrevCursor,
  onNextCursor,
}: OpsConsentPanelProps) {
  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load consent proofs" />;
  }

  const items = data?.items ?? [];

  return (
    <div className={styles.root} data-testid="ops-consent-page">
      <PageChrome title="Consent proofs" badge={<LoadingCountBadge loading={loading} label={`${items.length} rows`} />} />
      <div className={styles.toolbar}>
        <label className={styles.field}>
          <span className={styles.fieldLabel}>User ID hash</span>
          <input
            className={styles.textInput}
            value={userIdFilter}
            onChange={(event) => onUserIdFilterChange(event.target.value)}
          />
        </label>
        <Button type="button" onClick={onApplyFilter}>
          Apply
        </Button>
      </div>
      <div className={styles.content}>
        {loading && !data ? (
          <PageSkeleton rows={4} columns={5} />
        ) : items.length === 0 ? (
          <EmptyState message="No consent proofs for current filter." />
        ) : (
          <div className={`${styles.gridTable} ${styles.consentCols}`} role="grid">
            <div className={styles.gridHeader} role="row">
              <span className={styles.gridCell} role="columnheader">
                User hash
              </span>
              <span className={styles.gridCell} role="columnheader">
                Source
              </span>
              <span className={styles.gridCell} role="columnheader">
                Purposes
              </span>
              <span className={styles.gridCell} role="columnheader">
                Ad storage
              </span>
              <span className={styles.gridCell} role="columnheader">
                Recorded
              </span>
            </div>
            {items.map((row) => (
              <div key={row.id} className={styles.gridRow} role="row">
                <span className={styles.gridCell} role="gridcell">
                  {row.user_id_hash ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.source ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.purposes ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.ad_storage == null ? '-' : row.ad_storage ? 'yes' : 'no'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.recorded_at ?? '-'}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
      <div className={styles.cursorFooter}>
        <Button type="button" disabled={!cursor} onClick={onPrevCursor}>
          First page
        </Button>
        <Button type="button" disabled={!nextCursor} onClick={onNextCursor}>
          Next page
        </Button>
      </div>
    </div>
  );
}
