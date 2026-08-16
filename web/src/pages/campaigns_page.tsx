import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { isBuyer, can } from '../helpers/permissions.js';
import { api } from '../helpers/api_client.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { buyerEmptyCopy } from '../models/empty_state.js';
import { fetchBuyerDashboard } from '../helpers/buyer_dashboard.js';
import { buyerCampaignStat, buyerCampaignIndex } from '../models/buyer.js';
import { attentionByCampaignId } from '../models/campaign_health.js';
import { formatUsdDecimal } from '../helpers/money.js';
import { displayLabel } from '../helpers/display_labels.js';
import {
  isCustomerUuid,
  shortCustomerId,
  touchCustomerContext,
} from '../helpers/customer_context.js';
import { pauseCampaign, resumeCampaign } from '../helpers/campaign_actions.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { createInFlightGuard } from '../lib/async_guard.js';
import { isParallelSlotError, parallelAll } from '../helpers/request_multiplex.js';
import { to } from '../lib/to.js';
import { createSortState, sortRows, toggleSort } from '../lib/table_sort.js';
import type { CampaignDTO, CampaignListResponse } from '../types/api/campaign.js';
import { useResource } from '../hooks/use_resource.js';
import { AlertBanner } from '../components/alert_banner.js';
import { Breadcrumbs } from '../components/breadcrumbs.js';
import { Button, ButtonLink } from '../components/button.js';
import { CampaignHealthBadge } from '../components/campaign_health_badge.js';
import { CampaignWizard, useCampaignWizard } from '../components/campaign_wizard.js';
import { CampaignStatusLegend } from '../components/status_legend.js';
import { Checkbox } from '../components/checkbox.js';
import { CopyableUuid } from '../components/copyable_uuid.js';
import { ErrorBlock } from '../components/error_block.js';
import { FilterToolbar } from '../components/filter_toolbar.js';
import { Icon } from '../components/icon.js';
import { PaginationBar } from '../components/pagination_bar.js';
import { RecentCustomers } from '../components/recent_customers.js';
import { StatusBadge } from '../components/status_badge.js';

const PAGE_SIZE = 50;
const CAMPAIGNS_EMPTY = buyerEmptyCopy('campaigns_empty');

async function runBulkCampaignAction(
  ids: string[],
  fn: (id: string) => Promise<unknown>,
): Promise<Error | null> {
  const tasks = ids.map((id) => async () => {
    const [, err] = await to(fn(id));
    return err;
  });
  const [results] = await to(parallelAll(tasks, 3));
  if (!results) return null;
  for (let i = 0; i < results.length; i += 1) {
    const slot = results[i];
    if (isParallelSlotError(slot)) {
      return slot.error instanceof Error ? slot.error : new Error(String(slot.error));
    }
    if (slot instanceof ConfirmCancelledError) return slot;
    if (slot) return slot;
  }
  return null;
}

function buildUrl(page: number, customerId: string, status: string) {
  const offset = page * PAGE_SIZE;
  const params = new URLSearchParams({
    limit: String(PAGE_SIZE),
    offset: String(offset),
  });
  if (customerId) params.set('customer_id', customerId);
  if (status) params.set('status', status);
  return `/api/v1/campaigns?${params.toString()}`;
}

function TableSkeleton({ cols, rows = 5 }: { cols: number; rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }, (_, rowIndex) => (
        <tr key={`skel-${rowIndex}`} className="data-table__row--skeleton" aria-hidden="true">
          {Array.from({ length: cols }, (__, colIndex) => (
            <td key={`skel-${rowIndex}-${colIndex}`}><span className="skeleton-bar" /></td>
          ))}
        </tr>
      ))}
    </>
  );
}

/**
 * Campaigns list with filters, bulk actions, and buyer/admin column layouts.
 */
