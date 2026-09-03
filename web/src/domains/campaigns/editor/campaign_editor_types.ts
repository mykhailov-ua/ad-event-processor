import type { FlowPath } from '@/api/types';
import type {
  CampaignDiffResponse,
  CloneCampaignOptions,
  CloneCampaignPreview,
  MacroPreviewResponse,
} from '@/api/campaigns_api';
import type { Campaign, CampaignPublishBlockedError, CampaignPublishCheck, CampaignValidateResponse, PatchCampaignRequest } from '@/api/types';

export type CampaignEditorFormState = {
  name: string;
  status: string;
  budget_limit: string;
  pacing_mode: string;
  flow_id: string;
  brand_id: string;
  ingress_param: string;
  ingress_scale: string;
  ingress_max_micro: string;
  ingress_policy: string;
  traffic_template_id: string;
  click_query_params_json: string;
};

export type BuildCampaignPatchResult =
  | { ok: true; body: PatchCampaignRequest }
  | { ok: false; error: string };

export type MacroPreviewFormState = {
  sub1: string;
  country: string;
  click_id: string;
};

export type CampaignEditorProps = {
  campaign: Campaign | undefined;
  flowPaths?: FlowPath[];
  form: CampaignEditorFormState;
  fetching: boolean;
  saving: boolean;
  loadError: Error | undefined;
  saveError: Error | undefined;
  hasSnapshot: boolean;
  onFieldChange: <K extends keyof CampaignEditorFormState>(
    field: K,
    value: CampaignEditorFormState[K],
  ) => void;
  onSave: () => void;
  onSaveAndClose?: () => void;
  onOpenClone?: () => void;
  clickUrl?: string;
  checking: boolean;
  validating: boolean;
  publishing: boolean;
  forcePublish: boolean;
  publishCheck: CampaignPublishCheck | undefined;
  validateResult: CampaignValidateResponse | undefined;
  publishBlocked: CampaignPublishBlockedError | undefined;
  publishSuccess: boolean;
  publishCheckError: Error | undefined;
  validateError: Error | undefined;
  publishError: Error | undefined;
  onForcePublishChange: (force: boolean) => void;
  onCheckPublish: () => void;
  onValidateChanges: () => void;
  onPublish: () => void;
  macroPreviewForm: MacroPreviewFormState;
  onMacroPreviewFieldChange: <K extends keyof MacroPreviewFormState>(
    field: K,
    value: MacroPreviewFormState[K],
  ) => void;
  macroPreviewing: boolean;
  macroPreviewResult: MacroPreviewResponse | undefined;
  macroPreviewError: Error | undefined;
  onMacroPreview: () => void;
  cloneNameSuffix: string;
  onCloneNameSuffixChange: (value: string) => void;
  cloneOptions: CloneCampaignOptions;
  onCloneOptionChange: (field: keyof CloneCampaignOptions, value: boolean) => void;
  clonePreviewing: boolean;
  clonePreview: CloneCampaignPreview | undefined;
  clonePreviewError: Error | undefined;
  onClonePreview: () => void;
  cloning: boolean;
  cloneSuccess: boolean;
  clonedCampaignId: string | undefined;
  cloneError: Error | undefined;
  onCloneExecute: () => void;
  diffAgainstId: string;
  onDiffAgainstIdChange: (value: string) => void;
  comparingDiff: boolean;
  diffResult: CampaignDiffResponse | undefined;
  diffError: Error | undefined;
  onCompareDiff: () => void;
  draftOwnerUserId: string;
  onDraftOwnerUserIdChange: (value: string) => void;
  transferringOwner: boolean;
  ownerError: Error | undefined;
  ownerSuccess: boolean;
  onTransferOwner: () => void;
  exporting: boolean;
  exportError: Error | undefined;
  onExportCampaign: () => void;
};

export type CampaignDisplayFields = Campaign & {
  budget_limit_display?: string;
  current_spend_display?: string;
  daily_budget_display?: string;
  status_label?: string;
};
