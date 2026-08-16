import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { boundCustomerId, hasBoundCustomer } from '../helpers/buyer_session.js';
import { fetchBuyerDashboard, invalidateBuyerDashboard } from '../helpers/buyer_dashboard.js';
import { visiblePortfolioRows, type BuyerPortfolioVM, type PortfolioRowsCache } from '../models/buyer.js';
import { pauseCampaign, resumeCampaign } from '../helpers/campaign_actions.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { isParallelSlotError, parallelAll } from '../helpers/request_multiplex.js';
import { perfOverlayEnabled } from '../helpers/perf_display.js';
import { probeReport } from '../helpers/perf_probe.js';
import { buyerEmptyCopy } from '../models/empty_state.js';
import { createInFlightGuard } from '../lib/async_guard.js';
import { displayLabel } from '../helpers/display_labels.js';
import { to } from '../lib/to.js';
import { Button } from '../components/button.js';
import { Checkbox } from '../components/checkbox.js';
import { ErrorBlock } from '../components/error_block.js';
import { StatusBadge } from '../components/status_badge.js';

async function runBulk(ids: string[], fn: (id: string) => Promise<unknown>): Promise<Error | null> {
  const tasks = ids.map((id) => async () => {
    const [, err] = await to(fn(id));
    return err;
  });
  const results = await parallelAll(tasks, 3);
  for (let i = 0; i < results.length; i++) {
    const slot = results[i];
    if (isParallelSlotError(slot)) {
      return slot.error instanceof Error ? slot.error : new Error(String(slot.error));
    }
    if (slot instanceof ConfirmCancelledError) return slot;
    if (slot) return slot;
  }
  return null;
}

/**
 * Buyer campaign portfolio with drift sort and bulk pause/resume.
 */
