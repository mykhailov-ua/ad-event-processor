import { useEffect, useState } from 'react';

import { ApiError } from '@/api/client';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { StubBanner } from '@/shell/stub_banner';
import { formatCampaignStatusLabel } from '@/lib/admin_typography';
import { CampaignEditorAdvancedPanel } from '@/domains/campaigns/editor/campaign_editor_advanced_panel';
import { CampaignEditorShell } from '@/domains/campaigns/editor/campaign_editor_shell';
import { EditorStatusBanners } from '@/domains/campaigns/editor/campaign_editor_shared';
import type {
  CampaignDisplayFields,
  CampaignEditorProps,
} from '@/domains/campaigns/editor/campaign_editor_types';

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
    onSaveAndClose,
    clickUrl,
    checking,
    validating,
    publishing,
    publishCheck,
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

  if (fetching && !hasSnapshot && !loadError) {
    return <PageSkeleton />;
  }

  if (loadError && !hasSnapshot) {
    if (loadError instanceof ApiError && loadError.status === 501) {
      return (
        <StubBanner title="Campaign editor unavailable" message={loadError.message} />
      );
    }
    return <ErrorBlock title="Could not load campaign" message={loadError.message} />;
  }

  if (!campaign) {
    return <ErrorBlock title="Campaign not found" message="No campaign data returned." />;
  }

  const displayCampaign = campaign as CampaignDisplayFields;
  const statusLabel = formatCampaignStatusLabel(
    displayCampaign.status,
    displayCampaign.status_label,
  );
  const gateBusy = checking || validating || publishing;

  return (
    <CampaignEditorShell
      campaignId={campaign.id}
      campaignName={campaign.name}
      clickUrl={clickUrl ?? macroPreviewResult?.resolved_click_url}
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
          onCloneOpenChange={setCloneOpen}
        />
      }
      onClone={() => setCloneOpen(true)}
      onFieldChange={onFieldChange}
      onSave={onSave}
      onSaveAndClose={onSaveAndClose ?? onSave}
    />
  );
}
