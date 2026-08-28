import { memo, useCallback, useMemo, useState, type ChangeEvent, type MouseEvent } from 'react';
import { Link } from 'react-router-dom';
import type { BuyerPortfolioResponse } from '../../helpers/selfserve_api.js';
import {
  pauseSelfServeCampaign,
  resumeSelfServeCampaign,
} from '../../helpers/selfserve_api.js';
import { ConfirmCancelledError } from '../../helpers/confirmed_api.js';
import { canReadCampaigns } from '../../helpers/permissions.js';
import { formatAmountMicro } from '../../helpers/money.js';
import {
  useGridRowCheckboxChange,
  useGridRowDatasetAction,
} from '../../helpers/use_grid_row_action.js';
import { pushToastMessage } from '../../helpers/toast_ui.js';
import * as auth from '../../helpers/auth.js';
import { Button } from '../system/button.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import styles from './selfserve_shared.module.css';

export type PortfolioPanelProps = {
  data: BuyerPortfolioResponse | null;
  loading: boolean;
  error: unknown;
  canMutate: boolean;
  onReload: () => void;
};

type PortfolioCampaign = NonNullable<BuyerPortfolioResponse['campaigns']>[number];

function formatPct(value: number | undefined): string {
  if (value == null || !Number.isFinite(value)) return '-';
  return `${value.toFixed(1)}%`;
}

function buildRowView(campaigns: PortfolioCampaign[]) {
  const len = campaigns.length;
  const ids = new Array<string>(len);
  const names = new Array<string>(len);
  const statuses = new Array<string>(len);
  const spends = new Array<string>(len);
  const budgets = new Array<string>(len);
  const utilizations = new Array<string>(len);
  const riskFlags = new Array<boolean>(len);
  const showPause = new Array<boolean>(len);
  const showResume = new Array<boolean>(len);
  for (let i = 0; i < len; i += 1) {
    const row = campaigns[i];
    ids[i] = row.id;
    names[i] = row.name;
    statuses[i] = row.status;
    spends[i] = formatAmountMicro(row.spend_micro);
    budgets[i] = formatAmountMicro(row.budget_micro);
    utilizations[i] = formatPct(row.utilization_pct);
    riskFlags[i] = Boolean(row.overspend_risk);
    showPause[i] = row.status === 'active';
    showResume[i] = row.status === 'paused';
  }
  return { ids, names, statuses, spends, budgets, utilizations, riskFlags, showPause, showResume, len };
}

type PortfolioRowProps = {
  id: string;
  name: string;
  status: string;
  spend: string;
  budget: string;
  utilization: string;
  risk: boolean;
  showPause: boolean;
  showResume: boolean;
  checked: boolean;
  canMutate: boolean;
  showDetailLinks: boolean;
  busy: boolean;
  onRowCheck: (event: ChangeEvent<HTMLInputElement>) => void;
  onRowAction: (event: MouseEvent<HTMLElement>) => void;
};

const PortfolioRow = memo(function PortfolioRow({
  id,
  name,
  status,
  spend,
  budget,
  utilization,
  risk,
  showPause,
  showResume,
  checked,
  canMutate,
  showDetailLinks,
  busy,
  onRowCheck,
  onRowAction,
}: PortfolioRowProps) {
  return (
    <div className={styles.gridRow} role="row">
      {canMutate ? (
        <span role="gridcell">
          <input
            type="checkbox"
            data-row-id={id}
            checked={checked}
            onChange={onRowCheck}
            aria-label={`Select ${name}`}
          />
        </span>
      ) : (
        <span role="gridcell" />
      )}
      <span role="gridcell">
        {showDetailLinks ? <Link to={`/campaigns/${id}`}>{name}</Link> : name}
      </span>
      <span role="gridcell" className={risk ? styles.risk : undefined}>
        {status}
      </span>
      <span role="gridcell">{spend}</span>
      <span role="gridcell">{budget}</span>
      <span role="gridcell">{utilization}</span>
      <span role="gridcell" className={styles.toolbar}>
        {canMutate && showPause ? (
          <Button
            type="button"
           
            disabled={busy}
            data-row-id={id}
            data-row-action="pause"
            onClick={onRowAction}
          >
            Pause
          </Button>
        ) : null}
        {canMutate && showResume ? (
          <Button
            type="button"
           
            disabled={busy}
            data-row-id={id}
            data-row-action="resume"
            onClick={onRowAction}
          >
            Resume
          </Button>
        ) : null}
      </span>
    </div>
  );
});

