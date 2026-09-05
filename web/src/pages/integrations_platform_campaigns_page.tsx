import { IntegrationsPlatformCampaigns } from '@/domains/integrations/integrations_platform_campaigns';
import { useIntegrationsPlatformCampaignsPage } from '@/domains/integrations/use_integrations_platform_campaigns_page';

export function IntegrationsPlatformCampaignsPage() {
  return <IntegrationsPlatformCampaigns {...useIntegrationsPlatformCampaignsPage()} />;
}