export function BuyerPortfolioPage() {
  const navigate = useNavigate();
  const user = auth.getUser();
  const customerId = boundCustomerId(user);
  const sessionScoped = hasBoundCustomer(user?.role);
  const rowCacheRef = useRef<PortfolioRowsCache>({ portfolio: null, filter: '', rows: null });
  const bulkGateRef = useRef(createInFlightGuard());

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [portfolio, setPortfolio] = useState<BuyerPortfolioVM | null>(null);
  const [statusFilter, setStatusFilter] = useState('');
  const [selected, setSelected] = useState<Set<string>>(() => new Set());
  const [actionError, setActionError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);

  const rows = useMemo(
    () => visiblePortfolioRows(portfolio, statusFilter, rowCacheRef.current),
    [portfolio, statusFilter],
  );

  const allSelected = rows.length > 0 && rows.every((r) => selected.has(r.row.id));

  const load = useCallback(async () => {
    if (!sessionScoped || !customerId) return;
    setLoading(true);
    setError(null);
    rowCacheRef.current.rows = null;
    const [data, err] = await to(fetchBuyerDashboard(customerId));
    if (err) {
      setError(err.message ?? 'Failed to load portfolio');
      setPortfolio(null);
      setLoading(false);
      return;
    }
    setPortfolio(data);
    setLoading(false);
    rowCacheRef.current.rows = null;
  }, [customerId, sessionScoped]);

  useEffect(() => {
    void load();
  }, [load]);

  const toggleSelect = (id: string, checked: boolean) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (checked) next.add(id);
      else next.delete(id);
      return next;
    });
  };

  const toggleSelectAll = (checked: boolean) => {
    if (!checked) {
      setSelected(new Set());
      return;
    }
    setSelected(new Set(rows.map((r) => r.row.id)));
  };

  const runBulkAction = async (kind: 'pause' | 'resume') => {
    if (!bulkGateRef.current.tryAcquire()) return;
    const ids = [...selected];
    if (ids.length === 0) {
      bulkGateRef.current.release();
      return;
    }
    setActionLoading(true);
    setActionError(null);
    const fn = kind === 'pause' ? pauseCampaign : resumeCampaign;
    const err = await runBulk(ids, fn);
    setActionLoading(false);
    if (err && !(err instanceof ConfirmCancelledError)) {
      setActionError(err.message ?? `Bulk ${kind} failed`);
    }
    setSelected(new Set());
    invalidateBuyerDashboard(customerId);
    bulkGateRef.current.release();
    await load();
  };

  if (!sessionScoped || !customerId) {
    const copy = buyerEmptyCopy('session_customer');
    return (
      <div className="page-header">
        <h1 className="page-header__title">Portfolio</h1>
        <p className="page-header__desc">{copy.title}</p>
        <p className="text-muted text-sm">{copy.description}</p>
      </div>
    );
  }

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Portfolio unavailable" />;
  }

  return (
    <section className="stack" data-testid="buyer-portfolio-view">
      <div className="page-header">
        <h1 className="page-header__title">Portfolio</h1>
        <p className="page-header__desc">
          Sorted by pacing drift.{' '}
          <a href="/campaigns">Table view</a>
        </p>
      </div>

      {loading ? <p className="loading-hint">Loading portfolio…</p> : null}

      {!loading ? (
        <div className="filter-row">
          <label className="form-label form-label--flush" htmlFor="portfolio-status-filter">
            Status filter
          </label>
          <select
            id="portfolio-status-filter"
            className="form-select min-w-40"
            value={statusFilter}
            onChange={(e) => {
              rowCacheRef.current.rows = null;
              setStatusFilter(e.currentTarget.value);
            }}
          >
            <option value="">All</option>
            <option value="ACTIVE">Active</option>
            <option value="PAUSED">Paused</option>
            <option value="ARCHIVED">Archived</option>
          </select>
        </div>
      ) : null}

      {selected.size > 0 ? (
        <div id="portfolio-bulk-actions" className="toolbar-row">
          <Button
            label={`Pause selected (${selected.size})`}
            variant="secondary"
            size="sm"
            action="pause"
            disabled={actionLoading}
            onClick={() => void runBulkAction('pause')}
          />
          <Button
            label={`Resume selected (${selected.size})`}
            variant="secondary"
            size="sm"
            action="resume"
            disabled={actionLoading}
            onClick={() => void runBulkAction('resume')}
          />
        </div>
      ) : null}

      {actionError ? <p className="text-danger text-sm">{actionError}</p> : null}

      {!loading ? (
        <div className="table-wrapper table-wrapper--scroll elevation-raised">
          <table className="data-table">
            <thead>
              <tr>
                <th>
                  <Checkbox
                    id="portfolio-select-all"
                    checked={allSelected}
                    onChange={toggleSelectAll}
                    label="Select all"
                  />
                </th>
                <th>Campaign</th>
                <th>Status</th>
                <th>Drift %</th>
                <th>Util %</th>
                <th>Risk</th>
                <th>Impr. 7d</th>
                <th>Clicks 7d</th>
                <th>Pacing</th>
              </tr>
            </thead>
            <tbody id="portfolio-tbody">
              {rows.map(({ row: c, driftScore }) => (
                <tr
                  key={c.id}
                  id={`portfolio-row-${c.id}`}
                  tabIndex={0}
                  className="clickable-row"
                  onClick={() => navigate(`/campaigns/${c.id}`)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault();
                      navigate(`/campaigns/${c.id}`);
                    }
                  }}
                >
                  <td onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={selected.has(c.id)}
                      onChange={(checked) => toggleSelect(c.id, checked)}
                    />
                  </td>
                  <td>{c.name ?? c.id}</td>
                  <td><StatusBadge status={c.status ?? ''} /></td>
                  <td>
                    {c.pacing_drift_pct != null
                      ? `${Number(c.pacing_drift_pct).toFixed(0)}%`
                      : String(driftScore)}
                  </td>
                  <td>{c.utilization_pct != null ? `${c.utilization_pct.toFixed(0)}%` : '—'}</td>
                  <td>
                    {c.overspend_risk ? <StatusBadge status="warning" label="risk" /> : '—'}
                  </td>
                  <td>{String(c.impressions_7d ?? 0)}</td>
                  <td>{String(c.clicks_7d ?? 0)}</td>
                  <td>{displayLabel(c.pacing_mode)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      {perfOverlayEnabled() ? (
        <pre id="portfolio-perf-metrics" aria-label="Critical path metrics">
          {JSON.stringify(probeReport(), null, 2)}
        </pre>
      ) : null}
    </section>
  );
}
