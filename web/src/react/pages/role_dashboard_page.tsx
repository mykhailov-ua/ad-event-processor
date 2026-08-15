import { useCallback, useMemo } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { mountRoleDashboardPanel } from '../../panels/role_dashboard_panel.js';
import type { RouteContext } from '../../lib/router_types.js';
import { LegacyPanelHost } from '../components/legacy_panel_host.js';

/**
 * Role-specific dashboard (adops, cfo, accountant, fraud).
 */
export function RoleDashboardPage() {
  const params = useParams();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const queryKey = searchParams.toString();

  const ctx = useMemo((): RouteContext => ({
    params: Object.fromEntries(
      Object.entries(params).filter((entry): entry is [string, string] => entry[1] !== undefined),
    ),
    query: searchParams,
    navigate: (path: string) => navigate(path),
  }), [params, searchParams, navigate]);

  const mount = useCallback(
    (host: HTMLElement) => mountRoleDashboardPanel(host, ctx),
    [ctx],
  );

  return <LegacyPanelHost active mount={mount} deps={[params.role, queryKey]} />;
}
