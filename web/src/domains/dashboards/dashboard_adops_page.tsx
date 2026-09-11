import { DashboardPageView } from '@/domains/dashboards/dashboard_page_view';
import { useDashboardPageWorkspace } from '@/domains/dashboards/use_dashboard_page_workspace';

export function DashboardAdopsPage() {
  const workspace = useDashboardPageWorkspace('adops');
  return (
    <DashboardPageView
      {...workspace}
      description="Team-level pacing and source quality signals for ad ops."
      title="Ad ops dashboard"
    />
  );
}
