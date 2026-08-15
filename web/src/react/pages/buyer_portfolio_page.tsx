import { useCallback, useMemo } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { mountBuyerPortfolioPanel } from '../../panels/buyer_portfolio_panel.js';
import type { RouteContext } from '../../lib/router_types.js';
import { LegacyPanelHost } from '../components/legacy_panel_host.js';

/**
 * Buyer campaign portfolio drill-down.
 */
export function BuyerPortfolioPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const queryKey = searchParams.toString();

  const ctx = useMemo((): RouteContext => ({
    params: {},
    query: searchParams,
    navigate: (path: string) => navigate(path),
  }), [searchParams, navigate]);

  const mount = useCallback(
    (host: HTMLElement) => mountBuyerPortfolioPanel(host, ctx),
    [ctx],
  );

  return <LegacyPanelHost active mount={mount} deps={[queryKey]} />;
}
