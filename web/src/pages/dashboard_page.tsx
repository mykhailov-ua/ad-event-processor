import { RoleDashboardView } from '@/domains/dashboards/role_dashboard_view';
import { useDashboardPage } from '@/domains/dashboards/use_dashboard_page';

export function DashboardPage() {
  const workspace = useDashboardPage();

  return <RoleDashboardView {...workspace} />;
}
