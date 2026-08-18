import { useCallback, useEffect, useMemo, useState } from 'react';
import type { DashboardSummary, IncidentSnapshot, OpsDoctorSummary } from '../types/index.js';
import { to } from '../lib/to.js';
import { api, ApiError, type ApiResult } from '../helpers/api_client.js';
import { isParallelSlotError, parallelAll } from '../helpers/request_multiplex.js';
import { can, isBuyer } from '../helpers/permissions.js';
import * as auth from '../helpers/auth.js';
import { boundCustomerId } from '../helpers/buyer_session.js';
import {
  fetchBuyerDashboard,
  invalidateBuyerDashboard,
  type BuyerPortfolioVM,
} from '../helpers/buyer_dashboard.js';
import { probeReport, probeReset, type ProbeReport } from '../helpers/perf_probe.js';
import { pauseCampaign, resumeCampaign } from '../helpers/campaign_actions.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { displayLabel, formatYesNo } from '../helpers/display_labels.js';
import { buildHomeAlerts } from '../helpers/home_alerts.js';
import { connectOpsLiveFeed } from '../helpers/ops_live_feed.js';
import type { RecommendationCard } from '../types/recommendation.js';
import { BuyerOverviewSection } from '../components/buyer_overview_section.js';
import { ButtonLink } from '../components/button.js';
import { DoctorPanel } from '../components/doctor_panel.js';
import { ErrorBlock } from '../components/error_block.js';
import { FreshnessBadge } from '../components/freshness_badge.js';
import { HomeAlertFeed } from '../components/home_alert_feed.js';
import { Icon } from '../components/icon.js';

type PartialSourceError = { source?: string; code?: string };

type OverviewMeta = {
  version?: string;
  license?: { state?: string; valid_until?: string };
};

type QuickLink = {
  href: string;
  label: string;
  icon: string;
};

function OverviewMetric({ label, value, icon }: { label: string; value: string; icon: string }) {
  return (
    <div className="metric-card">
      <div className="metric-card__head">
        <div className="metric-card__label">{label}</div>
        <Icon name={icon} size={16} className="text-muted" />
      </div>
      <div className="metric-card__value font-mono">{value}</div>
    </div>
  );
}

function buildQuickLinks(perms: string[]): QuickLink[] {
  const links: QuickLink[] = [];
  if (can(perms, 'campaigns:read') || can(perms, 'campaigns:read:masked')) {
    links.push({ href: '/campaigns', label: 'Campaigns', icon: 'megaphone' });
    links.push({ href: '/margin-guard', label: 'Margin Guard', icon: 'trending-down' });
  }
  if (can(perms, 'customers:read') || can(perms, 'billing:read')) {
    links.push({ href: '/billing', label: 'Billing', icon: 'credit-card' });
  }
  if (can(perms, 'shards:read')) {
    links.push({ href: '/ops', label: 'Operations', icon: 'server' });
  }
  if (can(perms, 'settings:read')) {
    links.push({ href: '/settings', label: 'Settings', icon: 'settings' });
  }
  return links;
}

function deriveFreshness(
  incidents: IncidentSnapshot | null,
  summary: DashboardSummary | null,
  doctor: OpsDoctorSummary | null
) {
  if (incidents?.partial) {
    return { stale: true, lagSeconds: 0 };
  }
  const chCard = summary?.services?.find((s) =>
    (s.name || '').toLowerCase().includes('clickhouse')
  );
  if (chCard?.status && chCard.status !== 'ok' && chCard.status !== 'disabled') {
    return { stale: true, lagSeconds: 0 };
  }
  const chCheck = doctor?.checks?.find((c) => (c.id || '').toLowerCase().includes('clickhouse'));
  if (chCheck?.status && chCheck.status !== 'ok' && chCheck.status !== 'pass') {
    return { stale: true, lagSeconds: 0 };
  }
  return null;
}

