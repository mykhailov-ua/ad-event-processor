// publisher portal dashboard: scoped GET snapshot (supply:read:scoped).
import { getPublisherDashboard } from '@/api/publisher_api';
import { useResource } from '@/api/use_resource';

export function usePublisherDashboardPageWorkspace() {
  const { data, error, fetching } = useResource((signal) => getPublisherDashboard({}, signal), []);

  return {
    dashboard: data,
    fetching,
    error,
    hasSnapshot: data != null,
  };
}
