import type { DLQInbox, DLQList } from '../../helpers/ops_api.js';
import { useGridRowAction, useGridRowActionPair } from '../../helpers/use_grid_row_action.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import styles from './ops_shared.module.css';

export type OpsDlqPanelProps = {
  dlq: DLQList | null;
  inbox: DLQInbox | null;
  loading: boolean;
  error: unknown;
  sourceFilter: string;
  onSourceFilterChange: (source: string) => void;
  onApplySource: () => void;
  cursor: string;
  nextCursor: string | null;
  onPrevCursor: () => void;
  onNextCursor: () => void;
  onRetryDlq: (id: string) => void;
  onRetryInbox: (id: string, source: string) => void;
  retryBusyId: string | null;
};

export function OpsDlqPanel({
  dlq,
  inbox,
  loading,
  error,
  sourceFilter,
  onSourceFilterChange,
  onApplySource,
  cursor,
  nextCursor,
  onPrevCursor,
  onNextCursor,
  onRetryDlq,
  onRetryInbox,
  retryBusyId,
}: OpsDlqPanelProps) {
  const onRetryDlqClick = useGridRowAction(onRetryDlq);
  const onRetryInboxClick = useGridRowActionPair(onRetryInbox);

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load DLQ" />;
  }

  return (
    <div className={styles.root} data-testid="ops-dlq-page">
      <PageChrome title="Dead letter queue" />
      <div className={styles.toolbar}>
        <label className={styles.field}>
          <span className={styles.fieldLabel}>Inbox source</span>
          <input
            className={styles.textInput}
            value={sourceFilter}
            onChange={(event) => onSourceFilterChange(event.target.value)}
            placeholder="e.g. postback"
          />
        </label>
        <Button type="button" size="sm" onClick={onApplySource}>
          Apply
        </Button>
      </div>
      <div className={styles.content}>
        {loading && !dlq && !inbox ? (
          <PageSkeleton rows={5} columns={6} />
        ) : (
          <>
            <h2 className={styles.cardTitle}>Shard DLQ</h2>
            {dlq?.partial ? (
              <p className={styles.partialBanner}>Partial shard DLQ fan-out.</p>
            ) : null}
            <div className={`${styles.gridTable} ${styles.dlqCols}`} role="grid">
              <div className={styles.gridHeader} role="row">
                <span className={styles.gridCell} role="columnheader">
                  ID
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Shard
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Stream
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Entry
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Error
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Retries
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Action
                </span>
              </div>
              {(dlq?.items ?? []).length === 0 ? (
                <p className={styles.hint}>No shard DLQ entries.</p>
              ) : (
                (dlq?.items ?? []).map((row) => (
                  <div key={row.id} className={styles.gridRow} role="row">
                    <span className={styles.gridCell} role="gridcell">
                      {row.id}
                    </span>
                    <span className={styles.gridCell} role="gridcell">
                      {row.shard_id}
                    </span>
                    <span className={styles.gridCell} role="gridcell">
                      {row.stream_id}
                    </span>
                    <span className={styles.gridCell} role="gridcell">
                      {row.entry_id}
                    </span>
                    <span className={styles.gridCell} role="gridcell">
                      {row.error ?? '-'}
                    </span>
                    <span className={styles.gridCell} role="gridcell">
                      {row.retry_count}
                    </span>
                    <span className={styles.gridCell} role="gridcell">
                      <Button
                        type="button"
                        size="sm"
                        disabled={retryBusyId === row.id}
                        data-row-id={row.id}
                        onClick={onRetryDlqClick}
                      >
                        Retry
                      </Button>
                    </span>
                  </div>
                ))
              )}
            </div>

            <h2 className={styles.cardTitle}>DLQ inbox</h2>
            {inbox?.partial ? (
              <p className={styles.partialBanner}>Partial inbox fan-out.</p>
            ) : null}
            <div className={`${styles.gridTable} ${styles.inboxCols}`} role="grid">
              <div className={styles.gridHeader} role="row">
                <span className={styles.gridCell} role="columnheader">
                  ID
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Source
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Event
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Status
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Error
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Action
                </span>
              </div>
              {(inbox?.items ?? []).length === 0 ? (
                <EmptyState message="No inbox entries for current filter." />
              ) : (
                (inbox?.items ?? []).map((row) => (
                  <div key={`${row.source}-${row.id}`} className={styles.gridRow} role="row">
                    <span className={styles.gridCell} role="gridcell">
                      {row.id ?? '-'}
                    </span>
                    <span className={styles.gridCell} role="gridcell">
                      {row.source ?? '-'}
                    </span>
                    <span className={styles.gridCell} role="gridcell">
                      {row.event_type ?? '-'}
                    </span>
                    <span className={styles.gridCell} role="gridcell">
                      {row.status ?? '-'}
                    </span>
                    <span className={styles.gridCell} role="gridcell">
                      {row.error ?? '-'}
                    </span>
                    <span className={styles.gridCell} role="gridcell">
                      {row.id && row.source ? (
                        <Button
                          type="button"
                          size="sm"
                          disabled={retryBusyId === row.id}
                          data-row-id={row.id}
                          data-row-source={row.source}
                          onClick={onRetryInboxClick}
                        >
                          Retry
                        </Button>
                      ) : null}
                    </span>
                  </div>
                ))
              )}
            </div>
            <div className={styles.cursorFooter}>
              <Button type="button" size="sm" disabled={!cursor} onClick={onPrevCursor}>
                First page
              </Button>
              <Button type="button" size="sm" disabled={!nextCursor} onClick={onNextCursor}>
                Next page
              </Button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