export function OverviewPage() {
  const user = auth.getUser();
  const perms = user?.permissions ?? [];
  const buyerMode = isBuyer(user?.role);
  const canOps = can(perms, 'shards:read');

  const [loading, setLoading] = useState(true);
  const [blockError, setBlockError] = useState<unknown>(null);
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [doctor, setDoctor] = useState<OpsDoctorSummary | null>(null);
  const [incidents, setIncidents] = useState<IncidentSnapshot | null>(null);
  const [meta, setMeta] = useState<OverviewMeta | null>(null);
  const [partialErrors, setPartialErrors] = useState<PartialSourceError[]>([]);
  const [partialDismissed, setPartialDismissed] = useState(false);
  const [buyerPortfolio, setBuyerPortfolio] = useState<BuyerPortfolioVM | null>(null);
  const [buyerError, setBuyerError] = useState<string | null>(null);
  const [buyerPerf, setBuyerPerf] = useState<ProbeReport | null>(null);
  const [recActionLoading, setRecActionLoading] = useState(false);

  const quickLinks = useMemo(() => buildQuickLinks(perms), [perms]);

  const homeAlerts = useMemo(
    () =>
      buildHomeAlerts({
        summary,
        doctor,
        incidents,
        meta,
        buyerPortfolio: buyerPortfolio as Parameters<typeof buildHomeAlerts>[0]['buyerPortfolio'],
        canOps,
        buyerMode,
      }),
    [summary, doctor, incidents, meta, buyerPortfolio, canOps, buyerMode]
  );

  const freshness = useMemo(
    () => deriveFreshness(incidents, summary, doctor),
    [incidents, summary, doctor]
  );

  const loadData = useCallback(async () => {
    setLoading(true);
    setBlockError(null);

    type OverviewSlot = ApiResult | BuyerPortfolioVM | { error: unknown };
    const tasks: Array<() => Promise<OverviewSlot>> = [() => api('/api/v1/meta')];
    let buyerTaskIndex = -1;

    if (canOps) {
      tasks.push(
        () => api('/api/v1/ops/doctor'),
        () => api('/api/v1/ops/incidents').catch((err: unknown) => ({ error: err })),
        () => api('/api/v1/ops/dashboard/summary')
      );
    }

    if (buyerMode) {
      const customerId = boundCustomerId(user);
      buyerTaskIndex = tasks.length;
      tasks.push(async () => {
        if (!customerId) return { error: new Error('customer_id missing in session') };
        probeReset();
        const [portfolio, err] = await to(fetchBuyerDashboard(customerId));
        if (err) return { error: err };
        return portfolio as BuyerPortfolioVM;
      });
    }

    const [results, err] = await to(parallelAll(tasks, 3));
    if (err) {
      setBlockError(err);
      setLoading(false);
      return;
    }

    const metaRes = results[0];
    if (!isParallelSlotError(metaRes) && metaRes && 'data' in metaRes && metaRes.data) {
      setMeta(metaRes.data as OverviewMeta);
    }

    if (canOps) {
      const docRes = results[1];
      const incRes = results[2];
      const sumRes = results[3];

      if (!isParallelSlotError(docRes) && docRes && 'data' in docRes && docRes.data) {
        setDoctor(docRes.data as OpsDoctorSummary);
      }
      if (!isParallelSlotError(sumRes) && sumRes && 'data' in sumRes && sumRes.data) {
        setSummary(sumRes.data as DashboardSummary);
      }
      if (
        !isParallelSlotError(incRes) &&
        incRes &&
        'data' in incRes &&
        incRes.data &&
        !('error' in incRes && (incRes as { error?: unknown }).error)
      ) {
        setIncidents(incRes.data as IncidentSnapshot);
      }

      const errors: PartialSourceError[] = [];
      if (
        isParallelSlotError(incRes) ||
        (incRes && 'error' in incRes && (incRes as { error?: unknown }).error)
      ) {
        const incErr = (incRes as { error: unknown }).error;
        if (incErr instanceof ApiError && incErr.payload?.errors?.length) {
          errors.push(...(incErr.payload.errors as PartialSourceError[]));
        } else {
          const view = mapServiceError(incErr);
          pushToastMessage({ title: view.title, message: view.message, code: view.code });
        }
      } else if (!isParallelSlotError(incRes) && incRes && 'data' in incRes) {
        const incData = incRes.data as IncidentSnapshot | null;
        if (incData?.errors?.length) errors.push(...incData.errors);
      }
      setPartialErrors(errors);
    }

    if (buyerMode && buyerTaskIndex >= 0) {
      const buyerRes = results[buyerTaskIndex];
      if (
        isParallelSlotError(buyerRes) ||
        (buyerRes && typeof buyerRes === 'object' && 'error' in buyerRes && !('data' in buyerRes))
      ) {
        const buyerErr = (buyerRes as { error: unknown }).error;
        setBuyerError(
          (buyerErr instanceof Error ? buyerErr.message : null) || 'Failed to load buyer portfolio'
        );
        setBuyerPortfolio(null);
        setBuyerPerf(null);
      } else {
        setBuyerPortfolio(buyerRes as BuyerPortfolioVM);
        setBuyerPerf(probeReport());
        setBuyerError(null);
      }
    }

    setLoading(false);
  }, [buyerMode, canOps, user]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  useEffect(() => {
    if (!canOps) return undefined;

    const feed = connectOpsLiveFeed({
      pollMs: 30_000,
      onTick: (payload) => {
        if (!payload.summary) return;
        setSummary(payload.summary as DashboardSummary);
      },
      onPoll: async () => {
        const [sumRes] = await to(api('/api/v1/ops/dashboard/summary'));
        if (!sumRes?.data) return;
        setSummary(sumRes.data as DashboardSummary);
      },
    });

    return () => feed.destroy();
  }, [canOps]);

  const handleRecommendationAction = useCallback(
    async (actionId: string, card: RecommendationCard) => {
      const campaignId = card.campaign_id;
      if (!campaignId) return;
      if (actionId === 'edit_budget') {
        window.location.href = `/campaigns/${campaignId}`;
        return;
      }
      setRecActionLoading(true);
      const [, err] = await to(
        (async () => {
          if (actionId === 'pause') await pauseCampaign(campaignId);
          else if (actionId === 'resume') await resumeCampaign(campaignId);
        })()
      );
      setRecActionLoading(false);
      if (err && !(err instanceof ConfirmCancelledError)) {
        pushToastMessage({ title: 'Action failed', message: err.message ?? String(err) });
      }
      if (!err || err instanceof ConfirmCancelledError) {
        invalidateBuyerDashboard(boundCustomerId(user));
        const [portfolio, loadErr] = await to(fetchBuyerDashboard(boundCustomerId(user)));
        if (!loadErr && portfolio) {
          setBuyerPortfolio(portfolio);
          setBuyerPerf(probeReport());
        }
      }
    },
    [user]
  );

  if (blockError) {
    return <ErrorBlock error={blockError} />;
  }

  return (
    <>
      <div className="page-header">
        <div className="page-header__row">
          <div className="flex items-center gap-2">
            <h1 className="page-header__title">Overview</h1>
            {freshness ? (
              <FreshnessBadge stale={freshness.stale} lagSeconds={freshness.lagSeconds} />
            ) : null}
          </div>
          {meta?.version ? <span className="text-muted text-sm">{`v${meta.version}`}</span> : null}
        </div>
        {quickLinks.length > 0 ? (
          <div className="page-header__links">
            {quickLinks.map((link) => (
              <ButtonLink
                key={link.href}
                href={link.href}
                label={link.label}
                variant="secondary"
                size="sm"
                icon={link.icon}
              />
            ))}
          </div>
        ) : null}
      </div>

      {partialErrors.length > 0 && !partialDismissed ? (
        <div className="status-hint status-hint--error">
          <div className="flex items-center justify-between w-full">
            <span>
              {`Partial degradation: ${partialErrors.map((e) => `${e.source ?? '?'} (${e.code ?? 'err'})`).join('; ')}`}
            </span>
            <button
              type="button"
              className="alert-banner__close"
              onClick={() => setPartialDismissed(true)}
            >
              Dismiss
            </button>
          </div>
        </div>
      ) : null}

      {loading ? <div className="text-muted">Loading…</div> : null}

      {!loading && (canOps || buyerMode) && homeAlerts.length > 0 ? (
        <HomeAlertFeed alerts={homeAlerts} />
      ) : null}

      {!loading && canOps && summary ? (
        <div className="grid-stats">
          <OverviewMetric
            label="Outbox pending"
            value={String(summary.outbox_pending ?? 0)}
            icon="activity"
          />
          <OverviewMetric
            label="RPS (estimate)"
            value={String(summary.rps_estimate ?? '—')}
            icon="zap"
          />
          <OverviewMetric
            label="Drift alert"
            value={formatYesNo(summary.drift_alert)}
            icon="alert-triangle"
          />
          <OverviewMetric
            label="Emergency breaker"
            value={displayLabel(summary.emergency_breaker)}
            icon="shield"
          />
        </div>
      ) : null}

      {canOps ? (
        <DoctorPanel doctor={doctor} services={summary?.services} loading={loading} />
      ) : null}

      {!loading && buyerMode ? (
        <BuyerOverviewSection
          loading={false}
          portfolio={buyerPortfolio}
          perf={buyerPerf ?? undefined}
          error={buyerError}
          recActionLoading={recActionLoading}
          onRecommendationAction={handleRecommendationAction}
        />
      ) : null}

      {!loading && !canOps && !buyerMode ? (
        <p className="text-muted">Use the quick links above to manage campaigns and billing.</p>
      ) : null}
    </>
  );
}
