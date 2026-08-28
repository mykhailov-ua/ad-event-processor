import type { BlacklistEntry } from '../../helpers/ops_api.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { PaginationBar } from '../system/pagination_bar.js';
import styles from './ops_shared.module.css';

export type OpsBlacklistPanelProps = {
  items: BlacklistEntry[];
  total: number;
  limit: number;
  offset: number;
  loading: boolean;
  error: unknown;
  ip: string;
  reason: string;
  formBusy: boolean;
  onIpChange: (value: string) => void;
  onReasonChange: (value: string) => void;
  onBlock: () => void;
  onUnblock: (ip: string, reason: string) => void;
  onOffsetChange: (offset: number) => void;
};

export function OpsBlacklistPanel({
  items,
  total,
  limit,
  offset,
  loading,
  error,
  ip,
  reason,
  formBusy,
  onIpChange,
  onReasonChange,
  onBlock,
  onUnblock,
  onOffsetChange,
}: OpsBlacklistPanelProps) {
  if (error && items.length === 0 && !loading) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load blacklist" />;
  }

  return (
    <div className={styles.root} data-testid="ops-blacklist-page">
      <PageChrome title="IP blacklist" badge={loading ? null : <span>{total} entries</span>} />
      <form
        className={styles.formStack}
        onSubmit={(event) => {
          event.preventDefault();
          onBlock();
        }}
      >
        <label className={styles.field}>
          <span className={styles.fieldLabel}>IP address</span>
          <input
            className={styles.textInput}
            value={ip}
            onChange={(event) => onIpChange(event.target.value)}
            required
          />
        </label>
        <label className={styles.field}>
          <span className={styles.fieldLabel}>Reason (stored as source)</span>
          <input
            className={styles.textInput}
            value={reason}
            onChange={(event) => onReasonChange(event.target.value)}
            placeholder="manual"
          />
        </label>
        <div className={styles.actions}>
          <Button type="submit" size="sm" variant="danger" disabled={formBusy || !ip.trim()}>
            Block IP
          </Button>
        </div>
      </form>
      <div className={styles.content}>
        {loading && items.length === 0 ? (
          <PageSkeleton rows={4} columns={4} />
        ) : items.length === 0 ? (
          <EmptyState message="No blacklist entries." />
        ) : (
          <div className={`${styles.gridTable} ${styles.blacklistCols}`} role="grid">
            <div className={styles.gridHeader} role="row">
              <span className={styles.gridCell} role="columnheader">
                IP
              </span>
              <span className={styles.gridCell} role="columnheader">
                Reason
              </span>
              <span className={styles.gridCell} role="columnheader">
                Created
              </span>
              <span className={styles.gridCell} role="columnheader">
                Expires
              </span>
              <span className={styles.gridCell} role="columnheader">
                Action
              </span>
            </div>
            {items.map((row) => (
              <div key={`${row.id}-${row.ip}`} className={styles.gridRow} role="row">
                <span className={styles.gridCell} role="gridcell">
                  {row.ip ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.reason ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.created_at ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.expires_at ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.ip ? (
                    <Button
                      type="button"
                      size="sm"
                      variant="danger"
                      disabled={formBusy}
                      onClick={() => onUnblock(row.ip!, row.reason ?? 'manual')}
                    >
                      Unblock
                    </Button>
                  ) : null}
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
