import { Link } from 'react-router-dom';
import type {
  DashboardSummary,
  DLQList,
  OpsDoctor,
  OutboxList,
} from '../../helpers/ops_api.js';
import { useGridRowAction } from '../../helpers/use_grid_row_action.js';
import { visibleOpsCards } from '../../helpers/ops_catalog.js';
import * as auth from '../../helpers/auth.js';
import { can } from '../../helpers/permissions.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { StubBanner } from '../system/stub_banner.js';
import { TabBar } from '../system/tab_bar.js';
import styles from './ops_shared.module.css';

const OPS_TABS = [
  { id: 'overview', label: 'Overview' },
  { id: 'outbox', label: 'Outbox' },
  { id: 'dlq', label: 'DLQ' },
];

export type OpsHubProps = {
  activeTab: string;
  onTabChange: (tabId: string) => void;
  summary: DashboardSummary | null;
  doctor: OpsDoctor | null;
  outbox: OutboxList | null;
  dlq: DLQList | null;
  summaryLoading: boolean;
  tabLoading: boolean;
  summaryError: unknown;
  tabError: unknown;
  summaryStub: boolean;
  onReloadRoles: () => void;
  rolesBusy: boolean;
  onRetryDlq: (id: string) => void;
  dlqRetryBusyId: string | null;
};

function formatNumber(value: number | undefined): string {
  if (value == null || !Number.isFinite(value)) return '-';
  return String(value);
}

export function OpsHub({
  activeTab,
  onTabChange,
  summary,
  doctor,
  outbox,
  dlq,
  summaryLoading,
  tabLoading,
  summaryError,
  tabError,
  summaryStub,
  onReloadRoles,
  rolesBusy,
  onRetryDlq,
  dlqRetryBusyId,
}: OpsHubProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const onRetryDlqClick = useGridRowAction(onRetryDlq);
  const cards = visibleOpsCards(permissions);
  const canReloadRoles = can(permissions, 'settings:write') || can(permissions, 'shards:write');

  if (summaryError && !summary) {
    return <ErrorBlock error={summaryError} fallbackTitle="Failed to load operations dashboard" />;
  }

  return (
    <div className={styles.root} data-testid="ops-hub-page">
      <PageChrome
        title="Operations"
        badge={
          summary?.generated_at ? (
            <span>Updated {summary.generated_at}</span>
          ) : summaryLoading ? (
            <span>Loading</span>
          ) : null
        }
      />
      <p className={styles.intro}>
        Incident summary, stack doctor checks, and operational queues. KPI tiles use API fields
        only.
      </p>
      {summaryStub ? (
        <StubBanner title="Dashboard unavailable" message="Summary endpoint returned stub or 501." />
      ) : null}
      <div className={styles.toolbar}>
        <TabBar tabs={OPS_TABS} active={activeTab} onChange={onTabChange} />
        {canReloadRoles ? (
          <Button type="button" size="sm" disabled={rolesBusy} onClick={onReloadRoles}>
            Reload RBAC
          </Button>
        ) : null}
      </div>

      {activeTab === 'overview' ? (
        <>
          {summaryLoading && !summary ? (
            <PageSkeleton rows={2} />
          ) : summary ? (
            <div className={styles.kpiRow}>
              <div className={styles.kpiTile}>
                <p className={styles.kpiLabel}>Outbox pending</p>
                <p className={styles.kpiValue}>{formatNumber(summary.outbox_pending)}</p>
              </div>
              <div className={styles.kpiTile}>
                <p className={styles.kpiLabel}>RPS estimate</p>
                <p className={styles.kpiValue}>{formatNumber(summary.rps_estimate)}</p>
              </div>
              <div className={summary.drift_alert ? `${styles.kpiTile} ${styles.kpiAlert}` : styles.kpiTile}>
                <p className={styles.kpiLabel}>Drift micro max</p>
                <p className={styles.kpiValue}>{formatNumber(summary.drift_micro_max)}</p>
              </div>
              <div className={styles.kpiTile}>
                <p className={styles.kpiLabel}>Emergency breaker</p>
                <p className={styles.kpiValue}>{summary.emergency_breaker ?? '-'}</p>
              </div>
            </div>
          ) : null}
          {doctor ? (
            <section className={styles.doctorList} aria-label="Doctor checks">
              <h2 className={styles.cardTitle}>Doctor ({doctor.overall ?? 'unknown'})</h2>
              {(doctor.checks ?? []).map((check) => (
                <div key={check.id ?? check.message} className={styles.doctorRow}>
                  <span className={styles.doctorStatus}>{check.status ?? '-'}</span>
                  <span>
                    {check.message}
                    {check.hint ? ` - ${check.hint}` : ''}
                  </span>
                </div>
              ))}
            </section>
          ) : null}
          {cards.length > 0 ? (
            <div className={styles.cardGrid} role="list">
              {cards.map((card) => (
                <Link
                  key={card.id}
                  to={card.route}
                  className={styles.card}
                  role="listitem"
                  data-testid={`ops-card-${card.id}`}
                >
                  <h2 className={styles.cardTitle}>{card.title}</h2>
                  <p className={styles.cardDesc}>{card.description}</p>
                </Link>
              ))}
            </div>
          ) : (
            <EmptyState message="No operations surfaces available for this session." />
          )}
        </>
      ) : null}

      {activeTab === 'outbox' ? (
        <div className={styles.content}>
          {tabError ? (
            <ErrorBlock error={tabError} fallbackTitle="Failed to load outbox" />
          ) : tabLoading && !outbox ? (
            <PageSkeleton rows={4} />
          ) : (
            <div className={`${styles.gridTable} ${styles.outboxCols}`} role="grid">
              <div className={styles.gridHeader} role="row">
                <span className={styles.gridCell} role="columnheader">
                  ID
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Type
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Status
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Created
                </span>
              </div>
              {(outbox?.items ?? []).length === 0 ? (
                <p className={styles.hint}>No outbox events.</p>
              ) : (
                (outbox?.items ?? []).map((row) => (
                  <div key={row.id} className={styles.gridRow} role="row">
                    <span className={styles.gridCell} role="gridcell">
                      {row.id ?? '-'}
                    </span>
                    <span className={styles.gridCell} role="gridcell">
                      {row.event_type ?? '-'}
                    </span>
                    <span className={styles.gridCell} role="gridcell">
                      {row.status ?? '-'}
                    </span>
                    <span className={styles.gridCell} role="gridcell">
                      {row.created_at ?? '-'}
                    </span>
                  </div>
                ))
              )}
            </div>
          )}
        </div>
      ) : null}

      {activeTab === 'dlq' ? (
        <div className={styles.content}>
          {tabError ? (
            <ErrorBlock error={tabError} fallbackTitle="Failed to load DLQ" />
          ) : tabLoading && !dlq ? (
            <PageSkeleton rows={4} />
          ) : (
            <>
              {dlq?.partial ? (
                <p className={styles.partialBanner}>
                  Partial DLQ snapshot - some shard sources failed.
                </p>
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
                  <p className={styles.hint}>No DLQ entries.</p>
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
                          disabled={dlqRetryBusyId === row.id}
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
              <p className={styles.hint}>
                <Link to="/ops/dlq">Open full DLQ page</Link> for inbox filter and pagination.
              </p>
            </>
          )}
        </div>
      ) : null}
    </div>
  );
}
