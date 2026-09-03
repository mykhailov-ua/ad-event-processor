import { CampaignEditor } from '@/domains/campaigns/editor/campaign_editor';
import { useCampaignEditorPage } from '@/pages/use_campaign_editor_page';

export function CampaignEditorPage() {
  return <CampaignEditor {...useCampaignEditorPage()} />;
}
