import { PublisherDashboardPanel } from '@/domains/portals/publisher_dashboard_panel';
import { usePublisherDashboardPageWorkspace } from '@/domains/portals/use_publisher_dashboard_page_workspace';

export function PublisherDashboardPage() {
  return <PublisherDashboardPanel {...usePublisherDashboardPageWorkspace()} />;
}
