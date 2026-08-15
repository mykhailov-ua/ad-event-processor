import { useCallback, useMemo } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { mountBillingPanel } from '../../panels/billing_panel.js';
import type { RouteContext } from '../../lib/router_types.js';
import { LegacyPanelHost } from '../components/legacy_panel_host.js';

/**
 * Billing wallet / ledger / invoices / exports tabs.
 */
export function BillingPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const queryKey = searchParams.toString();

  const ctx = useMemo((): RouteContext => ({
    params: {},
    query: searchParams,
    navigate: (path: string) => navigate(path),
  }), [searchParams, navigate]);

  const mount = useCallback(
    (host: HTMLElement) => mountBillingPanel(host, ctx),
    [ctx],
  );

  return <LegacyPanelHost active mount={mount} deps={[queryKey]} />;
}
