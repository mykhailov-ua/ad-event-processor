import { Link } from 'react-router-dom';
import type {
  PublisherDashboard,
  PublisherStatementListResponse,
} from '../../helpers/publisher_api.js';
import * as auth from '../../helpers/auth.js';
import { can } from '../../helpers/permissions.js';
import { formatAmountMicro } from '../../helpers/money.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { PaginationBar } from '../system/pagination_bar.js';
import { StubBanner } from '../system/stub_banner.js';
import { TabBar } from '../system/tab_bar.js';
import styles from './publisher_shared.module.css';

const TABS = [
  { id: 'dashboard', label: 'Dashboard' },
  { id: 'statements', label: 'Statements' },
  { id: 'supply', label: 'Supply' },
];

export type PublisherHubProps = {
  activeTab: string;
  onTabChange: (tabId: string) => void;
  from: string;
  to: string;
  onRangeChange: (from: string, to: string) => void;
  dashboard: PublisherDashboard | null;
  statements: PublisherStatementListResponse | null;
  dashboardLoading: boolean;
  statementsLoading: boolean;
  dashboardError: unknown;
  statementsError: unknown;
  limit: number;
  offset: number;
  onOffsetChange: (offset: number) => void;
};

function formatPct(value: number | undefined): string {
  if (value == null || !Number.isFinite(value)) return '-';
  return `${(value * 100).toFixed(2)}%`;
}

function formatDateInput(iso: string): string {
  if (!iso) return '';
  return iso.slice(0, 10);
}

export function PublisherHub({
  activeTab,
  onTabChange,
  from,
  to,
  onRangeChange,
  dashboard,
  statements,
  dashboardLoading,
  statementsLoading,
  dashboardError,
  statementsError,
  limit,
  offset,
  onOffsetChange,
}: PublisherHubProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const canSupply = can(permissions, 'settings:read');

  return (
    <div className={styles.root} data-testid="publisher-hub-page">
      <PageChrome title="Publisher portal" badge={dashboard?.seller_id ? <span>{dashboard.seller_id}</span> : null} />
      <p className={styles.intro}>
        Scoped publisher KPIs and payout statements. Supply management links to integrations when
        permitted.
      </p>
      <div className={styles.toolbar}>
        <TabBar tabs={TABS} active={activeTab} onChange={onTabChange} />
      </div>
      {activeTab !== 'supply' ? (
        <form
          className={styles.form}
          onSubmit={(e) => {
            e.preventDefault();
          }}
        >
          <label className={styles.field}>
            <span className={styles.label}>From (date)</span>
            <input
              className={styles.input}
              type="date"
              value={formatDateInput(from)}
              onChange={(e) => {
                const next = e.target.value ? `${e.target.value}T00:00:00.000Z` : from;
                onRangeChange(next, to);
              }}
            />
          </label>
          <label className={styles.field}>
            <span className={styles.label}>To (date)</span>
            <input
              className={styles.input}
              type="date"
              value={formatDateInput(to)}
              onChange={(e) => {
                const next = e.target.value ? `${e.target.value}T23:59:59.000Z` : to;
                onRangeChange(from, next);
              }}
            />
          </label>
        </form>
      ) : null}
      <div className={styles.content}>
        {activeTab === 'dashboard' ? (
          dashboardLoading && !dashboard ? (
            <PageSkeleton rows={5} />
          ) : dashboardError && !dashboard ? (
            <ErrorBlock error={dashboardError} fallbackTitle="Failed to load publisher dashboard" />
          ) : (
            <>
              <div className={styles.kpiRow}>
                <div className={styles.kpiTile}>
                  <p className={styles.kpiLabel}>Impressions</p>
                  <p className={styles.kpiValue}>{dashboard?.kpis?.impressions ?? 0}</p>
                </div>
                <div className={styles.kpiTile}>
                  <p className={styles.kpiLabel}>Fill rate</p>
                  <p className={styles.kpiValue}>{formatPct(dashboard?.kpis?.fill_rate)}</p>
                </div>
                <div className={styles.kpiTile}>
                  <p className={styles.kpiLabel}>eCPM</p>
                  <p className={styles.kpiValue}>{formatAmountMicro(dashboard?.kpis?.ecpm_micro)}</p>
                </div>
              </div>
              <div className={styles.grid} role="grid" aria-label="Placements">
                <div className={styles.gridHead} role="row">
                  <span role="columnheader">Placement</span>
                  <span role="columnheader">Impressions</span>
                  <span role="columnheader">Clicks</span>
                  <span role="columnheader">Revenue</span>
                  <span role="columnheader">eCPM</span>
                </div>
                {(dashboard?.placements ?? []).length === 0 ? (
                  <p className={styles.gridEmpty}>No placement rows in dashboard response.</p>
                ) : (
                  (dashboard?.placements ?? []).map((row) => (
                    <div key={row.placement_id} className={styles.gridRow} role="row">
                      <span role="gridcell">{row.placement_id}</span>
                      <span role="gridcell">{row.impressions ?? 0}</span>
                      <span role="gridcell">{row.clicks ?? 0}</span>
                      <span role="gridcell">{formatAmountMicro(row.revenue_micro)}</span>
                      <span role="gridcell">{formatAmountMicro(row.ecpm_micro)}</span>
                    </div>
                  ))
                )}
              </div>
            </>
          )
        ) : null}
        {activeTab === 'statements' ? (
          statementsLoading && !statements ? (
            <PageSkeleton rows={5} />
          ) : statementsError && !statements ? (
            <ErrorBlock error={statementsError} fallbackTitle="Failed to load publisher statements" />
          ) : (statements?.items ?? []).length === 0 ? (
            <EmptyState message="No publisher payout lines for this range." />
          ) : (
            <>
              <div className={styles.grid} role="grid" aria-label="Publisher statements">
                <div className={styles.gridHead} role="row">
                  <span role="columnheader">Created</span>
                  <span role="columnheader">Amount</span>
                  <span role="columnheader">Campaign</span>
                  <span role="columnheader">Hash</span>
                  <span role="columnheader" />
                </div>
                {(statements?.items ?? []).map((row) => (
                  <div key={row.id ?? row.created_at} className={styles.gridRow} role="row">
                    <span role="gridcell">{row.created_at ?? '-'}</span>
                    <span role="gridcell">{formatAmountMicro(row.amount_micro)}</span>
                    <span role="gridcell">{row.campaign_id ?? '-'}</span>
                    <span role="gridcell">{row.idempotency_hash ?? '-'}</span>
                    <span role="gridcell" />
                  </div>
                ))}
              </div>
              <div className={styles.footer}>
                <PaginationBar
                  limit={limit}
                  offset={offset}
                  total={statements?.total ?? 0}
                  onOffsetChange={onOffsetChange}
                />
              </div>
            </>
          )
        ) : null}
        {activeTab === 'supply' ? (
          canSupply ? (
            <p className={styles.supplyLink}>
              <Link to="/integrations/supply">Open supply file management</Link>
            </p>
          ) : (
            <StubBanner
              title="Supply admin not available"
              message="This session lacks settings:read. Supply files require operator permissions."
            />
          )
        ) : null}
      </div>
    </div>
  );
}
