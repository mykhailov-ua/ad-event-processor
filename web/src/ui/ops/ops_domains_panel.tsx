import type { DomainRotation, DomainTlsAllowed } from '../../helpers/ops_api.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { LoadingCountBadge } from '../system/loading_count_badge.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { StubBanner } from '../system/stub_banner.js';
import styles from './ops_shared.module.css';

export type OpsDomainsPanelProps = {
  rotation: DomainRotation | null;
  tlsAllowed: DomainTlsAllowed | null;
  loading: boolean;
  error: unknown;
  stub: boolean;
};

export function OpsDomainsPanel({
  rotation,
  tlsAllowed,
  loading,
  error,
  stub,
}: OpsDomainsPanelProps) {
  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load domain rotation" />;
  }

  const hosts = rotation?.hosts ?? [];

  return (
    <div className={styles.root} data-testid="ops-domains-page">
      <PageChrome title="Domain rotation" badge={<LoadingCountBadge loading={loading} label={`${hosts.length} hosts`} />} />
      {stub ? (
        <StubBanner title="Domains unavailable" message="Rotation endpoint returned stub or 501." />
      ) : null}
      {tlsAllowed != null ? (
        <p className={styles.hint}>
          TLS allowlist probe: {tlsAllowed.allowed ? 'allowed' : 'denied'}
        </p>
      ) : null}
      <div className={styles.content}>
        {loading && !rotation ? (
          <PageSkeleton rows={4} columns={6} />
        ) : hosts.length === 0 ? (
          <EmptyState message="No rotation hosts returned." />
        ) : (
          <div className={`${styles.gridTable} ${styles.domainCols}`} role="grid">
            <div className={styles.gridHeader} role="row">
              <span className={styles.gridCell} role="columnheader">
                Hostname
              </span>
              <span className={styles.gridCell} role="columnheader">
                Role
              </span>
              <span className={styles.gridCell} role="columnheader">
                Health
              </span>
              <span className={styles.gridCell} role="columnheader">
                SSL
              </span>
              <span className={styles.gridCell} role="columnheader">
                DMR campaigns
              </span>
              <span className={styles.gridCell} role="columnheader">
                Active campaigns
              </span>
            </div>
            {hosts.map((row) => (
              <div key={row.hostname} className={styles.gridRow} role="row">
                <span className={styles.gridCell} role="gridcell">
                  {row.hostname ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.role ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.health_status ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.ssl_status ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.dmr_campaign_count ?? '-'}
                </span>
                <span className={styles.gridCell} role="gridcell">
                  {row.active_campaign_count ?? '-'}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
