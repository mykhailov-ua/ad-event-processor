import { CampaignForecastPanel } from '@/domains/portals/campaign_forecast_panel';
import { useCampaignForecastPageWorkspace } from '@/domains/portals/use_campaign_forecast_page_workspace';

export function CampaignForecastPage() {
  return <CampaignForecastPanel {...useCampaignForecastPageWorkspace()} />;
}
