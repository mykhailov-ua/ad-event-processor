import type { ShardHealthStatus, ShardStatus } from '../../helpers/ops_api.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { LoadingCountBadge } from '../system/loading_count_badge.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import styles from './ops_shared.module.css';

export type OpsShardsPanelProps = {
  data: ShardStatus | null;
  loading: boolean;
  error: unknown;
  catchupBusy: boolean;
  onCatchup: () => void;
};

export function OpsShardsPanel({
  data,
  loading,
  error,
  catchupBusy,
  onCatchup,
}: OpsShardsPanelProps) {
  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load shard health" />;
  }

  const shards = data?.shards ?? [];

  return (
    <div className={styles.root} data-testid="ops-shards-page">
      <PageChrome title="Shard health" badge={<LoadingCountBadge loading={loading} label={`${shards.length} shards`} />} />
      <div className={styles.toolbar}>
        <Button type="button" disabled={catchupBusy} onClick={onCatchup}>
          Run shard 0 catch-up
        </Button>
      </div>
      {data?.partial ? (
        <p className={styles.partialBanner}>Partial shard fan-out - some sources failed.</p>
      ) : null}
      <div className={styles.content}>
        {loading && !data ? (
          <PageSkeleton rows={4} columns={6} />
        ) : shards.length === 0 ? (
          <EmptyState message="No shard health rows returned." />
        ) : (
          <div className={`${styles.gridTable} ${styles.shardCols}`} role="grid">
            <div className={styles.gridHeader} role="row">
              <span className={styles.gridCell} role="columnheader">
                Shard
              </span>
              <span className={styles.gridCell} role="columnheader">
                Ping
              </span>
              <span className={styles.gridCell} role="columnheader">
                Latency (ms)
              </span>
              <span className={styles.gridCell} role="columnheader">
                Config version
              </span>
              <span className={styles.gridCell} role="columnheader">
                Config lag
              </span>
              <span className={styles.gridCell} role="columnheader">
                Error
              </span>
            </div>
            {shards.map((row: ShardHealthStatus) => (
              <div key={row.shard_id} className={styles.gridRow} role="row">
                <span className={styles.gridCell} role="gridcell">
                  {row.shard_id ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.ping_ok == null ? '-' : row.ping_ok ? 'ok' : 'down'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.ping_latency_ms ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.config_version ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.config_version_lag ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.ping_error ?? '-'}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
