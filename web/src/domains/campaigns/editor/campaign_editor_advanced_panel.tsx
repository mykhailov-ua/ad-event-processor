import { CampaignEditorAdvancedCloneSheet } from '@/domains/campaigns/editor/campaign_editor_advanced_clone_sheet';
import { CampaignEditorAdvancedCompareSection } from '@/domains/campaigns/editor/campaign_editor_advanced_compare_section';
import { CampaignEditorAdvancedMacroSection } from '@/domains/campaigns/editor/campaign_editor_advanced_macro_section';
import { CampaignEditorAdvancedOwnerSection } from '@/domains/campaigns/editor/campaign_editor_advanced_owner_section';
import { CampaignEditorAdvancedPublishSection } from '@/domains/campaigns/editor/campaign_editor_advanced_publish_section';
import { CampaignEditorAdvancedRoutingSection } from '@/domains/campaigns/editor/campaign_editor_advanced_routing_section';
import { EDITOR_MAIN_COLUMN_CLASS } from '@/shell/filter_panel';
import { CampaignEditorTools } from '@/domains/campaigns/editor/campaign_editor_tools';
import { cn } from '@/lib/utils';
import type { Campaign } from '@/api/types';
import type { CampaignEditorProps } from '@/domains/campaigns/editor/campaign_editor_types';

export type CampaignEditorAdvancedPanelProps = Pick<
  CampaignEditorProps,
  | 'form'
  | 'saving'
  | 'fetching'
  | 'onFieldChange'
  | 'checking'
  | 'validating'
  | 'publishing'
  | 'forcePublish'
  | 'publishCheck'
  | 'validateResult'
  | 'publishBlocked'
  | 'publishSuccess'
  | 'publishCheckError'
  | 'validateError'
  | 'publishError'
  | 'onForcePublishChange'
  | 'onCheckPublish'
  | 'onValidateChanges'
  | 'onPublish'
  | 'macroPreviewForm'
  | 'onMacroPreviewFieldChange'
  | 'macroPreviewing'
  | 'macroPreviewResult'
  | 'macroPreviewError'
  | 'onMacroPreview'
  | 'cloneNameSuffix'
  | 'onCloneNameSuffixChange'
  | 'cloneOptions'
  | 'onCloneOptionChange'
  | 'clonePreviewing'
  | 'clonePreview'
  | 'clonePreviewError'
  | 'onClonePreview'
  | 'cloning'
  | 'cloneSuccess'
  | 'clonedCampaignId'
  | 'cloneError'
  | 'onCloneExecute'
  | 'diffAgainstId'
  | 'onDiffAgainstIdChange'
  | 'comparingDiff'
  | 'diffResult'
  | 'diffError'
  | 'onCompareDiff'
  | 'draftOwnerUserId'
  | 'onDraftOwnerUserIdChange'
  | 'transferringOwner'
  | 'ownerError'
  | 'ownerSuccess'
  | 'onTransferOwner'
  | 'exporting'
  | 'exportError'
  | 'onExportCampaign'
> & {
  campaign: Campaign;
  statusLabel: string;
  gateBusy: boolean;
  cloneOpen: boolean;
  onCloneOpenChange: (open: boolean) => void;
};

