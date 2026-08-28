import { useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  buildPublisherDashboardUrl,
  buildPublisherStatementsUrl,
  type PublisherDashboard,
  type PublisherStatementListResponse,
} from '../helpers/publisher_api.js';
import { useResource } from '../helpers/use_resource.js';
import { PublisherHub } from '../ui/publisher/publisher_hub.js';

const STATEMENT_LIMIT = 25;

function defaultRange(): { from: string; to: string } {
  const to = new Date();
  const from = new Date(to.getTime() - 30 * 24 * 60 * 60 * 1000);
  return { from: from.toISOString(), to: to.toISOString() };
}

function parseTab(raw: string | null): string {
  if (raw === 'statements' || raw === 'supply') return raw;
  return 'dashboard';
}

function parseOffset(raw: string | null): number {
  const value = Number.parseInt(raw ?? '', 10);
  if (!Number.isFinite(value) || value < 0) return 0;
  return value;
}

export function PublisherPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const activeTab = parseTab(searchParams.get('tab'));
  const offset = parseOffset(searchParams.get('offset'));
  const from = searchParams.get('from') ?? defaultRange().from;
  const to = searchParams.get('to') ?? defaultRange().to;

  const dashboardUrl =
    activeTab === 'dashboard'
      ? buildPublisherDashboardUrl({ from, to })
      : null;
  const statementsUrl =
    activeTab === 'statements'
      ? buildPublisherStatementsUrl({ from, to, limit: STATEMENT_LIMIT, offset })
      : null;

  const {
    data: dashboard,
    loading: dashboardLoading,
    error: dashboardError,
  } = useResource<PublisherDashboard>(dashboardUrl);
  const {
    data: statements,
    loading: statementsLoading,
    error: statementsError,
  } = useResource<PublisherStatementListResponse>(statementsUrl);

  const onTabChange = useCallback(
    (tabId: string) => {
      const next = new URLSearchParams(searchParams);
      if (tabId === 'dashboard') next.delete('tab');
      else next.set('tab', tabId);
      next.delete('offset');
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  const onRangeChange = useCallback(
    (nextFrom: string, nextTo: string) => {
      const next = new URLSearchParams(searchParams);
      next.set('from', nextFrom);
      next.set('to', nextTo);
      next.delete('offset');
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  const onOffsetChange = useCallback(
    (nextOffset: number) => {
      const next = new URLSearchParams(searchParams);
      if (nextOffset <= 0) next.delete('offset');
      else next.set('offset', String(nextOffset));
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  return (
    <PublisherHub
      activeTab={activeTab}
      onTabChange={onTabChange}
      from={from}
      to={to}
      onRangeChange={onRangeChange}
      dashboard={dashboard}
      statements={statements}
      dashboardLoading={dashboardLoading}
      statementsLoading={statementsLoading}
      dashboardError={dashboardError}
      statementsError={statementsError}
      limit={STATEMENT_LIMIT}
      offset={offset}
      onOffsetChange={onOffsetChange}
    />
  );
}
