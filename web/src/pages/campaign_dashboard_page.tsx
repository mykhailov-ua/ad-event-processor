import { CampaignDashboardView } from '@/domains/campaigns/report/campaign_dashboard_view';
import { useCampaignDashboardPageWorkspace } from '@/domains/campaigns/report/use_campaign_dashboard_page_workspace';

export function CampaignDashboardPage() {
  return <CampaignDashboardView {...useCampaignDashboardPageWorkspace()} />;
}