export function CampaignEditorAdvancedPanel({
  campaign,
  form,
  saving,
  fetching,
  onFieldChange,
  checking,
  validating,
  publishing,
  forcePublish,
  publishCheck,
  validateResult,
  publishBlocked,
  publishSuccess,
  publishCheckError,
  validateError,
  publishError,
  onForcePublishChange,
  onCheckPublish,
  onValidateChanges,
  onPublish,
  macroPreviewForm,
  onMacroPreviewFieldChange,
  macroPreviewing,
  macroPreviewResult,
  macroPreviewError,
  onMacroPreview,
  cloneNameSuffix,
  onCloneNameSuffixChange,
  cloneOptions,
  onCloneOptionChange,
  clonePreviewing,
  clonePreview,
  clonePreviewError,
  onClonePreview,
  cloning,
  cloneSuccess,
  clonedCampaignId,
  cloneError,
  onCloneExecute,
  diffAgainstId,
  onDiffAgainstIdChange,
  comparingDiff,
  diffResult,
  diffError,
  onCompareDiff,
  draftOwnerUserId,
  onDraftOwnerUserIdChange,
  transferringOwner,
  ownerError,
  ownerSuccess,
  onTransferOwner,
  exporting,
  exportError,
  onExportCampaign,
  statusLabel,
  gateBusy,
  cloneOpen,
  onCloneOpenChange,
}: CampaignEditorAdvancedPanelProps) {
  return (
    <div className={cn(EDITOR_MAIN_COLUMN_CLASS, 'gap-8')}>
      <p className="text-sm text-muted-foreground">
        Status: {statusLabel}
        {checking ? '  /  Checking publish...' : ''}
        {publishCheck && !checking
          ? publishCheck.valid
            ? '  /  Publish ready'
            : '  /  Publish blocked'
          : ''}
      </p>

      <CampaignEditorAdvancedRoutingSection
        form={form}
        saving={saving}
        onFieldChange={onFieldChange}
      />

      <CampaignEditorAdvancedMacroSection
        macroPreviewForm={macroPreviewForm}
        onMacroPreviewFieldChange={onMacroPreviewFieldChange}
        macroPreviewing={macroPreviewing}
        fetching={fetching}
        macroPreviewResult={macroPreviewResult}
        macroPreviewError={macroPreviewError}
        onMacroPreview={onMacroPreview}
      />

      <CampaignEditorAdvancedPublishSection
        checking={checking}
        validating={validating}
        publishing={publishing}
        forcePublish={forcePublish}
        gateBusy={gateBusy}
        fetching={fetching}
        saving={saving}
        publishCheck={publishCheck}
        validateResult={validateResult}
        publishBlocked={publishBlocked}
        publishSuccess={publishSuccess}
        publishCheckError={publishCheckError}
        validateError={validateError}
        publishError={publishError}
        onForcePublishChange={onForcePublishChange}
        onCheckPublish={onCheckPublish}
        onValidateChanges={onValidateChanges}
        onPublish={onPublish}
      />

      <CampaignEditorAdvancedCloneSheet
        campaign={campaign}
        cloneOpen={cloneOpen}
        onCloneOpenChange={onCloneOpenChange}
        cloneNameSuffix={cloneNameSuffix}
        onCloneNameSuffixChange={onCloneNameSuffixChange}
        cloneOptions={cloneOptions}
        onCloneOptionChange={onCloneOptionChange}
        clonePreviewing={clonePreviewing}
        cloning={cloning}
        fetching={fetching}
        clonePreview={clonePreview}
        clonePreviewError={clonePreviewError}
        cloneError={cloneError}
        cloneSuccess={cloneSuccess}
        clonedCampaignId={clonedCampaignId}
        onClonePreview={onClonePreview}
        onCloneExecute={onCloneExecute}
      />

      <CampaignEditorAdvancedCompareSection
        campaign={campaign}
        diffAgainstId={diffAgainstId}
        onDiffAgainstIdChange={onDiffAgainstIdChange}
        comparingDiff={comparingDiff}
        fetching={fetching}
        diffResult={diffResult}
        diffError={diffError}
        onCompareDiff={onCompareDiff}
      />

      <CampaignEditorAdvancedOwnerSection
        draftOwnerUserId={draftOwnerUserId}
        onDraftOwnerUserIdChange={onDraftOwnerUserIdChange}
        transferringOwner={transferringOwner}
        fetching={fetching}
        ownerError={ownerError}
        ownerSuccess={ownerSuccess}
        onTransferOwner={onTransferOwner}
        exporting={exporting}
        exportError={exportError}
        onExportCampaign={onExportCampaign}
      />

      <CampaignEditorTools campaignId={campaign.id} />
    </div>
  );
}
