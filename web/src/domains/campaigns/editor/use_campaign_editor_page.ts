// L3 campaign editor page: composes load + form draft + actions; effectiveForm falls back to campaignToFormState when form unset.
import { campaignToFormState } from '@/domains/campaigns/editor/campaign_editor';
import type { CampaignEditorProps } from '@/domains/campaigns/editor/campaign_editor_types';
import { useCampaignEditorActions } from '@/domains/campaigns/editor/use_campaign_editor_actions';
import {
  CAMPAIGN_EDITOR_EMPTY_FORM,
  useCampaignEditorForm,
} from '@/domains/campaigns/editor/use_campaign_editor_form';
import { useCampaignEditorLoad } from '@/domains/campaigns/editor/use_campaign_editor_load';
import { useBreadcrumbSegmentLabel } from '@/shell/breadcrumb_context';

export function useCampaignEditorPage(): CampaignEditorProps {
  const { form, onFieldChange, syncFormFromCampaign } = useCampaignEditorForm();

  const load = useCampaignEditorLoad({ syncFormFromCampaign });

  const actions = useCampaignEditorActions({
    id: load.id,
    form,
    campaignSnapshot: load.campaignSnapshot,
    setCampaignSnapshot: load.setCampaignSnapshot,
    syncFormFromCampaign,
    setChecking: load.setChecking,
    setPublishCheck: load.setPublishCheck,
    setPublishCheckError: load.setPublishCheckError,
  });

  const effectiveForm = form ?? (load.campaign ? campaignToFormState(load.campaign) : undefined);

  useBreadcrumbSegmentLabel(load.id, load.campaign?.name);

  return {
    campaign: load.campaign,
    flowPaths: load.flowPaths,
    form: effectiveForm ?? CAMPAIGN_EDITOR_EMPTY_FORM,
    fetching: load.fetching,
    loadError: load.loadError,
    hasSnapshot: load.hasSnapshot,
    checking: load.checking,
    publishCheck: load.publishCheck,
    publishCheckError: load.publishCheckError,
    onFieldChange,
    ...actions,
  };
}
