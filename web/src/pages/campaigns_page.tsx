import { CampaignsDirectory } from '@/domains/campaigns/list/campaigns_directory';
import { useCampaignsPage } from '@/pages/use_campaigns_page';

export function CampaignsPage() {
  const directoryProps = useCampaignsPage();
  return <CampaignsDirectory {...directoryProps} />;
}
