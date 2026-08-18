import { useEffect, useState } from 'react';
import type { BuyerPortfolioVM } from '../helpers/buyer_dashboard.js';
import { fetchSmartAlertHistory, type SmartAlertEvent } from '../helpers/smart_alerts_api.js';
import { buyerEmptyCopy } from '../models/empty_state.js';
import type { RecommendationCard } from '../types/recommendation.js';
import { can } from '../helpers/permissions.js';
import * as auth from '../helpers/auth.js';
import { boundCustomerId } from '../helpers/buyer_session.js';
import { Button, ButtonLink } from './button.js';
import { HomeAlertFeed } from './home_alert_feed.js';
import type { HomeAlertCard } from '../helpers/home_alerts.js';
import { CommercialMetrics } from './commercial_metrics.js';
import { BuyerFraudKpiTiles } from './fraud_kpi_tiles.js';
import { StatusBadge } from './status_badge.js';

export type BuyerOverviewSectionProps = {
  loading?: boolean;
  portfolio?: BuyerPortfolioVM | null;
  perf?: Record<string, { count: number; nsPerOp: number; allocPerOp: number; bytesPerOp: number }>;
  error?: string | null;
  recActionLoading?: boolean;
  onRecommendationAction?: (actionId: string, card: RecommendationCard) => void | Promise<void>;
};

function RecommendationCards({
  cards,
  actionLoading,
  onAction,
}: {
  cards: RecommendationCard[];
  actionLoading: boolean;
  onAction?: (actionId: string, card: RecommendationCard) => void | Promise<void>;
}) {
  if (!cards.length) return null;
  return (
    <section className="section-block" data-testid="recommendation-cards">
      <h3 className="subsection-title">Recommendations</h3>
      <ul className="recommendation-list">
        {cards.map((card) => (
          <li key={card.id} className="recommendation-card" data-testid={`rec-${card.id}`}>
            <strong>{card.title}</strong>
            <p>{card.detail}</p>
            <div className="toolbar-row">
              {(card.actions ?? []).length > 0 ? (
                card.actions!.map((action) => {
                  const actionId = action.id ?? '';
                  return (
                    <Button
                      key={actionId}
                      label={action.label ?? actionId}
                      variant="secondary"
                      size="sm"
                      action={actionId}
                      disabled={actionLoading}
                      onClick={() => void onAction?.(actionId, card)}
                    />
                  );
                })
              ) : card.campaign_id ? (
                <ButtonLink
                  href={`/campaigns/${card.campaign_id}`}
                  label="Open campaign"
                  variant="secondary"
                  size="sm"
                />
              ) : null}
            </div>
            {card.confidence != null ? (
              <span className="text-muted text-xs">{` confidence ${Math.round(card.confidence * 100)}%`}</span>
            ) : null}
          </li>
        ))}
      </ul>
    </section>
  );
}

