import { Link } from 'react-router-dom';

import { PageChrome } from '@/shell/page_chrome';
import { PageSkeleton } from '@/shell/page_skeleton';
import type { PublisherDashboard } from '@/api/types';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { PortalsNav, portalsPanelError } from '@/domains/portals/portals_nav';

export type PublisherDashboardPanelProps = {
  dashboard: PublisherDashboard | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

export function PublisherDashboardPanel({
  dashboard,
  fetching,
  error,
  hasSnapshot,
}: PublisherDashboardPanelProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Publisher dashboard">
        <PortalsNav />
        {portalsPanelError(error, 'Could not load publisher dashboard')}
      </PageChrome>
    );
  }

  return (
    <PageChrome title="Publisher dashboard">
      <PortalsNav />
      <Link className="text-sm text-muted-foreground hover:underline" to="/publisher/statements">
        Statements
      </Link>

      {dashboard ? (
        <JsonDashboardView payload={dashboard as unknown as Record<string, unknown>} />
      ) : null}

      {error && hasSnapshot ? portalsPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
