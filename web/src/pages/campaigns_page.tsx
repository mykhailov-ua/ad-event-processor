import { CampaignsDirectory } from '@/domains/campaigns/list/campaigns_directory';
import { useCampaignsPage } from '@/domains/campaigns/list/use_campaigns_page';

export function CampaignsPage() {
  const directoryProps = useCampaignsPage();
  return <CampaignsDirectory {...directoryProps} />;
}
