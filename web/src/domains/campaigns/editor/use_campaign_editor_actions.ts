// campaign editor actions: PATCH/publish/clone mutations; save errors via saveError (no toast on save; toast only after publish/clone 2xx).
import { useCallback, useRef, useState } from 'react';
import { toast } from 'sonner';

import {
  checkCampaignPublish,
  cloneCampaign,
  exportCampaign,
  getCampaignDiff,
  patchCampaign,
  previewCampaignClone,
  previewCampaignMacros,
  publishCampaign,
  putCampaignOwner,
  validateCampaignPatch,
  type CampaignDiffResponse,
  type CloneCampaignOptions,
  type CloneCampaignPreview,
  type MacroPreviewResponse,
} from '@/api/campaigns_api';
import { isAbortError } from '@/api/client';
import type {
  Campaign,
  CampaignPublishBlockedError,
  CampaignPublishCheck,
  CampaignValidateResponse,
} from '@/api/types';
import {
  buildCloneRequestBody,
  DEFAULT_CLONE_OPTIONS,
} from '@/domains/campaigns/editor/campaign_clone_request';
import {
  buildCampaignPatchBody,
  type CampaignEditorFormState,
  type MacroPreviewFormState,
} from '@/domains/campaigns/editor/campaign_editor';
import { toError } from '@/lib/admin_error';
import { validationError } from '@/lib/admin_validation_error';
import { newRandomUuid } from '@/lib/uuid';

export type UseCampaignEditorActionsArgs = {
  id: string | undefined;
  form: CampaignEditorFormState | undefined;
  campaignSnapshot: Campaign | undefined;
  setCampaignSnapshot: (campaign: Campaign | undefined) => void;
  syncFormFromCampaign: (campaign: Campaign) => void;
  setChecking: (value: boolean) => void;
  setPublishCheck: (value: CampaignPublishCheck | undefined) => void;
  setPublishCheckError: (value: Error | undefined) => void;
};