export function PortfolioPanel({
  data,
  loading,
  error,
  canMutate,
  onReload,
}: PortfolioPanelProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const showDetailLinks = canReadCampaigns(permissions);
  const campaigns = data?.campaigns ?? [];
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set());
  const [busyId, setBusyId] = useState<string | null>(null);
  const [bulkBusy, setBulkBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const rowView = useMemo(() => buildRowView(campaigns), [campaigns]);

  const runPauseResume = useCallback(
    async (ids: string[], action: 'pause' | 'resume') => {
      setActionError(null);
      const fn = action === 'pause' ? pauseSelfServeCampaign : resumeSelfServeCampaign;
      for (const id of ids) {
        setBusyId(id);
        try {
          await fn(id);
        } catch (err) {
          if (err instanceof ConfirmCancelledError) return;
          setActionError(err instanceof Error ? err.message : 'Action failed');
          return;
        } finally {
          setBusyId(null);
        }
      }
      pushToastMessage({
        title: action === 'pause' ? 'Campaigns paused' : 'Campaigns resumed',
        message: `${ids.length} campaign(s) updated`,
      });
      setSelectedIds(new Set());
      onReload();
    },
    [onReload]
  );

  const onBulk = useCallback(
    (action: 'pause' | 'resume') => {
      const ids = [...selectedIds];
      if (ids.length === 0) return;
      setBulkBusy(true);
      void runPauseResume(ids, action).finally(() => setBulkBusy(false));
    },
    [runPauseResume, selectedIds]
  );

  const onRowActionHandler = useCallback(
    (id: string, action: string) => {
      if (action !== 'pause' && action !== 'resume') return;
      setBulkBusy(true);
      void runPauseResume([id], action).finally(() => setBulkBusy(false));
    },
    [runPauseResume]
  );
  const onRowAction = useGridRowDatasetAction(onRowActionHandler);

  const toggleRow = useCallback((id: string, checked: boolean) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (checked) next.add(id);
      else next.delete(id);
      return next;
    });
  }, []);

  const onRowCheck = useGridRowCheckboxChange(toggleRow);

  const toggleAll = useCallback(
    (checked: boolean) => {
      if (!checked) {
        setSelectedIds(new Set());
        return;
      }
      setSelectedIds(new Set(campaigns.map((row) => row.id).filter(Boolean)));
    },
    [campaigns]
  );

  const onClearSelection = useCallback(() => {
    setSelectedIds(new Set());
  }, []);

  const onBulkPause = useCallback(() => onBulk('pause'), [onBulk]);
  const onBulkResume = useCallback(() => onBulk('resume'), [onBulk]);

  if (loading && !data) {
    return <PageSkeleton rows={6} />;
  }

  if (error && !data) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load buyer portfolio" />;
  }

  const freshness = data?.kpis?.freshness;
  const badge =
    freshness?.as_of != null ? (
      <span>{freshness.stale ? 'Stale' : 'Fresh'}</span>
    ) : loading ? (
      <span>Loading</span>
    ) : null;

  return (
    <div data-testid="selfserve-portfolio-panel">
      <PageChrome title="Self-serve portfolio" badge={badge} />
      <p className={styles.intro}>
        Campaign rows come from GET /api/v1/dashboards/buyer. GET /api/v1/selfserve/campaigns is not
        shipped.
      </p>
      <div className={styles.kpiRow}>
        <div className={styles.kpiTile}>
          <p className={styles.kpiLabel}>Active</p>
          <p className={styles.kpiValue}>{data?.active ?? 0}</p>
        </div>
        <div className={styles.kpiTile}>
          <p className={styles.kpiLabel}>Paused</p>
          <p className={styles.kpiValue}>{data?.paused ?? 0}</p>
        </div>
        <div className={styles.kpiTile}>
          <p className={styles.kpiLabel}>Spend</p>
          <p className={styles.kpiValue}>{formatAmountMicro(data?.kpis?.spend_micro)}</p>
        </div>
        <div className={styles.kpiTile}>
          <p className={styles.kpiLabel}>ROI</p>
          <p className={styles.kpiValue}>{formatPct(data?.kpis?.roi_pct)}</p>
        </div>
      </div>
      {canMutate && selectedIds.size > 0 ? (
        <div className={styles.toolbar}>
          <span>{selectedIds.size} selected</span>
          <Button type="button" disabled={bulkBusy} onClick={onBulkPause}>
            Pause
          </Button>
          <Button type="button" disabled={bulkBusy} onClick={onBulkResume}>
            Resume
          </Button>
          <Button type="button" disabled={bulkBusy} onClick={onClearSelection}>
            Clear
          </Button>
        </div>
      ) : null}
      {actionError ? <ErrorBlock error={new Error(actionError)} fallbackTitle="Bulk action failed" /> : null}
      <div className={styles.grid} role="grid" aria-label="Campaign portfolio">
        <div className={styles.gridHead} role="row">
          {canMutate ? (
            <span role="columnheader">
              <input
                type="checkbox"
                aria-label="Select all campaigns"
                checked={campaigns.length > 0 && selectedIds.size === campaigns.length}
                onChange={(event) => toggleAll(event.target.checked)}
              />
            </span>
          ) : (
            <span role="columnheader" />
          )}
          <span role="columnheader">Name</span>
          <span role="columnheader">Status</span>
          <span role="columnheader">Spend</span>
          <span role="columnheader">Budget</span>
          <span role="columnheader">Utilization</span>
          <span role="columnheader">Actions</span>
        </div>
        {campaigns.length === 0 ? (
          <p className={styles.gridEmpty}>No campaigns in the buyer dashboard response.</p>
        ) : (
          Array.from({ length: rowView.len }, (_, index) => (
            <PortfolioRow
              key={rowView.ids[index]}
              id={rowView.ids[index]}
              name={rowView.names[index]}
              status={rowView.statuses[index]}
              spend={rowView.spends[index]}
              budget={rowView.budgets[index]}
              utilization={rowView.utilizations[index]}
              risk={rowView.riskFlags[index]}
              showPause={rowView.showPause[index]}
              showResume={rowView.showResume[index]}
              checked={selectedIds.has(rowView.ids[index])}
              canMutate={canMutate}
              showDetailLinks={showDetailLinks}
              busy={busyId === rowView.ids[index] || bulkBusy}
              onRowCheck={onRowCheck}
              onRowAction={onRowAction}
            />
          ))
        )}
      </div>
    </div>
  );
}