export function BuyerOverviewSection({
  loading = false,
  portfolio,
  perf,
  error,
  recActionLoading = false,
  onRecommendationAction,
}: BuyerOverviewSectionProps) {
  const user = auth.getUser();
  const perms = user?.permissions ?? [];
  const customerId = boundCustomerId(user);

  const portfolioAlerts = (portfolio?.alerts ?? []) as HomeAlertCard[];
  const [openSmartAlerts, setOpenSmartAlerts] = useState<SmartAlertEvent[]>([]);

  useEffect(() => {
    if (!customerId) {
      setOpenSmartAlerts([]);
      return;
    }
    void fetchSmartAlertHistory(customerId)
      .then((events) => {
        setOpenSmartAlerts(events.filter((e) => !e.acked_at).slice(0, 5));
      })
      .catch(() => setOpenSmartAlerts([]));
  }, [customerId]);

  if (loading) {
    return (
      <section className="buyer-overview" data-testid="buyer-overview">
        <h2 className="subsection-title">Buyer portfolio</h2>
        <p className="loading-hint">Loading portfolio metrics…</p>
      </section>
    );
  }

  if (error) {
    const copy = buyerEmptyCopy('campaigns_blocked');
    return (
      <section className="buyer-overview" data-testid="buyer-overview">
        <h2 className="subsection-title">Buyer portfolio</h2>
        <div className="stack stack--sm">
          <p className="empty-hint">{copy.title}</p>
          <p className="text-muted text-sm">{copy.description}</p>
          <p className="text-sm">{error}</p>
        </div>
      </section>
    );
  }

  if (!portfolio) {
    return (
      <section className="buyer-overview" data-testid="buyer-overview">
        <h2 className="subsection-title">Buyer portfolio</h2>
        <p className="empty-hint">Portfolio metrics unavailable.</p>
      </section>
    );
  }

  return (
    <section className="buyer-overview" data-testid="buyer-overview">
      <h2 className="subsection-title">Buyer portfolio</h2>
      {portfolio.kpis ? <CommercialMetrics kpis={portfolio.kpis} masked /> : null}
      {can(perms, 'audit:read') ? <BuyerFraudKpiTiles /> : null}
      {(portfolio.overspendCount ?? 0) > 0 ? (
        <p data-testid="buyer-overspend-alert">
          <StatusBadge
            status="warning"
            label={`${portfolio.overspendCount} campaign(s) at overspend risk`}
          />
        </p>
      ) : null}
      <RecommendationCards
        cards={(portfolio.recommendations ?? []) as RecommendationCard[]}
        actionLoading={recActionLoading}
        onAction={onRecommendationAction}
      />
      {portfolioAlerts.length > 0 ? <HomeAlertFeed alerts={portfolioAlerts} /> : null}
      <dl className="definition-list">
        <dt>Active campaigns</dt>
        <dd id="buyer-metric-active">{String(portfolio.active)}</dd>
        <dt>Paused campaigns</dt>
        <dd id="buyer-metric-paused">{String(portfolio.paused)}</dd>
        <dt>Archived campaigns</dt>
        <dd id="buyer-metric-archived">{String(portfolio.archived)}</dd>
        <dt>Impressions (7d)</dt>
        <dd id="buyer-metric-impressions">{String(portfolio.impressions7d)}</dd>
        <dt>Clicks (7d)</dt>
        <dd id="buyer-metric-clicks">{String(portfolio.clicks7d)}</dd>
        <dt>Campaigns in portfolio</dt>
        <dd>{String(portfolio.sampled)}</dd>
      </dl>
      <div className="stack stack--sm" data-testid="portfolio-attention-panel">
        <h3 className="subsection-title">Needs attention</h3>
        {portfolio.attention.length === 0 && openSmartAlerts.length === 0 ? (
          <p className="text-muted text-sm">
            No paused campaigns, pacing flags, or open smart alerts.
          </p>
        ) : (
          <ul className="plain-list">
            {portfolio.attention.map((row) => (
              <li key={row.id} className="plain-list__item">
                <a href={`/campaigns/${row.id}`}>{row.name}</a>
                {` — ${row.reason}`}
              </li>
            ))}
            {openSmartAlerts.map((evt) => (
              <li key={evt.id} className="plain-list__item" data-testid="attention-smart-alert">
                <a
                  href={`/integrations/smart-alerts?customer_id=${encodeURIComponent(customerId ?? '')}`}
                >
                  Smart alert
                </a>
                {` — ${evt.metric} ${evt.operator} ${evt.threshold} (observed ${evt.observed_value})`}
              </li>
            ))}
          </ul>
        )}
      </div>
      <div className="stack stack--sm">
        <h3 className="subsection-title">Next steps</h3>
        <ul className="plain-list">
          <li className="plain-list__item">
            <a href="/campaigns/portfolio">Portfolio view (drift sort)</a>
          </li>
          <li className="plain-list__item">
            <a href="/campaigns">Review campaign delivery</a>
          </li>
          <li className="plain-list__item">
            <a href="/reports">Reports</a>
          </li>
          <li className="plain-list__item">
            <a href="/reports/placements">Check placement report</a>
          </li>
          <li className="plain-list__item">
            <a href="/reports/keywords">Check keyword report</a>
          </li>
          {can(perms, 'audit:read') && customerId ? (
            <li className="plain-list__item">
              <a href={`/reports/ivt-by-source?customer_id=${encodeURIComponent(customerId)}`}>
                IVT by source
              </a>
            </li>
          ) : null}
          {can(perms, 'audit:read') && customerId ? (
            <li className="plain-list__item">
              <a href={`/dashboards/fraud?customer_id=${encodeURIComponent(customerId)}`}>
                Fraud dashboard
              </a>
            </li>
          ) : null}
        </ul>
      </div>
      {perf && Object.keys(perf).length > 0 ? (
        <div className="stack stack--sm">
          <h3 className="subsection-title">Critical path metrics</h3>
          <pre
            id="buyer-perf-metrics"
            className="code-block"
            aria-label="Critical path probe metrics"
          >
            {JSON.stringify(perf, null, 2)}
          </pre>
        </div>
      ) : null}
    </section>
  );
}