export function useCampaignEditorActions({
  id,
  form,
  campaignSnapshot,
  setCampaignSnapshot,
  syncFormFromCampaign,
  setChecking,
  setPublishCheck,
  setPublishCheckError,
}: UseCampaignEditorActionsArgs) {
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<Error | undefined>(undefined);
  const [validating, setValidating] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [forcePublish, setForcePublish] = useState(false);
  const [validateResult, setValidateResult] = useState<CampaignValidateResponse | undefined>(
    undefined
  );
  const [publishBlocked, setPublishBlocked] = useState<CampaignPublishBlockedError | undefined>(
    undefined
  );
  const [publishSuccess, setPublishSuccess] = useState(false);
  const [validateError, setValidateError] = useState<Error | undefined>(undefined);
  const [publishError, setPublishError] = useState<Error | undefined>(undefined);
  const [macroPreviewForm, setMacroPreviewForm] = useState<MacroPreviewFormState>({
    sub1: '',
    country: '',
    click_id: '',
  });
  const [macroPreviewing, setMacroPreviewing] = useState(false);
  const [macroPreviewResult, setMacroPreviewResult] = useState<MacroPreviewResponse | undefined>(
    undefined
  );
  const [macroPreviewError, setMacroPreviewError] = useState<Error | undefined>(undefined);
  const [cloneNameSuffix, setCloneNameSuffix] = useState('');
  const [cloneOptions, setCloneOptions] = useState<CloneCampaignOptions>(DEFAULT_CLONE_OPTIONS);
  const [clonePreviewing, setClonePreviewing] = useState(false);
  const [clonePreview, setClonePreview] = useState<CloneCampaignPreview | undefined>(undefined);
  const [clonePreviewError, setClonePreviewError] = useState<Error | undefined>(undefined);
  const [cloning, setCloning] = useState(false);
  const [cloneSuccess, setCloneSuccess] = useState(false);
  const [clonedCampaignId, setClonedCampaignId] = useState<string | undefined>(undefined);
  const [cloneError, setCloneError] = useState<Error | undefined>(undefined);
  const [diffAgainstId, setDiffAgainstId] = useState('');
  const [comparingDiff, setComparingDiff] = useState(false);
  const [diffResult, setDiffResult] = useState<CampaignDiffResponse | undefined>(undefined);
  const [diffError, setDiffError] = useState<Error | undefined>(undefined);
  const [draftOwnerUserId, setDraftOwnerUserId] = useState('');
  const [transferringOwner, setTransferringOwner] = useState(false);
  const [ownerError, setOwnerError] = useState<Error | undefined>(undefined);
  const [ownerSuccess, setOwnerSuccess] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [exportError, setExportError] = useState<Error | undefined>(undefined);
  const cloneIdempotencyKeyRef = useRef<string | null>(null);

  const onSave = useCallback(async () => {
    if (!id || !campaignSnapshot || !form) {
      return;
    }

    const patchResult = buildCampaignPatchBody(campaignSnapshot, form);
    if (!patchResult.ok) {
      setSaveError(validationError(patchResult.error));
      return;
    }
    if (Object.keys(patchResult.body).length === 0) {
      return;
    }

    setSaving(true);
    setSaveError(undefined);

    try {
      const updated = await patchCampaign(id, patchResult.body);
      setCampaignSnapshot(updated);
      syncFormFromCampaign(updated);
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setSaveError(toError(err));
    } finally {
      setSaving(false);
    }
  }, [campaignSnapshot, form, id, setCampaignSnapshot, syncFormFromCampaign]);

  const onCheckPublish = useCallback(async () => {
    if (!id) {
      return;
    }

    setChecking(true);
    setPublishCheckError(undefined);
    setPublishCheck(undefined);

    try {
      const result = await checkCampaignPublish(id);
      setPublishCheck(result);
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setPublishCheckError(toError(err));
    } finally {
      setChecking(false);
    }
  }, [id, setChecking, setPublishCheck, setPublishCheckError]);

  const onValidateChanges = useCallback(async () => {
    if (!id || !campaignSnapshot || !form) {
      return;
    }

    const patchResult = buildCampaignPatchBody(campaignSnapshot, form);
    if (!patchResult.ok) {
      setValidateError(validationError(patchResult.error));
      setValidateResult(undefined);
      return;
    }
    if (Object.keys(patchResult.body).length === 0) {
      setValidateError(validationError('No unsaved changes to validate.'));
      setValidateResult(undefined);
      return;
    }

    setValidating(true);
    setValidateError(undefined);
    setValidateResult(undefined);

    try {
      const result = await validateCampaignPatch(id, patchResult.body);
      setValidateResult(result);
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setValidateError(toError(err));
    } finally {
      setValidating(false);
    }
  }, [campaignSnapshot, form, id]);

  const onPublish = useCallback(async () => {
    if (!id) {
      return;
    }

    setPublishing(true);
    setPublishError(undefined);
    setPublishBlocked(undefined);
    setPublishSuccess(false);

    try {
      const result = await publishCampaign(id, { force: forcePublish });
      if (result.status === 'published') {
        setPublishSuccess(true);
        toast.success('Campaign published');
        setCampaignSnapshot(result.campaign);
        syncFormFromCampaign(result.campaign);
        return;
      }
      setPublishBlocked(result.error);
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setPublishError(toError(err));
    } finally {
      setPublishing(false);
    }
  }, [forcePublish, id, setCampaignSnapshot, syncFormFromCampaign]);

  const onMacroPreviewFieldChange = useCallback(
    <K extends keyof MacroPreviewFormState>(field: K, value: MacroPreviewFormState[K]) => {
      setMacroPreviewForm((prev) => ({ ...prev, [field]: value }));
    },
    []
  );

  const onMacroPreview = useCallback(async () => {
    if (!id) {
      return;
    }

    setMacroPreviewing(true);
    setMacroPreviewError(undefined);
    setMacroPreviewResult(undefined);

    const body: Parameters<typeof previewCampaignMacros>[1] = {};
    if (macroPreviewForm.sub1.trim() !== '') {
      body.sub1 = macroPreviewForm.sub1.trim();
    }
    if (macroPreviewForm.country.trim() !== '') {
      body.country = macroPreviewForm.country.trim();
    }
    if (macroPreviewForm.click_id.trim() !== '') {
      body.click_id = macroPreviewForm.click_id.trim();
    }

    try {
      const result = await previewCampaignMacros(id, body);
      setMacroPreviewResult(result);
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setMacroPreviewError(toError(err));
    } finally {
      setMacroPreviewing(false);
    }
  }, [id, macroPreviewForm.click_id, macroPreviewForm.country, macroPreviewForm.sub1]);

  const onClonePreview = useCallback(async () => {
    if (!id) {
      return;
    }

    setClonePreviewing(true);
    setClonePreviewError(undefined);
    setClonePreview(undefined);

    try {
      const result = await previewCampaignClone(id, buildCloneRequestBody(cloneNameSuffix, cloneOptions));
      setClonePreview(result);
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setClonePreviewError(toError(err));
    } finally {
      setClonePreviewing(false);
    }
  }, [cloneNameSuffix, cloneOptions, id]);

  const onCloneExecute = useCallback(async () => {
    if (!id) {
      return;
    }

    if (!cloneIdempotencyKeyRef.current) {
      cloneIdempotencyKeyRef.current = newRandomUuid();
    }

    setCloning(true);
    setCloneError(undefined);
    setCloneSuccess(false);
    setClonedCampaignId(undefined);

    try {
      const result = await cloneCampaign(id, buildCloneRequestBody(cloneNameSuffix, cloneOptions), {
        idempotencyKey: cloneIdempotencyKeyRef.current,
      });
      cloneIdempotencyKeyRef.current = null;
      setCloneSuccess(true);
      setClonedCampaignId(result.id);
      toast.success('Campaign clone created');
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setCloneError(toError(err));
    } finally {
      setCloning(false);
    }
  }, [cloneNameSuffix, cloneOptions, id]);

  const onCloneOptionChange = useCallback((field: keyof CloneCampaignOptions, value: boolean) => {
    setCloneOptions((prev) => ({ ...prev, [field]: value }));
  }, []);

  const onCompareDiff = useCallback(async () => {
    if (!id) {
      return;
    }

    const against = diffAgainstId.trim();
    if (against === '') {
      setDiffError(validationError('Enter a campaign id to compare against.', { field: 'diff_against_id' }));
      setDiffResult(undefined);
      return;
    }

    setComparingDiff(true);
    setDiffError(undefined);
    setDiffResult(undefined);

    try {
      const result = await getCampaignDiff(id, against);
      setDiffResult(result);
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setDiffError(toError(err));
    } finally {
      setComparingDiff(false);
    }
  }, [diffAgainstId, id]);

  const onTransferOwner = useCallback(async () => {
    if (!id) {
      return;
    }
    const userId = draftOwnerUserId.trim();
    if (!userId) {
      setOwnerError(validationError('Owner user ID is required.', { field: 'owner_user_id' }));
      return;
    }
    setTransferringOwner(true);
    setOwnerError(undefined);
    setOwnerSuccess(false);

    try {
      await putCampaignOwner(id, { user_id: userId });
      setOwnerSuccess(true);
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setOwnerError(toError(err));
    } finally {
      setTransferringOwner(false);
    }
  }, [draftOwnerUserId, id]);

  const onExportCampaign = useCallback(async () => {
    if (!id) {
      return;
    }
    setExporting(true);
    setExportError(undefined);

    try {
      const bundle = await exportCampaign(id);
      const blob = new Blob([JSON.stringify(bundle, null, 2)], {
        type: 'application/json',
      });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = `campaign-${id}.json`;
      anchor.click();
      URL.revokeObjectURL(url);
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setExportError(toError(err));
    } finally {
      setExporting(false);
    }
  }, [id]);

  return {
    saving,
    saveError,
    onSave,
    validating,
    publishing,
    forcePublish,
    validateResult,
    publishBlocked,
    publishSuccess,
    validateError,
    publishError,
    onForcePublishChange: setForcePublish,
    onCheckPublish,
    onValidateChanges,
    onPublish,
    macroPreviewForm,
    onMacroPreviewFieldChange,
    macroPreviewing,
    macroPreviewResult,
    macroPreviewError,
    onMacroPreview,
    clickUrl: macroPreviewResult?.resolved_click_url,
    cloneNameSuffix,
    onCloneNameSuffixChange: setCloneNameSuffix,
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
    onDiffAgainstIdChange: setDiffAgainstId,
    comparingDiff,
    diffResult,
    diffError,
    onCompareDiff,
    draftOwnerUserId,
    onDraftOwnerUserIdChange: setDraftOwnerUserId,
    transferringOwner,
    ownerError,
    ownerSuccess,
    onTransferOwner,
    exporting,
    exportError,
    onExportCampaign,
  };
}