export function CampaignsPage() {
  const navigate = useNavigate();
  const {
    wizardOpen,
    wizardCustomerId,
    openWizard,
    closeWizard,
  } = useCampaignWizard();
  const [searchParams] = useSearchParams();
  const customerInputRef = useRef<HTMLInputElement>(null);

  const user = auth.getUser();
  const sessionScoped = hasBoundCustomer(user?.role);
  const buyerView = isBuyer(user?.role);
  const canWrite = can(user?.permissions ?? [], 'campaigns:write');
  const tenantCustomerId = boundCustomerId(user);
  const queryCustomer = searchParams.get('customer_id')?.trim() ?? '';

  const [page, setPage] = useState(0);
  const [customerIdInput, setCustomerIdInput] = useState(
    sessionScoped ? tenantCustomerId : queryCustomer,
  );
  const [debouncedCustomerId, setDebouncedCustomerId] = useState(customerIdInput.trim());
  const [statusFilter, setStatusFilter] = useState('');
  const [showDetailedBudget, setShowDetailedBudget] = useState(false);
  const [sortState, setSortState] = useState(() => createSortState('name', 'asc'));
  const [licenseGrace, setLicenseGrace] = useState(false);
  const [buyerDashboard, setBuyerDashboard] = useState<Awaited<ReturnType<typeof fetchBuyerDashboard>> | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [actionLoading, setActionLoading] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [customerFilterError, setCustomerFilterError] = useState<string | null>(null);
  const bulkGateRef = useRef(createInFlightGuard());

  const skipFetch = sessionScoped && !tenantCustomerId;
  const bulkEnabled = !buyerView && canWrite && can(user?.permissions ?? [], 'campaigns:pause');

  const effectiveCustomerId = sessionScoped ? tenantCustomerId : debouncedCustomerId;

  useEffect(() => {
    const trimmed = customerIdInput.trim();
    const err = trimmed && !isCustomerUuid(trimmed) ? 'Invalid UUID format' : null;
    setCustomerFilterError(err);
    const timer = window.setTimeout(() => {
      if (!err) {
        setDebouncedCustomerId(trimmed);
        setPage(0);
      }
    }, 400);
    return () => window.clearTimeout(timer);
  }, [customerIdInput]);

  useEffect(() => {
    if (queryCustomer && isCustomerUuid(queryCustomer)) {
      touchCustomerContext(queryCustomer);
    }
  }, [queryCustomer]);

  useEffect(() => {
    void (async () => {
      const [res] = await Promise.all([
        api<{ license?: { state?: string } }>('/api/v1/meta').catch(() => null),
      ]);
      const licenseState = String(res?.data?.license?.state ?? '').toLowerCase();
      setLicenseGrace(licenseState === 'grace');
    })();
  }, []);

  useEffect(() => {
    if (!buyerView || skipFetch) return;
    const cid = effectiveCustomerId;
    if (!isCustomerUuid(cid)) {
      setBuyerDashboard(null);
      return;
    }
    void fetchBuyerDashboard(cid).then((dash) => setBuyerDashboard(dash));
  }, [buyerView, skipFetch, effectiveCustomerId]);

  const listUrl = skipFetch ? null : buildUrl(page, effectiveCustomerId, statusFilter);
  const { data, loading, error, reload } = useResource<CampaignListResponse>(listUrl, { skip: skipFetch });

  useEffect(() => {
    if (!loading && !error && isCustomerUuid(effectiveCustomerId)) {
      touchCustomerContext(effectiveCustomerId);
    }
  }, [loading, error, effectiveCustomerId]);

  const buyerIndex = useMemo(() => {
    if (!buyerView) return null;
    return buyerCampaignIndex(buyerDashboard?.campaigns);
  }, [buyerView, buyerDashboard?.campaigns]);

  const attentionMap = useMemo(
    () => (buyerView ? attentionByCampaignId(buyerDashboard?.attention) : {}),
    [buyerView, buyerDashboard?.attention],
  );

  const statFor = useCallback((c: CampaignDTO) => {
    const row = buyerIndex?.[c.id];
    const portfolioRow = row && typeof row === 'object' && !Array.isArray(row) ? row : null;
    return buyerCampaignStat(portfolioRow);
  }, [buyerIndex]);

  const sortAccessors = useMemo((): Record<string, (c: CampaignDTO) => unknown> => (
    buyerView
      ? {
        name: (c: CampaignDTO) => c.name ?? '',
        status: (c: CampaignDTO) => c.status ?? '',
        impressions: (c: CampaignDTO) => statFor(c).impressions,
        clicks: (c: CampaignDTO) => statFor(c).clicks,
        pacing_mode: (c: CampaignDTO) => c.pacing_mode ?? '',
      }
      : {
        name: (c: CampaignDTO) => c.name ?? '',
        status: (c: CampaignDTO) => c.status ?? '',
        budget_limit: (c: CampaignDTO) => Number(c.budget_limit ?? 0),
        current_spend: (c: CampaignDTO) => Number(c.current_spend ?? 0),
        pacing_mode: (c: CampaignDTO) => c.pacing_mode ?? '',
        customer_id: (c: CampaignDTO) => c.customer_id ?? '',
      }
  ), [buyerView, statFor]);

  const campaigns = useMemo(
    () => sortRows(data?.items ?? [], sortState, sortAccessors),
    [data?.items, sortState, sortAccessors],
  );

  const healthCtxFor = useCallback((c: CampaignDTO) => {
    const row = buyerIndex?.[c.id];
    const portfolioRow = row && typeof row === 'object' && !Array.isArray(row) ? row : undefined;
    return {
      portfolioRow,
      attentionReason: attentionMap[c.id],
      marginBreach: !!c.margin_breach
        || !!(portfolioRow as { margin_breach?: boolean } | undefined)?.margin_breach,
      licenseGrace,
    };
  }, [buyerIndex, attentionMap, licenseGrace]);

  const bulkPause = async () => {
    if (!bulkGateRef.current.tryAcquire()) return;
    const ids = [...selected];
    if (ids.length === 0) {
      bulkGateRef.current.release();
      return;
    }
    setActionLoading(true);
    setActionError(null);
    const err = await runBulkCampaignAction(ids, pauseCampaign);
    setActionLoading(false);
    if (err && !(err instanceof ConfirmCancelledError)) {
      setActionError(err.message ?? 'Bulk pause failed');
    } else if (!err) {
      setSelected(new Set());
      reload();
    }
    bulkGateRef.current.release();
  };

  const bulkResume = async () => {
    if (!bulkGateRef.current.tryAcquire()) return;
    const ids = [...selected];
    if (ids.length === 0) {
      bulkGateRef.current.release();
      return;
    }
    setActionLoading(true);
    setActionError(null);
    const err = await runBulkCampaignAction(ids, resumeCampaign);
    setActionLoading(false);
    if (err && !(err instanceof ConfirmCancelledError)) {
      setActionError(err.message ?? 'Bulk resume failed');
    } else if (!err) {
      setSelected(new Set());
      reload();
    }
    bulkGateRef.current.release();
  };

  if (skipFetch) {
    const copy = buyerEmptyCopy('session_customer');
    return (
      <section>
        <h1>Campaigns</h1>
        <p>{copy.title}</p>
        <p>{copy.description}</p>
      </section>
    );
  }

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load campaigns" />;
  }

  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / PAGE_SIZE) || 1;
  const colCount = bulkEnabled ? (buyerView ? 7 : 8) : (buyerView ? 6 : 7);

  const onSort = (key: string) => {
    setSortState((prev) => {
      const next = { ...prev };
      toggleSort(next, key);
      return next;
    });
  };

  const sortHeader = (label: string, key: string) => {
    const active = sortState.key === key;
    const iconName = active
      ? (sortState.dir === 'asc' ? 'chevron-up' : 'chevron-down')
      : 'arrow-up-down';
    return (
      <th
        key={key}
        scope="col"
        className={[
          'data-table__th--sortable',
          active ? 'data-table__th--sorted' : '',
        ].filter(Boolean).join(' ')}
        aria-sort={active ? (sortState.dir === 'asc' ? 'ascending' : 'descending') : 'none'}
        tabIndex={0}
        onClick={() => onSort(key)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            onSort(key);
          }
        }}
      >
        <span className="data-table__th-label">{label}</span>
        <Icon name={iconName} size={13} className="data-table__sort-icon" />
      </th>
    );
  };

  return (
    <>
      <div className="page-header">
        {effectiveCustomerId && isCustomerUuid(effectiveCustomerId) ? (
          <Breadcrumbs items={[
            { label: 'Customers', href: '/customers' },
            { label: shortCustomerId(effectiveCustomerId, 14), href: `/customers/${effectiveCustomerId}` },
            { label: 'Campaigns' },
          ]}
          />
        ) : null}
        <div className="page-header__row cluster--actions">
          <div className="flex items-center gap-3">
            <Icon name="megaphone" size={22} className="text-muted" />
            <h1 className="page-header__title">Campaigns</h1>
          </div>
          {buyerView ? (
            <ButtonLink href="/campaigns/portfolio" label="Portfolio view" variant="secondary" size="sm" />
          ) : null}
          <Button
            label={showDetailedBudget ? 'Precision: Micro ($00.000000)' : 'Precision: Standard ($00.00)'}
            variant="secondary"
            size="sm"
            icon="sliders"
            title="Toggle budget precision (Standard $00.00 / Detailed $00.000000)"
            onClick={() => setShowDetailedBudget((v) => !v)}
          />
          {canWrite && effectiveCustomerId && isCustomerUuid(effectiveCustomerId) ? (
            <Button
              label="Create campaign"
              variant="primary"
              size="sm"
              onClick={() => openWizard(effectiveCustomerId)}
            />
          ) : null}
          <span className="text-muted text-sm">{loading ? '' : `${total} total`}</span>
        </div>
        <RecentCustomers tenant={sessionScoped && !buyerView} />
      </div>

      <div className="mb-4">
        <FilterToolbar
          leading={!sessionScoped ? (
            <input
              ref={customerInputRef}
              id="campaigns-customer-input"
              type="text"
              className="form-input form-input--sm"
              placeholder="Customer UUID…"
              value={customerIdInput}
              onChange={(e) => setCustomerIdInput(e.target.value)}
            />
          ) : (
            sessionScoped && effectiveCustomerId && !buyerView ? (
              <p className="text-muted text-hint">
                Customer:{' '}
                <a href={`/customers/${effectiveCustomerId}`} className="font-mono">
                  {effectiveCustomerId}
                </a>
              </p>
            ) : null
          )}
          chips={[
            { value: '', label: 'All' },
            { value: 'ACTIVE', label: 'Active' },
            { value: 'PAUSED', label: 'Paused' },
            { value: 'ARCHIVED', label: 'Archived' },
          ]}
          chipSelected={statusFilter}
          onChipSelect={(value) => {
            setStatusFilter(value);
            setPage(0);
          }}
          pagination={totalPages > 1 ? (
            <PaginationBar
              label={`${page + 1} / ${totalPages}`}
              prevDisabled={page === 0}
              nextDisabled={page >= totalPages - 1}
              onPrev={() => setPage((p) => Math.max(0, p - 1))}
              onNext={() => setPage((p) => p + 1)}
            />
          ) : null}
        />
      </div>

      {customerFilterError ? (
        <AlertBanner variant="error" message={customerFilterError} />
      ) : null}
      {!effectiveCustomerId && !sessionScoped && !loading && campaigns.length === 0 ? (
        <AlertBanner variant="info" message="Enter a customer UUID to load the campaign list." />
      ) : null}
      {effectiveCustomerId ? <CampaignStatusLegend /> : null}

      {bulkEnabled && selected.size > 0 ? (
        <div id="campaigns-bulk-actions" className="toolbar-row mb-3">
          <Button
            label={`Pause selected (${selected.size})`}
            variant="secondary"
            size="sm"
            action="pause"
            disabled={actionLoading}
            onClick={() => void bulkPause()}
          />
          <Button
            label={`Resume selected (${selected.size})`}
            variant="secondary"
            size="sm"
            action="resume"
            disabled={actionLoading}
            onClick={() => void bulkResume()}
          />
        </div>
      ) : null}
      {actionError ? <p className="text-danger text-sm mb-3">{actionError}</p> : null}

      <div className="table-wrapper table-wrapper--scroll elevation-raised">
        <table className="data-table">
          <thead>
            <tr>
              {bulkEnabled ? (
                <th scope="col">
                  <Checkbox
                    id="campaigns-select-all"
                    label="Select all"
                    checked={campaigns.length > 0 && campaigns.every((c) => selected.has(c.id))}
                    onChange={(checked) => {
                      if (checked) setSelected(new Set(campaigns.map((c) => c.id)));
                      else setSelected(new Set());
                    }}
                  />
                </th>
              ) : null}
              {sortHeader('Name', 'name')}
              {sortHeader('Status', 'status')}
              {buyerView
                ? sortHeader('Impr. (7d)', 'impressions')
                : sortHeader('Budget limit', 'budget_limit')}
              {buyerView
                ? sortHeader('Clicks (7d)', 'clicks')
                : sortHeader('Spend', 'current_spend')}
              {sortHeader('Pacing', 'pacing_mode')}
              <th scope="col">Health</th>
              {!buyerView ? sortHeader('Customer', 'customer_id') : null}
            </tr>
          </thead>
          <tbody>
            {loading && campaigns.length === 0 ? <TableSkeleton cols={colCount} /> : null}
            {!loading && campaigns.length === 0 && effectiveCustomerId ? (
              <tr>
                <td colSpan={colCount}>
                  <div className="empty-state">
                    <div className="empty-state__title">{CAMPAIGNS_EMPTY.title}</div>
                    <div className="empty-state__desc text-muted text-sm">{CAMPAIGNS_EMPTY.description}</div>
                    <Button
                      label={CAMPAIGNS_EMPTY.actionLabel ?? 'Continue'}
                      variant="secondary"
                      size="sm"
                      className="empty-state__action"
                      onClick={() => navigate(CAMPAIGNS_EMPTY.actionHref ?? '/reports/placements')}
                    />
                  </div>
                </td>
              </tr>
            ) : null}
            {!loading && campaigns.length === 0 && !effectiveCustomerId ? (
              <tr>
                <td colSpan={colCount}>
                  <div className="empty-state">
                    <div className="empty-state__title">Customer required</div>
                    <div className="empty-state__desc text-muted text-sm">
                      Enter a customer UUID above to load campaigns.
                    </div>
                    <Button
                      label="Focus customer field"
                      variant="secondary"
                      size="sm"
                      className="empty-state__action"
                      onClick={() => customerInputRef.current?.focus()}
                    />
                  </div>
                </td>
              </tr>
            ) : null}
            {campaigns.map((c) => (
              <tr
                key={c.id}
                id={`row-campaign-${c.id}`}
                className="data-table__row--clickable"
                tabIndex={0}
                onClick={() => navigate(`/campaigns/${c.id}`)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    navigate(`/campaigns/${c.id}`);
                  }
                }}
              >
                {bulkEnabled ? (
                  <td onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={selected.has(c.id)}
                      label={`Select ${c.name}`}
                      onChange={(checked) => {
                        setSelected((prev) => {
                          const next = new Set(prev);
                          if (checked) next.add(c.id);
                          else next.delete(c.id);
                          return next;
                        });
                      }}
                    />
                  </td>
                ) : null}
                <td>{c.name}</td>
                <td><StatusBadge status={c.status} kind="campaign" /></td>
                {buyerView ? (
                  <td>{String(statFor(c).impressions || '—')}</td>
                ) : (
                  <td className="font-mono">
                    {formatUsdDecimal(c.budget_limit ?? '0.00', { full: showDetailedBudget })}
                  </td>
                )}
                {buyerView ? (
                  <td>{String(statFor(c).clicks || '—')}</td>
                ) : (
                  <td className="font-mono">
                    {formatUsdDecimal(c.current_spend ?? '0.00', { full: showDetailedBudget })}
                  </td>
                )}
                <td>{displayLabel(c.pacing_mode)}</td>
                <td><CampaignHealthBadge campaign={c} ctx={healthCtxFor(c)} /></td>
                {!buyerView ? (
                  <td onClick={(e) => e.stopPropagation()}>
                    {c.customer_id ? (
                      <a href={`/customers/${c.customer_id}`}>
                        <CopyableUuid uuid={c.customer_id} />
                      </a>
                    ) : '—'}
                  </td>
                ) : null}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <CampaignWizard
        open={wizardOpen}
        customerId={wizardCustomerId}
        onClose={closeWizard}
        onCreated={(cid) => {
          closeWizard();
          navigate(`/campaigns/${cid}`);
        }}
      />
    </>
  );
}
