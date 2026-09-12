import { DashboardPageView } from '@/domains/dashboards/dashboard_page_view';
import { useDashboardPageWorkspace } from '@/domains/dashboards/use_dashboard_page_workspace';

export function DashboardBuyerPage() {
  const workspace = useDashboardPageWorkspace('buyer');
  return (
    <DashboardPageView
      {...workspace}
      description="Morning KPI for your portfolio. Export Hub holds full tabular reports. Append ?chart_mock=1 for labeled synthetic chart preview (buyer only)."
      title="Buyer dashboard"
    />
  );
}
