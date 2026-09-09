import { useEffect, useState } from 'react';

import { CampaignEditorAdvancedPanel } from '@/domains/campaigns/editor/campaign_editor_advanced_panel';
import { CampaignEditorShell } from '@/domains/campaigns/editor/campaign_editor_shell';
import { EditorStatusBanners } from '@/domains/campaigns/editor/campaign_editor_shared';
import type {
  CampaignDisplayFields,
  CampaignEditorProps,
} from '@/domains/campaigns/editor/campaign_editor_types';
import { formatCampaignStatusLabel } from '@/lib/admin_typography';
import { EditorPageShell } from '@/shell/editor_page_shell';
import { panelError } from '@/shell/panel_error';

export type {
  BuildCampaignPatchResult,
  CampaignEditorFormState,
  CampaignEditorProps,
  MacroPreviewFormState,
} from '@/domains/campaigns/editor/campaign_editor_types';

export {
  buildCampaignPatchBody,
  campaignToFormState,
  parseClickQueryParamsJson,
} from '@/domains/campaigns/editor/campaign_editor_form';

export function CampaignEditor(props: CampaignEditorProps) {
  const {
    campaign,
    flowPaths,
    form,
    fetching,
    saving,
    loadError,
    saveError,
    hasSnapshot,
    onFieldChange,
    onSave,
    clickUrl,
    checking,
    validating,
    publishing,
    macroPreviewResult,
    publishCheckError,
    validateError,
    publishError,
    cloneSuccess,
  } = props;

  const [cloneOpen, setCloneOpen] = useState(false);

  useEffect(() => {
    if (cloneSuccess) {
      setCloneOpen(false);
    }
  }, [cloneSuccess]);

  const fetchState = {
    fetching,
    error: loadError,
    hasSnapshot,
  };

  return (
    <EditorPageShell blockingErrorTitle="Could not load campaign" fetchState={fetchState}>
      <CampaignEditorContent
        {...props}
        cloneOpen={cloneOpen}
        onCloneOpenChange={setCloneOpen}
        clickUrl={clickUrl ?? macroPreviewResult?.resolved_click_url}
      />
    </EditorPageShell>
  );
}

type CampaignEditorContentProps = CampaignEditorProps & {
  cloneOpen: boolean;
  onCloneOpenChange: (open: boolean) => void;
  clickUrl?: string;
};

function CampaignEditorContent(props: CampaignEditorContentProps) {
  const {
    campaign,
    flowPaths,
    form,
    saving,
    saveError,
    onFieldChange,
    onSave,
    clickUrl,
    checking,
    validating,
    publishing,
    publishCheckError,
    validateError,
    publishError,
    cloneOpen,
    onCloneOpenChange,
  } = props;
  if (!campaign) {
    return panelError(new Error('No campaign data returned.'), 'Campaign not found');
  }

  const displayCampaign = campaign as CampaignDisplayFields;
  const statusLabel = formatCampaignStatusLabel(
    displayCampaign.status,
    displayCampaign.status_label
  );
  const gateBusy = checking || validating || publishing;

  return (
    <CampaignEditorShell
      campaignId={campaign.id}
      campaignName={campaign.name}
      clickUrl={clickUrl}
      flowPaths={flowPaths}
      form={form}
      saving={saving}
      statusBanner={
        <EditorStatusBanners
          publishCheckError={publishCheckError}
          publishError={publishError}
          saveError={saveError}
          validateError={validateError}
        />
      }
      advancedPanel={
        <CampaignEditorAdvancedPanel
          {...props}
          campaign={campaign}
          gateBusy={gateBusy}
          statusLabel={statusLabel}
          cloneOpen={cloneOpen}
          onCloneOpenChange={onCloneOpenChange}
        />
      }
      onClone={() => onCloneOpenChange(true)}
      onFieldChange={onFieldChange}
      onSave={onSave}
    />
  );
}
