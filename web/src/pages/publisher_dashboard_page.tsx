import { getPublisherDashboard } from '@/api/publisher_api';
import { PublisherDashboardPanel } from '@/domains/portals/publisher_dashboard_panel';
import { useResource } from '@/api/use_resource';

export function PublisherDashboardPage() {
  const { data, error, fetching } = useResource((signal) => getPublisherDashboard({}, signal), []);

  return (
    <PublisherDashboardPanel
      dashboard={data}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null || Boolean(error)}
    />
  );
}
